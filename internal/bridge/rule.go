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
	defaulted := s.Domains == nil
	if err := s.takeFacts(bs); err != nil || !defaulted {
		return err
	}
	return st.keepRegistered(ctx, s)
}

// keepRegistered keeps the domains of s that are registered: the batches'
// old hosts a rule of batches takes by default (D56).
func (st *store) keepRegistered(ctx context.Context, s *Site) error {
	raw, err := json.Marshal(s.Domains)
	if err != nil {
		return err
	}
	s.Domains = []string{}
	err = st.each(ctx, "SELECT name FROM domains WHERE name IN (SELECT value FROM json_each(?)) ORDER BY name", func(r *sql.Rows) error {
		var n string
		err := r.Scan(&n)
		s.Domains = append(s.Domains, n)
		return err
	}, string(raw))
	if err != nil {
		return fmt.Errorf("read registered domains: %w", err)
	}
	return nil
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

// Why a rule's domains or https were refused (D48, D63).
const (
	ReasonNotRegistered = "not_registered" // a domain that is not a domains row
	ReasonPending       = "pending"        // a domain whose Let's Encrypt order runs and that has no certificate yet
	ReasonNoCert        = "no_certificate" // HTTPS asked for while a domain, or the port, has no certificate
)

// ruleCert is what a rule save derives its certificate from (D63).
type ruleCert struct {
	self       RuleRef  // the rule; ID 0 on POST
	port       int      // as saved
	domains    []string // normalized, sorted
	https      *bool    // as sent; nil when absent
	stored     *bool    // the stored https on PATCH; nil on POST
	notAdopted bool     // the stored rule is served from its own certificate (D54)
}

// derive is the rule's certificate (D63): none for plain HTTP; with
// domains, for the first one, the port's pin of it, else its
// certificate; with no domains, the port's fallback. Every domain must be
// registered, and HTTPS needs a certificate for every domain, or the
// port's fallback without domains. Absent https is the stored value on
// PATCH, and on POST true only when those certificates are there.
func (st *store) derive(ctx context.Context, in ruleCert) (int64, error) {
	certs, lacking, err := st.ruleDomains(ctx, in.domains)
	if err != nil {
		return 0, err
	}
	var fallback int64
	if len(in.domains) == 0 {
		if fallback, err = st.portFallback(ctx, in.port, in.self); err != nil {
			return 0, err
		}
	}
	ready := len(lacking) == 0 && (len(in.domains) > 0 || fallback != 0)
	if in.notAdopted && !ready {
		return 0, ErrNotAdopted // its own certificate stays until its domains have certificates
	}
	switch {
	case !in.wantsHTTPS(ready):
		return 0, nil
	case len(in.domains) == 0 && fallback == 0:
		return 0, &InvalidError{Field: "https", Reason: ReasonNoCert}
	case len(in.domains) == 0:
		return fallback, nil
	case len(lacking) > 0:
		reason := ReasonPending
		if slices.ContainsFunc(lacking, func(d string) bool { return !certs[d].pending }) {
			reason = ReasonNoCert
		}
		return 0, &InvalidError{Field: "domains", Reason: reason, Names: lacking}
	}
	return domainsCert(ctx, st.db, in.port, in.domains)
}

// wantsHTTPS is https as sent, else the stored value on PATCH, else ready
// (every certificate the rule needs is there).
func (in ruleCert) wantsHTTPS(ready bool) bool {
	switch {
	case in.https != nil:
		return *in.https
	case in.stored != nil:
		return *in.stored
	}
	return ready
}

// domainState is a rule domain's certificate and whether a Let's Encrypt
// order for it runs.
type domainState struct {
	certID  int64
	pending bool
}

// ruleDomains reads the domains of a rule; every one must be registered
// (D48). lacking are those without a certificate, sorted.
func (st *store) ruleDomains(ctx context.Context, names []string) (map[string]domainState, []string, error) {
	out := map[string]domainState{}
	raw, err := json.Marshal(names)
	if err != nil {
		return nil, nil, err
	}
	rows, err := st.db.QueryContext(ctx, "SELECT name, COALESCE(certificate_id, 0), EXISTS (SELECT 1 FROM acme_orders o "+
		"WHERE o.domain_id = d.id AND o.state IN ('starting', 'waiting_dns', 'validating', 'issuing')) FROM domains d "+
		"WHERE name IN (SELECT value FROM json_each(?))", string(raw))
	if err != nil {
		return nil, nil, fmt.Errorf("read rule domains: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var d domainState
		if err := rows.Scan(&name, &d.certID, &d.pending); err != nil {
			return nil, nil, err
		}
		out[name] = d
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	var missing, lacking []string
	for _, n := range names {
		d, ok := out[n]
		switch {
		case !ok:
			missing = append(missing, n)
		case d.certID == 0:
			lacking = append(lacking, n)
		}
	}
	if len(missing) > 0 {
		return nil, nil, &InvalidError{Field: "domains", Reason: ReasonNotRegistered, Names: missing}
	}
	return out, lacking, nil
}

// portFallback is the certificate a handshake without SNI on port gets
// now (the D53 order with each rule's certificate). The rule being saved
// takes part as stored, and only while it has a certificate, so a save
// that changes nothing keeps it and a plain rule never hides the others.
func (st *store) portFallback(ctx context.Context, port int, self RuleRef) (int64, error) {
	all, err := st.rules(ctx)
	if err != nil {
		return 0, err
	}
	var routes []route
	for _, r := range all {
		if r.port == port && (r.ref.Kind != self.Kind || r.ref.ID != self.ID || r.certID != 0) {
			routes = append(routes, route{rule: r})
		}
	}
	pick, _ := d53(routes, "")
	return pick.certID, nil
}

// prunePins deletes the pins of (port, domain) pairs no rule holds any
// more (D62).
func prunePins(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "DELETE FROM domain_pins WHERE NOT EXISTS (SELECT 1 FROM domains d, legacy_sites s, "+
		"json_each(s.domains) j WHERE d.id = domain_pins.domain_id AND s.listen_port = domain_pins.port AND j.value = d.name) "+
		"AND NOT EXISTS (SELECT 1 FROM domains d, link_redirects r, json_each(r.domains) j WHERE d.id = domain_pins.domain_id "+
		"AND r.listen_port = domain_pins.port AND j.value = d.name)")
	if err != nil {
		return fmt.Errorf("delete pins: %w", err)
	}
	return nil
}

// pickCertificate reads the certificate a save derived into s, so TLS()
// and the port checks see it; 0 is plain HTTP.
func (st *store) pickCertificate(ctx context.Context, s *Site) error {
	s.CertFile, s.KeyFile, s.CertPEM, s.KeyPEM, s.HasKey, s.certName = "", "", "", "", false, ""
	if s.CertificateID == 0 {
		return nil
	}
	r, err := scanCertRow(st.db.QueryRowContext(ctx, certRowColumns+" WHERE id = ?", s.CertificateID))
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
