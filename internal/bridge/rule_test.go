package bridge

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/batch"
	"github.com/PdYrust/karvan/internal/panel"
)

// factsOf sets the old link facts of batch id.
func (e *env) factsOf(id int64, path, hosts string, port int, ip, pin string) {
	e.t.Helper()
	if _, err := e.b.DB.Exec("UPDATE batches SET old_path = ?, old_hosts = ?, old_port = ?, source_ip = ?, source_pin = ? WHERE id = ?",
		path, hosts, port, ip, pin, id); err != nil {
		e.t.Fatal(err)
	}
}

// fromBatches is a rule body made from batches, as the web panel sends it.
func fromBatches(name string, ids ...int64) Site {
	return Site{Name: name, Mode: ModeBatches, BatchIDs: ids, Lazy: true, PlainHTTPOK: true}
}

func (e *env) ruleBatchIDs(siteID int64) []int64 {
	e.t.Helper()
	rows, err := e.b.DB.Query("SELECT batch_id FROM legacy_site_batches WHERE site_id = ? ORDER BY batch_id", siteID)
	if err != nil {
		e.t.Fatal(err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			e.t.Fatal(err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		e.t.Fatal(err)
	}
	return out
}

func (e *env) domainNames() []string {
	e.t.Helper()
	rows, err := e.b.DB.Query("SELECT name FROM domains ORDER BY name")
	if err != nil {
		e.t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			e.t.Fatal(err)
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		e.t.Fatal(err)
	}
	return out
}

// A rule made from batches takes its destination, old panel, old path,
// address and pin from them, and its domains and port by default (D56).
func TestBatchRuleTakesItsFieldsFromTheBatches(t *testing.T) {
	e := newEnv(t, "ali", "bob")
	ctx := context.Background()
	a := e.batch("active", "", "ali")
	b := e.batch("active", "", "bob")
	e.factsOf(a, "sub", `["old.example.com"]`, 8443, "127.0.0.1", "")
	e.factsOf(b, "sub", `["cdn.example.com","old.example.com"]`, 2096, "127.0.0.2", strings.Repeat("ab", 32))
	s, err := e.b.st.Save(ctx, fromBatches("both", b, a))
	if err != nil {
		t.Fatal(err)
	}
	want := Site{PanelID: 1, SourceKind: panel.Marzban, SourceURL: e.site.origin, SourceIP: "127.0.0.2",
		SourcePin: strings.Repeat("ab", 32), OldPath: "sub", Domains: []string{"cdn.example.com", "old.example.com"}, Port: 2096}
	got := Site{PanelID: s.PanelID, SourceKind: s.SourceKind, SourceURL: s.SourceURL, SourceIP: s.SourceIP,
		SourcePin: s.SourcePin, OldPath: s.OldPath, Domains: s.Domains, Port: s.Port}
	if !reflect.DeepEqual(got, want) || s.Mode != ModeBatches || s.origin != e.site.origin {
		t.Fatalf("rule %+v, want %+v", s, want)
	}
	if ids := e.ruleBatchIDs(s.ID); !slices.Equal(ids, []int64{a, b}) {
		t.Fatalf("batches %v", ids)
	}
	if d := e.domainNames(); !slices.Equal(d, []string{"cdn.example.com", "old.example.com"}) {
		t.Fatalf("domain rows %v", d)
	}
	picked := fromBatches("picked", a)
	picked.Domains, picked.Port = []string{}, 9000
	if s, err := e.b.st.Save(ctx, picked); err != nil || len(s.Domains) != 0 || s.Port != 9000 {
		t.Fatalf("picked domains and port: %+v %v", s, err)
	}
}

// Batches that lack the old path or address take them from the body.
func TestBatchRuleAsksWhatTheBatchesLack(t *testing.T) {
	e := newEnv(t, "ali")
	ctx := context.Background()
	a := e.batch("active", "", "ali")
	s := fromBatches("asked", a)
	s.Port = 9000
	if _, err := e.b.st.Save(ctx, s); FieldOf(err) != "oldPath" {
		t.Fatalf("no old path: %v", err)
	}
	s.OldPath, s.SourceIP = "sub", "127.0.0.1"
	saved, err := e.b.st.Save(ctx, s)
	if err != nil || saved.OldPath != "sub" || saved.SourceIP != "127.0.0.1" {
		t.Fatalf("asked fields: %+v %v", saved, err)
	}
}

func reasonOf(err error) string {
	var e *InvalidError
	if errors.As(err, &e) {
		return e.Reason
	}
	return ""
}

// Every refused batch rule names its field, and its reason for batchIds.
func TestBatchRuleValidation(t *testing.T) {
	e := newEnv(t, "ali", "bob", "cyrus", "dara")
	ctx := context.Background()
	a := e.batch("active", "", "ali")
	e.factsOf(a, "sub", `[]`, 8443, "127.0.0.1", "")
	inactive := e.batch("inactive", "rolled_back", "bob")
	deleted := e.batch("deleted", "", "cyrus")
	other := e.batch("active", "", "dara")
	e.factsOf(other, "other", `[]`, 8443, "", "")
	otherPanel := e.batch("active", "")
	if _, err := e.b.DB.Exec("INSERT INTO panels (id, env_path, name, kind, scanned_at) VALUES (2, '/opt/b/.env', 'b', 'marzban', 0)"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.b.DB.Exec("UPDATE batches SET panel_id = 2 WHERE id = ?", otherPanel); err != nil {
		t.Fatal(err)
	}
	otherOrigin := e.batch("active", "")
	if _, err := e.b.DB.Exec("UPDATE batches SET source_origin = 'https://elsewhere.example.com' WHERE id = ?", otherOrigin); err != nil {
		t.Fatal(err)
	}
	many := make([]int64, 51)
	for i := range many {
		many[i] = int64(i + 1)
	}
	for name, c := range map[string]struct {
		edit          func(s *Site)
		field, reason string
	}{
		"no batches":      {func(s *Site) { s.BatchIDs = nil }, "batchIds", ""},
		"too many":        {func(s *Site) { s.BatchIDs = many }, "batchIds", ""},
		"twice":           {func(s *Site) { s.BatchIDs = []int64{a, a} }, "batchIds", ""},
		"unknown":         {func(s *Site) { s.BatchIDs = []int64{a, 999} }, "batchIds", "unknown"},
		"deleted":         {func(s *Site) { s.BatchIDs = []int64{a, deleted} }, "batchIds", "unknown"},
		"inactive":        {func(s *Site) { s.BatchIDs = []int64{a, inactive} }, "batchIds", "inactive"},
		"other path":      {func(s *Site) { s.BatchIDs = []int64{a, other} }, "batchIds", "mixed"},
		"other panel":     {func(s *Site) { s.BatchIDs = []int64{a, otherPanel} }, "batchIds", "mixed"},
		"other old panel": {func(s *Site) { s.BatchIDs = []int64{a, otherOrigin} }, "batchIds", "mixed"},
		"panelId":         {func(s *Site) { s.PanelID = 1 }, "panelId", ""},
		"sourceKind":      {func(s *Site) { s.SourceKind = panel.Marzban }, "sourceKind", ""},
		"sourceUrl":       {func(s *Site) { s.SourceURL = e.old.URL }, "sourceUrl", ""},
		"sourcePin":       {func(s *Site) { s.SourcePin = strings.Repeat("ab", 32) }, "sourcePin", ""},
		"known oldPath":   {func(s *Site) { s.OldPath = "sub" }, "oldPath", ""},
		"known sourceIp":  {func(s *Site) { s.SourceIP = "127.0.0.1" }, "sourceIp", ""},
		"mode":            {func(s *Site) { s.Mode = "both" }, "mode", ""},
		"manual batches":  {func(s *Site) { s.Mode = ModeManual }, "batchIds", ""},
		"certificate":     {func(s *Site) { s.CertificateID = 999 }, "certificateId", ""},
	} {
		s := fromBatches("refused", a)
		c.edit(&s)
		if s.Mode == ModeManual {
			s = e.site
			s.ID, s.Name, s.BatchIDs = 0, "refused", []int64{a}
		}
		_, err := e.b.st.Save(ctx, s)
		if !errors.Is(err, ErrInvalidSite) || FieldOf(err) != c.field || reasonOf(err) != c.reason {
			t.Errorf("%s: %v (reason %q)", name, err, reasonOf(err))
		}
	}
	if n := len(e.ruleBatchIDs(0)); n != 0 {
		t.Fatalf("links of a refused rule: %d", n)
	}
	var sites int
	_ = e.b.DB.QueryRow("SELECT count(*) FROM legacy_sites").Scan(&sites)
	if sites != 1 {
		t.Fatalf("sites after refused saves: %d", sites)
	}
}

// The mode is fixed after create; batches may change, and the derived
// columns are read again from the batches on every save.
func TestBatchRuleChanges(t *testing.T) {
	e := newEnv(t, "ali", "bob")
	ctx := context.Background()
	a := e.batch("active", "", "ali")
	b := e.batch("active", "", "bob")
	e.factsOf(a, "sub", `["old.example.com"]`, 9000, "127.0.0.1", "")
	e.factsOf(b, "sub", `["old.example.com"]`, 9000, "127.0.0.1", "")
	s, err := e.b.st.Save(ctx, fromBatches("rule", a))
	if err != nil {
		t.Fatal(err)
	}
	manual := s
	manual.Mode = ModeManual
	if _, err := e.b.st.Save(ctx, manual); FieldOf(err) != "mode" {
		t.Fatalf("mode change: %v", err)
	}
	if _, err := e.b.st.Save(ctx, Site{ID: e.site.ID, Name: "old", Mode: ModeBatches, BatchIDs: []int64{a}}); FieldOf(err) != "mode" {
		t.Fatalf("manual rule to batches: %v", err)
	}
	e.factsOf(b, "sub", `["old.example.com"]`, 9000, "127.0.0.9", "")
	edit := fromBatches("rule", b)
	edit.ID, edit.Domains, edit.Port = s.ID, s.Domains, s.Port
	got, err := e.b.st.Save(ctx, edit)
	if err != nil || !slices.Equal(e.ruleBatchIDs(s.ID), []int64{b}) || got.SourceIP != "127.0.0.9" {
		t.Fatalf("batches changed: %+v %v %v", got, err, e.ruleBatchIDs(s.ID))
	}
	if _, err := e.b.st.Save(ctx, Site{ID: 999, Name: "x", Mode: ModeBatches, BatchIDs: []int64{a}}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown rule: %v", err)
	}
}

// A token proven while its row was in the rule's scope stays stored when
// the row leaves the scope, but the rule no longer serves it (D23).
func TestTokenLeavingTheScopeStaysStored(t *testing.T) {
	e := newEnv(t, "ali", "bob")
	ctx := context.Background()
	a := e.batch("active", "", "ali")
	b := e.batch("active", "", "bob")
	e.factsOf(a, "sub", `["old.example.com"]`, 0, "127.0.0.1", "")
	e.factsOf(b, "sub", `["old.example.com"]`, 0, "127.0.0.1", "")
	s := fromBatches("rule", a)
	s.Port = 5
	saved, err := e.b.SaveSite(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(e.b.stopAll)
	e.port = 5
	tok := e.oldToken("ali", e.clock.now())
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusOK || e.tokens() != 1 {
		t.Fatalf("in scope: %d, %d tokens", code, e.tokens())
	}
	saved.BatchIDs = []int64{b}
	saved.PanelID, saved.SourceKind, saved.SourceURL, saved.SourceIP, saved.OldPath = 0, "", "", "", ""
	if _, err := e.b.SaveSite(ctx, saved); err != nil {
		t.Fatal(err)
	}
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusNotFound || e.tokens() != 1 {
		t.Fatalf("out of scope: %d, %d tokens", code, e.tokens())
	}
}

// A newer batch of the same old panel and destination joins the batch
// rules of the older one by itself, and its users are served (D26).
func TestNewerBatchJoinsTheRule(t *testing.T) {
	e := newEnv(t, "ali", "bob")
	ctx := context.Background()
	a := e.batch("active", "", "ali")
	e.factsOf(a, "sub", `["old.example.com"]`, 0, "127.0.0.1", "")
	s := fromBatches("rule", a)
	s.Port = 5
	saved, err := e.b.SaveSite(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(e.b.stopAll)
	e.port = 5
	tok := e.oldToken("bob", e.clock.now())
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusNotFound {
		t.Fatalf("bob before his batch: %d", code)
	}
	if _, err := e.b.DB.Exec("INSERT INTO jobs (id, state, source_kind, source_origin, panel_id, safe_mode, created_at, updated_at, "+
		"ended_at) VALUES ('job-new', 'done', 'marzban', ?, 1, 1, 1, 1, ?)", e.site.origin, e.clock.now().UnixMilli()); err != nil {
		t.Fatal(err)
	}
	if _, err := e.b.DB.Exec("UPDATE ledger SET job_id = 'job-new' WHERE id = ?", e.users["bob"].ID); err != nil {
		t.Fatal(err)
	}
	newer, err := batch.New(e.b.DB, e.clock.now).Ensure(ctx, "job-new", "fake-dest", batch.Facts{})
	if err != nil || newer == 0 {
		t.Fatalf("ensure: %d %v", newer, err)
	}
	if ids := e.ruleBatchIDs(saved.ID); !slices.Equal(ids, []int64{a, newer}) {
		t.Fatalf("rule batches %v", ids)
	}
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusOK {
		t.Fatalf("bob after his batch joined: %d", code)
	}
	v := e.b.View(ctx, mustSite(t, e, saved.ID))
	if want := []BatchRef{{ID: a, Name: "job-1", State: "active"}, {ID: newer, Name: "batch", State: "active"}}; !slices.Equal(v.Batches, want) ||
		!slices.Equal(v.BatchIDs, []int64{a, newer}) {
		t.Fatalf("view batches %+v %v", v.Batches, v.BatchIDs)
	}
}

// A manual rule reads and saves as before, and its domains get rows.
func TestManualRuleSavesAsBefore(t *testing.T) {
	e := newEnv(t, "ali")
	ctx := context.Background()
	before := mustSite(t, e, e.site.ID)
	if before.Mode != ModeManual || before.CertificateID != 0 || len(before.BatchIDs) != 0 {
		t.Fatalf("read %+v", before)
	}
	body := before
	body.Mode = "" // a body without a mode is manual
	after, err := e.b.st.Save(ctx, body)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("after %+v, before %+v", after, before)
	}
	if d := e.domainNames(); !slices.Equal(d, []string{"old.example.com"}) {
		t.Fatalf("domain rows %v", d)
	}
	body.Domains = nil // every host on the port, as before
	if _, err := e.b.st.Save(ctx, body); err != nil {
		t.Fatalf("no domains: %v", err)
	}
}

// ownColumns is a site's own certificate columns, as stored.
func (e *env) ownColumns(id int64) string {
	e.t.Helper()
	var certFile, keyFile, certPEM string
	var sealed []byte
	var certID sql.NullInt64
	if err := e.b.DB.QueryRow("SELECT cert_file, key_file, cert_pem, key_sealed, certificate_id FROM legacy_sites WHERE id = ?", id).
		Scan(&certFile, &keyFile, &certPEM, &sealed, &certID); err != nil {
		e.t.Fatal(err)
	}
	return fmt.Sprintf("%q %q %d %v %v", certFile, keyFile, len(certPEM), sealed != nil, certID)
}

// A site whose certificate was not adopted cannot be saved without a
// certificate; picking one clears its own columns in the same save, and
// the handshake then serves the picked certificate (D54).
func TestSiteNotAdoptedNeedsACertificate(t *testing.T) {
	e := newCertEnv(t)
	ctx := context.Background()
	t.Cleanup(e.b.stopAll)
	port := freePort(t)
	cert, key := longCert(t, "old.example.com")
	e.seedOldSite(oldSite{id: 7, name: "pasted", path: "tls", domains: `["old.example.com"]`, port: port, certPEM: cert,
		keyPEM: key, aadID: 99}) // sealed under another id, so it cannot be opened or adopted
	b := e.restart(slog.New(slog.DiscardHandler))
	before := handshake(t, port, "old.example.com")
	s, err := b.st.Site(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	columns := e.ownColumns(7)
	s.Name = "renamed"
	if _, err := b.SaveSite(ctx, s); !errors.Is(err, ErrNotAdopted) {
		t.Fatalf("save without a certificate: %v", err)
	}
	if got := e.ownColumns(7); got != columns || mustSite(t, e, 7).Name != "pasted" {
		t.Fatalf("refused save changed the row: %s", got)
	}
	other, otherKey := longCert(t, "old.example.com")
	replaced := e.addCert(CertificateInput{Name: "replaced", CertPEM: other, KeyPEM: otherKey})
	s.CertificateID = replaced.ID
	saved, err := b.SaveSite(ctx, s)
	if err != nil || saved.CertificateID != replaced.ID || saved.Name != "renamed" {
		t.Fatalf("save with a certificate: %+v %v", saved, err)
	}
	if got := e.ownColumns(7); got != fmt.Sprintf(`"" "" 0 false {%d true}`, replaced.ID) {
		t.Fatalf("own columns after the pick: %s", got)
	}
	if after := handshake(t, port, "old.example.com"); after == before || after == "error" {
		t.Fatalf("handshake after the pick: %s", after)
	}
	if v := b.View(ctx, saved); v.Certificate == nil || v.Certificate.ID != replaced.ID || v.Certificate.Name != "replaced" ||
		slices.Contains(v.Warnings, WarnNotAdopted) {
		t.Fatalf("view %+v", v)
	}
}

// A site served over TLS takes its certificate by id; the handshake
// serves it, and plain HTTP is a site without one.
func TestRuleCertificateByID(t *testing.T) {
	e := newCertEnv(t)
	ctx := context.Background()
	t.Cleanup(e.b.stopAll)
	cert, key := longCert(t, "old.example.com")
	c := e.addCert(CertificateInput{Name: "tls", CertPEM: cert, KeyPEM: key})
	s := e.site
	s.ID, s.Name, s.OldPath, s.Port, s.CertificateID = 0, "tls", "tls", freePort(t), c.ID
	saved, err := e.b.SaveSite(ctx, s)
	if err != nil || !saved.TLS() || saved.CertificateID != c.ID {
		t.Fatalf("save %+v %v", saved, err)
	}
	if got := e.ownColumns(saved.ID); got != fmt.Sprintf(`"" "" 0 false {%d true}`, c.ID) {
		t.Fatalf("own columns %s", got)
	}
	if h := handshake(t, saved.Port, "old.example.com"); h == "error" {
		t.Fatal("no handshake")
	}
	plain := e.site
	plain.ID, plain.Name, plain.OldPath, plain.Port = 0, "plain", "plain", saved.Port
	if _, err := e.b.SaveSite(ctx, plain); !errors.Is(err, ErrPortMixed) {
		t.Fatalf("plain on a TLS port: %v", err)
	}
	v := e.b.View(ctx, saved)
	if v.Certificate == nil || v.Certificate.Name != "tls" || v.Cert == nil || len(v.Cert.Warnings) != 0 {
		t.Fatalf("view %+v", v)
	}
}
