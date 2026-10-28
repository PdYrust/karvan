package store

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// migrateSeededV11 is a database at version 11 with the sites of v9Seed,
// their domain rows and one certificate, migrated to the latest version.
func migrateSeededV11(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "k.db")+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrateFrom(ctx, db, upTo(t, 12)); err != nil {
		t.Fatal(err)
	}
	seed := append(slices.Clone(v9Seed),
		"INSERT INTO domains (name, created_at, updated_at) VALUES ('old.example.com', 1, 1), ('*.example.net', 2, 2)",
		"INSERT INTO certificates (id, name, cert_file, key_file, created_at, updated_at) VALUES (1, 'files', '/c/a.crt', '/c/a.key', 1, 1)",
		"UPDATE legacy_sites SET certificate_id = 1, cert_file = '', key_file = '' WHERE id = 1")
	for _, q := range seed {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	_, before := schemaAndCounts(t, db)
	if err := migrateFrom(ctx, db, upTo(t, 13)); err != nil {
		t.Fatal(err)
	}
	_, after := schemaAndCounts(t, db)
	maps.DeleteFunc(after, func(table string, _ int) bool { _, old := before[table]; return !old })
	if !maps.Equal(before, after) {
		t.Fatalf("row counts after 0012 = %v, want %v", after, before)
	}
	return db
}

// 0012 keeps every row and value: domains from before wait for the
// bridge's fill (cert_filled 0, no certificate), certificates get no
// origin, and nothing is pinned or ordered.
func TestDomainCertificatesKeepEveryRow(t *testing.T) {
	db := migrateSeededV11(t)
	got := strs(t, db, "SELECT name || '|' || COALESCE(certificate_id, 'null') || '|' || cert_filled || '|' || acme_method || '|' || "+
		"renew_at || '|' || renew_tries || '|' || renew_error || '|' || created_at FROM domains ORDER BY name")
	if want := []string{"*.example.net|null|0||0|0||2", "old.example.com|null|0||0|0||1"}; !slices.Equal(got, want) {
		t.Fatalf("domains = %v, want %v", got, want)
	}
	got = strs(t, db, "SELECT id || '|' || origin FROM certificates UNION ALL SELECT count(*) FROM domain_pins "+
		"UNION ALL SELECT count(*) FROM acme_accounts UNION ALL SELECT count(*) FROM acme_orders")
	if want := []string{"1|", "0", "0", "0"}; !slices.Equal(got, want) {
		t.Fatalf("certificates and new tables = %v, want %v", got, want)
	}
	got = strs(t, db, "SELECT id || '|' || COALESCE(certificate_id, 'null') || '|' || cert_file FROM legacy_sites ORDER BY id")
	if want := []string{"1|1|", "2|null|", "3|null|/c/a.crt", "4|null|", "5|null|/c/w.crt", "6|null|/c/a.crt"}; !slices.Equal(got, want) {
		t.Fatalf("sites = %v, want %v", got, want)
	}
	if _, err := db.Exec("INSERT INTO domains (name, created_at, updated_at) VALUES ('new.example.com', 3, 3)"); err != nil {
		t.Fatal(err)
	}
	if got := strs(t, db, "SELECT cert_filled FROM domains WHERE name = 'new.example.com'"); !slices.Equal(got, []string{"1"}) {
		t.Fatalf("a new domain cert_filled = %v, want 1", got)
	}
}

// The new columns and tables refuse what section 3 rules out, and one
// running order per domain is the index's rule (D69).
func TestDomainCertificatesChecks(t *testing.T) {
	db := migrateSeededV11(t)
	const order = "INSERT INTO acme_orders (domain_id, account_id, method, state, created_at, updated_at) VALUES (1, 1, 'http-01', '%s', 1, 1)"
	for _, q := range []string{
		"INSERT INTO acme_accounts (directory, key_sealed, created_at) VALUES ('https://ca.example/dir', NULL, 1)",
		fmt.Sprintf(order, "starting"),
		fmt.Sprintf(order, "failed"),
		fmt.Sprintf(order, "done"),
		"INSERT INTO domain_pins (port, domain_id, certificate_id) VALUES (443, 1, 1)",
		"UPDATE domains SET certificate_id = 1, acme_method = 'dns-01' WHERE id = 1",
		"UPDATE certificates SET origin = 'acme' WHERE id = 1",
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	for _, q := range []string{
		fmt.Sprintf(order, "validating"), // a second running order
		fmt.Sprintf(order, "paused"),
		"INSERT INTO acme_orders (domain_id, account_id, method, state, created_at, updated_at) VALUES (2, 1, 'tls-alpn-01', 'failed', 1, 1)",
		"INSERT INTO acme_orders (domain_id, account_id, method, state, error_detail, created_at, updated_at) VALUES (2, 1, 'dns-01', 'failed', '" +
			strings.Repeat("x", 301) + "', 1, 1)",
		"INSERT INTO acme_accounts (directory, created_at) VALUES ('https://ca.example/dir', 1)",
		"INSERT INTO domain_pins (port, domain_id, certificate_id) VALUES (443, 1, 1)",
		"INSERT INTO domain_pins (port, domain_id, certificate_id) VALUES (0, 2, 1)",
		"INSERT INTO domain_pins (port, domain_id, certificate_id) VALUES (443, 2, 99)",
		"UPDATE domains SET cert_filled = 2 WHERE id = 1",
		"UPDATE domains SET acme_method = 'tls-alpn-01' WHERE id = 1",
		"UPDATE domains SET certificate_id = 99 WHERE id = 1",
		"UPDATE certificates SET origin = 'other' WHERE id = 1",
	} {
		if _, err := db.Exec(q); err == nil {
			t.Errorf("accepted: %s", q)
		}
	}
	if _, err := db.Exec("DELETE FROM domains WHERE id = 1"); err != nil {
		t.Fatal(err)
	}
	if got := strs(t, db, "SELECT count(*) FROM domain_pins UNION ALL SELECT count(*) FROM acme_orders"); !slices.Equal(got, []string{"0", "0"}) {
		t.Fatalf("pins and orders of a deleted domain = %v, want none", got)
	}
}
