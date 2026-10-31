package bridge

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
)

// Where a certificate can be copied.
const (
	CopyLetsEncrypt = "letsencrypt" // /etc/letsencrypt/live/<domain>
	CopyPanel       = "panel"       // /var/lib/<panel folder>/certs/<domain>
)

// ErrCopyTarget means a copy target is neither of the two kinds.
var ErrCopyTarget = errors.New("unknown certificate copy target")

// Recorder writes audit events.
type Recorder interface {
	Record(ctx context.Context, e audit.Event) error
}

// CopyTarget is where a certificate is to be copied: Let's Encrypt live,
// or a registered panel's certificate folder.
type CopyTarget struct {
	Target  string `json:"target"`
	PanelID int64  `json:"panelId"`
}

// CopyPanelRef is the panel a copy is for.
type CopyPanelRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// CertCopy is a folder Karvan keeps a certificate copied to. After each
// change of the certificate it copies it there again.
type CertCopy struct {
	ID       int64         `json:"id"`
	Target   string        `json:"target"`
	Panel    *CopyPanelRef `json:"panel"`
	Dir      string        `json:"dir"`
	CertFile string        `json:"certFile"`
	KeyFile  string        `json:"keyFile"`
	CopiedAt *time.Time    `json:"copiedAt"`
	Error    *CopyError    `json:"error"` // the last copy failed
	certID   int64
	certName string
	base     string
	rel      string
	sums     [2]string
}

const copyColumns = "SELECT k.id, k.certificate_id, c.name, COALESCE(k.panel_id, 0), COALESCE(p.name, ''), COALESCE(p.kind, ''), k.base, k.rel, " +
	"k.cert_sum, k.key_sum, k.error_reason, k.error_path, k.copied_at FROM certificate_copies k " +
	"JOIN certificates c ON c.id = k.certificate_id LEFT JOIN panels p ON p.id = k.panel_id"

func scanCopy(r interface{ Scan(...any) error }) (CertCopy, error) {
	var c CertCopy
	var panelID, copied int64
	var panelName, panelKind, reason, errPath string
	err := r.Scan(&c.ID, &c.certID, &c.certName, &panelID, &panelName, &panelKind, &c.base, &c.rel, &c.sums[0], &c.sums[1], &reason, &errPath,
		&copied)
	c.Target, c.Dir = CopyLetsEncrypt, path.Join(c.base, c.rel)
	if panelID != 0 {
		c.Target, c.Panel = CopyPanel, &CopyPanelRef{ID: panelID, Name: panelName, Kind: panelKind}
	}
	c.CertFile, c.KeyFile = path.Join(c.Dir, copyCertName), path.Join(c.Dir, copyKeyName)
	c.CopiedAt = unixOrNil(copied)
	if reason != "" {
		c.Error = &CopyError{Reason: reason, Path: errPath}
	}
	return c, err
}

// copies reads the copies of certificate id, or every copy for 0, by folder.
func (st *store) copies(ctx context.Context, id int64) ([]CertCopy, error) {
	out := []CertCopy{}
	err := st.each(ctx, copyColumns+" WHERE ? IN (0, k.certificate_id) ORDER BY k.base, k.rel", func(r *sql.Rows) error {
		c, err := scanCopy(r)
		out = append(out, c)
		return err
	}, id)
	if err != nil {
		return nil, fmt.Errorf("read certificate copies: %w", err)
	}
	return out, nil
}

// withCopies adds each certificate's copies, and whether its files are in
// a place the certificate scan reads.
func (b *Bridge) withCopies(ctx context.Context, certs []Certificate) error {
	copies, err := b.st.copies(ctx, 0)
	if err != nil {
		return err
	}
	byCert := map[int64][]CertCopy{}
	for _, c := range copies {
		byCert[c.certID] = append(byCert[c.certID], c)
	}
	places, err := b.scanPlaces(ctx)
	if err != nil {
		return err
	}
	for i := range certs {
		c := &certs[i]
		c.Copies = append([]CertCopy{}, byCert[c.ID]...)
		c.Found = c.Kind == CertKindFile && slices.ContainsFunc(places, func(p scanPlace) bool { return within(c.CertFile, p.dir) })
	}
	return nil
}

// copySum is the keyed sum of data, so a stored sum says nothing about a key.
func (b *Bridge) copySum(data []byte) string {
	m := hmac.New(sha256.New, b.copyKey)
	m.Write(data)
	return hex.EncodeToString(m.Sum(nil))
}

// copyPlace is the base and folder of target for a certificate whose
// main domain is name, and the panel it is for.
func (b *Bridge) copyPlace(ctx context.Context, t CopyTarget, name string) (base, rel string, panel *Ref, err error) {
	switch t.Target {
	case CopyLetsEncrypt:
		return "/etc/letsencrypt", "live/" + name, nil, nil
	case CopyPanel:
	default:
		return "", "", nil, ErrCopyTarget
	}
	var p Ref
	var env string
	err = b.DB.QueryRowContext(ctx, "SELECT id, name, env_path FROM panels WHERE id = ?", t.PanelID).Scan(&p.ID, &p.Name, &env)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil, ErrNotFound
	}
	if err != nil {
		return "", "", nil, fmt.Errorf("read panel: %w", err)
	}
	folder := filepath.Base(filepath.Dir(env))
	if folder == "." || folder == ".." || folder == "/" || strings.ContainsAny(folder, "/\x00") {
		return "", "", nil, &CopyError{Reason: CopyOutside, Path: env}
	}
	return "/var/lib/" + folder, "certs/" + name, &p, nil
}

// CopyCertificate copies certificate id's fullchain.pem and privkey.pem
// into target's folder for its main domain, and remembers the folder so
// every later change of the certificate is copied there too. Karvan
// copies, never moves, and touches no other file there.
func (b *Bridge) CopyCertificate(ctx context.Context, id int64, t CopyTarget) (CertCopy, error) {
	b.copyMu.Lock()
	defer b.copyMu.Unlock()
	r, err := scanCertRow(b.DB.QueryRowContext(ctx, certRowColumns+" WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return CertCopy{}, ErrNotFound
	}
	if err != nil {
		return CertCopy{}, fmt.Errorf("read certificate: %w", err)
	}
	certPEM, keyPEM := b.material(r)
	leaf := firstCert(certPEM)
	if leaf == nil || len(keyPEM) == 0 {
		return CertCopy{}, &CopyError{Reason: CopyNoKey}
	}
	name := copyFolderName(certNames(leaf), leaf.Subject.CommonName)
	if name == "" {
		return CertCopy{}, &CopyError{Reason: CopyNoName}
	}
	base, rel, _, err := b.copyPlace(ctx, t, name)
	if err != nil {
		return CertCopy{}, err
	}
	var owner int64
	err = b.DB.QueryRowContext(ctx, "SELECT certificate_id FROM certificate_copies WHERE base = ? AND rel = ?", base, rel).Scan(&owner)
	switch {
	case err == nil && owner != id:
		return CertCopy{}, &CopyError{Reason: CopyTaken, Path: path.Join(base, rel)}
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		return CertCopy{}, fmt.Errorf("read certificate copies: %w", err)
	}
	if err := b.copyFiles(ctx, base, rel, certPEM, keyPEM); err != nil {
		return CertCopy{}, err
	}
	now := b.Now().Unix()
	_, err = b.DB.ExecContext(ctx, "INSERT INTO certificate_copies (certificate_id, panel_id, base, rel, cert_sum, key_sum, copied_at, created_at) "+
		"VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (base, rel) DO UPDATE SET cert_sum = excluded.cert_sum, key_sum = excluded.key_sum, "+
		"error_reason = '', error_path = '', copied_at = excluded.copied_at", id, nullID(t.PanelID), base, rel, b.copySum(certPEM),
		b.copySum(keyPEM), now, now)
	if err != nil {
		return CertCopy{}, fmt.Errorf("remember certificate copy: %w", err)
	}
	c, err := scanCopy(b.DB.QueryRowContext(ctx, copyColumns+" WHERE k.base = ? AND k.rel = ?", base, rel))
	if err != nil {
		return CertCopy{}, fmt.Errorf("read certificate copy: %w", err)
	}
	b.Log.Info("bridge: copied a certificate", "certificate", r.name, "dir", c.Dir)
	return c, nil
}

// copyFiles writes the two files and records what it wrote.
func (b *Bridge) copyFiles(ctx context.Context, base, rel string, certPEM, keyPEM []byte) error {
	paths := []string{path.Join(base, rel, copyCertName), path.Join(base, rel, copyKeyName)}
	sums := map[string]string{}
	for _, p := range paths {
		var s string
		err := b.DB.QueryRowContext(ctx, "SELECT sum FROM certificate_copy_files WHERE path = ?", p).Scan(&s)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("read written files: %w", err)
		}
		sums[p] = s
	}
	written, err := b.writeCopy(base, rel, []copyFile{
		{name: copyCertName, data: certPEM, mode: copyCertMode},
		{name: copyKeyName, data: keyPEM, mode: copyKeyMode},
	}, sums)
	if err != nil {
		return err
	}
	now := b.Now().Unix()
	for _, p := range paths {
		if _, err := b.DB.ExecContext(ctx, "INSERT INTO certificate_copy_files (path, sum, written_at) VALUES (?, ?, ?) "+
			"ON CONFLICT (path) DO UPDATE SET sum = excluded.sum, written_at = excluded.written_at", p, written[p], now); err != nil {
			return fmt.Errorf("record written file: %w", err)
		}
	}
	return nil
}

// StopCopy forgets copy copyID of certificate id. The files stay where
// they are.
func (b *Bridge) StopCopy(ctx context.Context, id, copyID int64) (CertCopy, error) {
	b.copyMu.Lock()
	defer b.copyMu.Unlock()
	c, err := scanCopy(b.DB.QueryRowContext(ctx, copyColumns+" WHERE k.id = ? AND k.certificate_id = ?", copyID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return CertCopy{}, ErrNotFound
	}
	if err != nil {
		return CertCopy{}, fmt.Errorf("read certificate copy: %w", err)
	}
	if _, err := b.DB.ExecContext(ctx, "DELETE FROM certificate_copies WHERE id = ?", copyID); err != nil {
		return CertCopy{}, fmt.Errorf("stop certificate copy: %w", err)
	}
	return c, nil
}

// recopy copies every certificate that changed since its last copy or
// try to its folders again, and logs and audits each copy. A failed copy
// is tried again at the certificate's next change.
func (b *Bridge) recopy(ctx context.Context) {
	b.copyMu.Lock()
	defer b.copyMu.Unlock()
	list, err := b.st.copies(ctx, 0)
	if err != nil {
		b.Log.Warn("bridge: read certificate copies", "err", err)
		return
	}
	for _, c := range list {
		r, err := scanCertRow(b.DB.QueryRowContext(ctx, certRowColumns+" WHERE id = ?", c.certID))
		if err != nil {
			b.Log.Warn("bridge: read certificate", "certificate", c.certName, "err", err)
			continue
		}
		certPEM, keyPEM := b.material(r)
		sums := [2]string{b.copySum(certPEM), b.copySum(keyPEM)}
		if sums == c.sums {
			continue
		}
		err = &CopyError{Reason: CopyNoKey}
		if firstCert(certPEM) != nil && len(keyPEM) > 0 {
			err = b.copyFiles(ctx, c.base, c.rel, certPEM, keyPEM)
		}
		b.copied(ctx, c, sums, err)
	}
}

// copied stores how copy c went with the material of sums, and logs and
// audits it.
func (b *Bridge) copied(ctx context.Context, c CertCopy, sums [2]string, err error) {
	reason, at := "", b.Now().Unix()
	detail := map[string]string{"certificate": c.certName, "dir": c.Dir}
	outcome := audit.Success
	var ce *CopyError
	if err != nil {
		reason, at, outcome = CopyFailed, c.unixCopied(), audit.Failure
		detail["reason"] = CopyFailed
		if errors.As(err, &ce) {
			reason, detail["reason"] = ce.Reason, ce.Reason
		}
		b.Log.Warn("bridge: copy a certificate again", "certificate", c.certName, "dir", c.Dir, "err", err)
	} else {
		b.Log.Info("bridge: copied a certificate again", "certificate", c.certName, "dir", c.Dir)
	}
	errPath := ""
	if ce != nil {
		errPath = ce.Path
	}
	if _, err := b.DB.ExecContext(ctx, "UPDATE certificate_copies SET cert_sum = ?, key_sum = ?, error_reason = ?, error_path = ?, "+
		"copied_at = ? WHERE id = ?", sums[0], sums[1], reason, errPath, at, c.ID); err != nil {
		b.Log.Warn("bridge: record certificate copy", "err", err)
	}
	if b.Audit == nil {
		return
	}
	if err := b.Audit.Record(ctx, audit.Event{Actor: audit.ActorSystem, Action: audit.ActionCertCopy, Outcome: outcome,
		Detail: detail}); err != nil {
		b.Log.Warn("bridge: audit certificate copy", "err", err)
	}
}

func (c CertCopy) unixCopied() int64 {
	if c.CopiedAt == nil {
		return 0
	}
	return c.CopiedAt.Unix()
}
