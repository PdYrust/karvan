package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// settingsAfter14 is the settings table after 0014 on a database at
// version 13 with rows set, each key=value.
func settingsAfter14(t *testing.T, rows map[string]string) []string {
	t.Helper()
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "k.db")+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrateFrom(ctx, db, upTo(t, 14)); err != nil {
		t.Fatal(err)
	}
	for k, v := range rows {
		if _, err := db.ExecContext(ctx, "INSERT INTO settings (key, value) VALUES (?, ?)", k, v); err != nil {
			t.Fatal(err)
		}
	}
	if err := migrateFrom(ctx, db, migrationFiles); err != nil {
		t.Fatal(err)
	}
	return strs(t, db, "SELECT key || '=' || value FROM settings WHERE key LIKE 'http.%' ORDER BY key")
}

// installedHTTP is the address every installed panel has; 0014 leaves it alone.
var installedHTTP = map[string]string{"http.bind": "", "http.port": "41000", "http.base_path": "hiddenPath234567",
	"http.tls_self_cert": "pem", "http.tls_self_key": "sealed"}

func withHTTP(extra map[string]string) map[string]string {
	out := map[string]string{}
	for _, m := range []map[string]string{installedHTTP, extra} {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// Domain mode keeps its domain enabled, and a shared port with it.
func TestDomainModeKeepsItsDomain(t *testing.T) {
	for _, shared := range []string{"false", "true"} {
		got := settingsAfter14(t, withHTTP(map[string]string{"http.tls": "domain", "http.tls_domain": "3", "http.tls_shared": shared}))
		want := []string{"http.base_path=hiddenPath234567", "http.bind=", "http.port=41000", "http.tls_domain=3",
			"http.tls_self_cert=pem", "http.tls_self_key=sealed", "http.tls_shared=" + shared}
		if !slices.Equal(got, want) {
			t.Fatalf("shared %s: settings = %v, want %v", shared, got, want)
		}
	}
}

// Self mode enables no domain; a domain left from before is dropped.
func TestSelfModeEnablesNoDomain(t *testing.T) {
	got := settingsAfter14(t, withHTTP(map[string]string{"http.tls": "self", "http.tls_domain": "0", "http.tls_shared": "false"}))
	want := []string{"http.base_path=hiddenPath234567", "http.bind=", "http.port=41000", "http.tls_domain=0",
		"http.tls_self_cert=pem", "http.tls_self_key=sealed", "http.tls_shared=false"}
	if !slices.Equal(got, want) {
		t.Fatalf("settings = %v, want %v", got, want)
	}
}

// Off, written or from before HTTPS existed, enables no domain and takes
// plain HTTP for a Karvan domain without a certificate.
func TestOffModeTakesPlainHTTP(t *testing.T) {
	for name, rows := range map[string]map[string]string{
		"off":          withHTTP(map[string]string{"http.tls": "off", "http.tls_domain": "5", "http.tls_shared": "false"}),
		"before HTTPS": withHTTP(nil),
	} {
		got := settingsAfter14(t, rows)
		if !slices.Contains(got, "http.tls_plain=true") || slices.Contains(got, "http.tls_domain=5") ||
			slices.ContainsFunc(got, func(s string) bool { return strings.HasPrefix(s, "http.tls=") }) {
			t.Fatalf("%s: settings = %v", name, got)
		}
	}
}

// A database without the panel's address yet (before karvan init) gets
// no settings row.
func TestKarvanDomainMigrationAddsNothingBeforeInit(t *testing.T) {
	if got := settingsAfter14(t, nil); len(got) != 0 {
		t.Fatalf("settings = %v, want none", got)
	}
}
