package store

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"testing"
)

// v9Seed is a database at version 9 with sites of every certificate kind:
// a file pair, a pasted certificate, two sites sharing one file pair,
// plain HTTP, a wildcard domain and no domains, each with a token.
var v9Seed = []string{
	"INSERT INTO panels (id, env_path, name, scanned_at) VALUES (1, '/p/.env', 'p', 0)",
	"INSERT INTO jobs (id, state, source_kind, source_origin, panel_id, safe_mode, created_at, updated_at) VALUES ('j', 'done', 'marzban', 'https://old.example', 1, 1, 1, 1)",
	`INSERT INTO ledger (id, kind, job_id, state, chunk, source_origin, source_key, source_created, source_username, dest, dest_username, dest_created, created_at, updated_at) VALUES
		(1, 'user', 'j', 'done', 1, 'https://old.example', 'u1', 1, 'u1', 'd', 'u1', 1, 1, 1)`,
	`INSERT INTO legacy_sites (id, name, panel_id, source_kind, source_url, old_path, domains, listen_port, cert_file, key_file, cert_pem, key_sealed, created_at, updated_at) VALUES
		(1, 'file', 1, 'marzban', 'https://old.example', 'a', '["old.example.com"]', 443, '/c/a.crt', '/c/a.key', '', NULL, 30, 30),
		(2, 'pasted', 1, 'marzban', 'https://old.example', 'b', '["paste.example.com"]', 443, '', '', 'PEM', X'00', 20, 20),
		(3, 'shared', 1, 'marzban', 'https://old.example', 'c', '["old.example.com","cdn.example.com"]', 443, '/c/a.crt', '/c/a.key', '', NULL, 10, 10),
		(4, 'plain', 1, 'marzban', 'https://old.example', 'd', '["plain.example.com"]', 8080, '', '', '', NULL, 40, 40),
		(5, 'wild', 1, 'marzban', 'https://old.example', 'e', '["*.example.net"]', 443, '/c/w.crt', '/c/w.key', '', NULL, 50, 50),
		(6, 'any', 1, 'marzban', 'https://old.example', 'f', '[]', 8443, '/c/a.crt', '/c/a.key', '', NULL, 60, 60)`,
	`INSERT INTO legacy_tokens (mac, site_id, ledger_id, issued_at, verified_at, checked_at) VALUES
		(X'01', 1, 1, 1, 1, 1), (X'02', 2, 1, 1, 1, 1), (X'03', 3, 1, 1, 1, 1), (X'04', 4, 1, 1, 1, 1), (X'05', 5, 1, 1, 1, 1), (X'06', 6, 1, 1, 1, 1)`,
	"INSERT INTO legacy_closed (ledger_id, reason, closed_at) VALUES (1, 'revoked', 1)",
}

func migrateSeededV9(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "k.db")+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrateFrom(ctx, db, upTo(t, 10)); err != nil {
		t.Fatal(err)
	}
	for _, q := range v9Seed {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	cols := columns(t, db)
	before := snapshot(t, db, cols, false)
	if err := migrateFrom(ctx, db, fs.FS(migrationFiles)); err != nil {
		t.Fatal(err)
	}
	if after := snapshot(t, db, cols, false); fmt.Sprint(after) != fmt.Sprint(before) {
		t.Fatalf("rows changed:\nbefore %v\nafter  %v", before, after)
	}
	return db
}

// 0010 gives every site domain one row, dated by the first site that
// used it, and changes no existing row.
func TestSubscriptionsDomainRows(t *testing.T) {
	db := migrateSeededV9(t)
	got := strs(t, db, "SELECT name || '|' || created_at || '|' || updated_at FROM domains ORDER BY name")
	want := []string{"*.example.net|50|50", "cdn.example.com|10|10", "old.example.com|10|10", "paste.example.com|20|20", "plain.example.com|40|40"}
	if !slices.Equal(got, want) {
		t.Fatalf("domains = %v, want %v", got, want)
	}
}

// Certificates stay on the sites until the bridge adopts them (D54): SQL
// cannot reach the sealing key, so 0010 adds no certificate row.
func TestSubscriptionsLeaveCertificatesToTheBridge(t *testing.T) {
	db := migrateSeededV9(t)
	if got := strs(t, db, "SELECT count(*) FROM certificates UNION ALL SELECT count(*) FROM link_redirects"); !slices.Equal(got, []string{"0", "0"}) {
		t.Fatalf("certificates, redirects = %v", got)
	}
	got := strs(t, db, "SELECT id || '|' || COALESCE(certificate_id, 'null') FROM legacy_sites ORDER BY id")
	if want := []string{"1|null", "2|null", "3|null", "4|null", "5|null", "6|null"}; !slices.Equal(got, want) {
		t.Fatalf("sites = %v", got)
	}
}

// The new tables refuse what section 3 rules out.
func TestSubscriptionsChecks(t *testing.T) {
	db := migrateSeededV9(t)
	for _, q := range []string{
		"INSERT INTO domains (name, created_at, updated_at) VALUES ('', 1, 1)",
		"INSERT INTO domains (name, created_at, updated_at) VALUES ('old.example.com', 1, 1)",
		"INSERT INTO certificates (name, created_at, updated_at) VALUES ('none', 1, 1)",
		"INSERT INTO certificates (name, cert_file, key_file, cert_pem, created_at, updated_at) VALUES ('both', '/a', '/b', 'PEM', 1, 1)",
		"INSERT INTO certificates (name, cert_file, created_at, updated_at) VALUES ('half', '/a', 1, 1)",
		"INSERT INTO certificates (name, cert_pem, created_at, updated_at) VALUES ('', 'PEM', 1, 1)",
		"UPDATE legacy_sites SET certificate_id = 99 WHERE id = 1",
		"INSERT INTO link_redirects (name, listen_port, from_path, match, to_url, created_at, updated_at) VALUES ('r', 0, '/a', 'exact', 'https://x', 1, 1)",
		"INSERT INTO link_redirects (name, listen_port, from_path, match, to_url, created_at, updated_at) VALUES ('r', 80, '/a', 'glob', 'https://x', 1, 1)",
	} {
		if _, err := db.Exec(q); err == nil {
			t.Errorf("accepted: %s", q)
		}
	}
	for _, q := range []string{
		"INSERT INTO certificates (id, name, cert_file, key_file, created_at, updated_at) VALUES (1, 'files', '/a', '/b', 1, 1)",
		"INSERT INTO certificates (name, cert_pem, key_sealed, created_at, updated_at) VALUES ('pasted', 'PEM', X'00', 1, 1)",
		"UPDATE legacy_sites SET certificate_id = 1 WHERE id = 1",
		"INSERT INTO link_redirects (name, domains, listen_port, from_path, match, to_url, certificate_id, created_at, updated_at) VALUES ('r', '[]', 80, '/a', 'prefix', 'https://x', NULL, 1, 1)",
	} {
		if _, err := db.Exec(q); err != nil {
			t.Errorf("%s: %v", q, err)
		}
	}
}
