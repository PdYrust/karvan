package bridge

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/PdYrust/karvan/internal/panel"
)

// Why a rule's batchIds were refused (section 4).
const (
	ReasonUnknown  = "unknown"  // no such batch, or a deleted one
	ReasonInactive = "inactive" // the batch is not active
	ReasonMixed    = "mixed"    // the batches do not share one old panel, destination and old path (D25)
)

// maxRuleBatches caps the batches of one rule.
const maxRuleBatches = 50

// ErrNotAdopted means the site still serves its own certificate, since its
// adoption failed, and the save picked none: a save must pick one (D54).
var ErrNotAdopted = errors.New("the site's certificate was not adopted; pick a certificate")

// batchesRefused is a batchIds refusal with its reason.
func batchesRefused(reason string) error { return &InvalidError{Field: "batchIds", Reason: reason} }

// ruleBatch is what a rule takes from one of its batches.
type ruleBatch struct {
	id, panelID                              int64
	state, dest, kind, origin, ip, pin, path string
	hosts                                    []string
	port                                     int
}

// checkMode checks the mode and batch list against the stored rule, if any.
func checkMode(s *Site, stored *Site) error {
	if s.Mode == "" {
		s.Mode = ModeManual
	}
	switch {
	case s.Mode != ModeManual && s.Mode != ModeBatches:
		return invalid("mode")
	case stored != nil && stored.Mode != s.Mode:
		return invalid("mode") // fixed after create
	case s.Mode == ModeManual && len(s.BatchIDs) > 0:
		return invalid("batchIds")
	}
	return nil
}

// fromBatches fills a rule of batches from them (D25, D56): every batch
// known and active, all of one old panel, destination and old path.
func (st *store) fromBatches(ctx context.Context, s *Site) error {
	ids := slices.Clone(s.BatchIDs)
	slices.Sort(ids)
	if len(ids) == 0 || len(ids) > maxRuleBatches || len(slices.Compact(slices.Clone(ids))) != len(ids) {
		return invalid("batchIds")
	}
	switch {
	case s.PanelID != 0:
		return invalid("panelId")
	case s.SourceKind != "":
		return invalid("sourceKind")
	case s.SourceURL != "":
		return invalid("sourceUrl")
	case s.SourcePin != "":
		return invalid("sourcePin")
	}
	bs, err := st.ruleBatches(ctx, ids)
	if err != nil {
		return err
	}
	if err := checkBatches(bs, len(ids)); err != nil {
		return err
	}
	s.BatchIDs = ids
	return s.takeFacts(bs)
}

// checkBatches checks the batches read for n ids.
func checkBatches(bs []ruleBatch, n int) error {
	if len(bs) != n {
		return batchesRefused(ReasonUnknown)
	}
	paths := map[string]bool{}
	for _, b := range bs {
		if b.state != "active" {
			return batchesRefused(ReasonInactive)
		}
		if b.path != "" {
			paths[b.path] = true
		}
	}
	for _, b := range bs {
		f := bs[0]
		if b.origin != f.origin || b.panelID != f.panelID || b.dest != f.dest || b.kind != f.kind || len(paths) > 1 {
			return batchesRefused(ReasonMixed)
		}
	}
	return nil
}

// takeFacts sets the derived fields from bs (sorted by id): the newest
// batch that knows a fact gives it, and the domains default to every old
// host. The old path and address come from the body only when no batch
// knows them.
func (s *Site) takeFacts(bs []ruleBatch) error {
	f := bs[0]
	s.PanelID, s.SourceKind, s.SourceURL = f.panelID, panel.Kind(f.kind), f.origin
	var path, ip, pin string
	port := 0
	var hosts []string
	for _, b := range bs {
		path = orPrev(b.path, path)
		ip = orPrev(b.ip, ip)
		pin = orPrev(b.pin, pin)
		if b.port != 0 {
			port = b.port
		}
		hosts = append(hosts, b.hosts...)
	}
	switch {
	case path != "" && s.OldPath != "":
		return invalid("oldPath")
	case ip != "" && s.SourceIP != "":
		return invalid("sourceIp")
	}
	if path != "" {
		s.OldPath = path
	}
	if ip != "" {
		s.SourceIP = ip
	}
	s.SourcePin = pin
	if s.Domains == nil {
		s.Domains = append([]string{}, hosts...)
	}
	if s.Port == 0 {
		s.Port = port
	}
	return nil
}

// orPrev is v when set, else prev.
func orPrev(v, prev string) string {
	if v != "" {
		return v
	}
	return prev
}

// ruleBatches reads the batches of ids that are not deleted, by id.
func (st *store) ruleBatches(ctx context.Context, ids []int64) ([]ruleBatch, error) {
	raw, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	rows, err := st.db.QueryContext(ctx, "SELECT id, panel_id, state, dest, source_kind, source_origin, source_ip, source_pin, "+
		"old_path, old_hosts, old_port FROM batches WHERE id IN (SELECT value FROM json_each(?)) AND state <> 'deleted' "+
		"ORDER BY id", string(raw))
	if err != nil {
		return nil, fmt.Errorf("read batches: %w", err)
	}
	defer rows.Close()
	var out []ruleBatch
	for rows.Next() {
		var b ruleBatch
		var hosts string
		if err := rows.Scan(&b.id, &b.panelID, &b.state, &b.dest, &b.kind, &b.origin, &b.ip, &b.pin, &b.path, &hosts,
			&b.port); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(hosts), &b.hosts); err != nil {
			return nil, fmt.Errorf("batch %d old hosts: %w", b.id, err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// linkBatches writes the batches of a rule of batches, inside the save. A
// batch that went inactive or was deleted since the check is refused.
func linkBatches(ctx context.Context, tx *sql.Tx, s Site) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM legacy_site_batches WHERE site_id = ?", s.ID); err != nil {
		return fmt.Errorf("unlink batches: %w", err)
	}
	if s.Mode != ModeBatches {
		return nil
	}
	raw, err := json.Marshal(s.BatchIDs)
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, "INSERT INTO legacy_site_batches (site_id, batch_id) SELECT ?, id FROM batches "+
		"WHERE id IN (SELECT value FROM json_each(?)) AND state = 'active'", s.ID, string(raw))
	if err != nil {
		return fmt.Errorf("link batches: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil || n != int64(len(s.BatchIDs)) {
		return errors.Join(err, batchesRefused(ReasonInactive))
	}
	return nil
}

// addDomains inserts the rule's domain names that have no row (D48).
func addDomains(ctx context.Context, tx *sql.Tx, domains []string, now int64) error {
	if len(domains) == 0 {
		return nil // json_each of null would give one NULL name
	}
	raw, err := json.Marshal(domains)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO domains (name, created_at, updated_at) SELECT value, ?, ? FROM json_each(?) "+
		"WHERE true ON CONFLICT (name) DO NOTHING", now, now, string(raw))
	if err != nil {
		return fmt.Errorf("add domains: %w", err)
	}
	return nil
}

// pickCertificate reads the certificate a save picks into s, so TLS() and
// the port checks see it; 0 is plain HTTP.
func (st *store) pickCertificate(ctx context.Context, s *Site) error {
	s.CertFile, s.KeyFile, s.CertPEM, s.KeyPEM, s.HasKey, s.certName = "", "", "", "", false, ""
	if s.CertificateID == 0 {
		return nil
	}
	r, err := scanCertRow(st.db.QueryRowContext(ctx, certRowColumns+" WHERE id = ?", s.CertificateID))
	if errors.Is(err, sql.ErrNoRows) {
		return invalid("certificateId")
	}
	if err != nil {
		return fmt.Errorf("read certificate: %w", err)
	}
	s.CertFile, s.KeyFile, s.CertPEM, s.HasKey, s.certName = r.certFile, r.keyFile, r.pem, r.sealed != nil, r.name
	return nil
}

// BatchRef is a batch a rule serves.
type BatchRef struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

// siteBatches lists the batches of rule id, by id.
func (st *store) siteBatches(ctx context.Context, id int64) ([]BatchRef, error) {
	rows, err := st.db.QueryContext(ctx, "SELECT b.id, b.name, b.state FROM legacy_site_batches sb "+
		"JOIN batches b ON b.id = sb.batch_id WHERE sb.site_id = ? ORDER BY b.id", id)
	if err != nil {
		return nil, fmt.Errorf("read rule batches: %w", err)
	}
	defer rows.Close()
	out := []BatchRef{}
	for rows.Next() {
		var b BatchRef
		if err := rows.Scan(&b.ID, &b.Name, &b.State); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
