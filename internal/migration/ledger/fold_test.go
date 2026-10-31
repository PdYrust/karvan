package ledger

import (
	"context"
	"net/url"
	"slices"
	"testing"
)

// Every way of typing one old panel's address gives one name.
func TestOriginFoldsOneAddress(t *testing.T) {
	for raw, want := range map[string]string{
		"https://Old.Example.com":            "https://old.example.com",
		"https://old.example.com:443":        "https://old.example.com",
		"https://old.example.com:0443":       "https://old.example.com",
		"https://old.example.com.":           "https://old.example.com",
		"https://OLD.example.com.:443/":      "https://old.example.com",
		"https://old.example.com//Panel//":   "https://old.example.com/panel",
		"https://old.example.com/a//b":       "https://old.example.com/a/b",
		"http://old.example.com:80":          "http://old.example.com",
		"http://old.example.com:08000":       "http://old.example.com:8000",
		"http://old.example.com:8000/":       "http://old.example.com:8000",
		"https://203.0.113.7:2053/dash":      "https://203.0.113.7:2053/dash",
		"http://old.example.com:443":         "http://old.example.com:443",
		"https://old.example.com:80/x":       "https://old.example.com:80/x",
		"https://xn--mgbh0fb.example:8443/p": "https://xn--mgbh0fb.example:8443/p",
	} {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		if got := Origin(u); got != want {
			t.Errorf("Origin(%q) = %q, want %q", raw, got, want)
		}
	}
}

// foldRows adds a job, a user row, a batch and a rule of batches under
// origin, the way a build before the folds stored them.
func foldRows(t *testing.T, l *Ledger, n int, origin, key string) {
	t.Helper()
	job := "job-" + key
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{"INSERT OR IGNORE INTO panels (id, env_path, name, kind, scanned_at) VALUES (1, '/opt/p/.env', 'p', 'marzban', 0)", nil},
		{"INSERT INTO jobs (id, state, source_kind, source_origin, panel_id, safe_mode, created_at, updated_at) " +
			"VALUES (?, 'done', 'marzban', ?, 1, 1, 1, 1)", []any{job, origin}},
		{"INSERT INTO batches (job_id, panel_id, dest, source_kind, source_origin, name, state, created_at, updated_at) " +
			"VALUES (?, 1, 'sqlite:///var/lib/marzban/db.sqlite3', 'marzban', ?, ?, 'active', 1, 1)", []any{job, origin, key}},
		{"INSERT INTO legacy_sites (name, panel_id, source_kind, source_url, old_path, listen_port, mode, created_at, updated_at) " +
			"VALUES (?, 1, 'marzban', ?, 'sub', ?, 'batches', 1, 1)", []any{"rule " + key, origin, 9000 + n}},
	} {
		if _, err := l.db.Exec(q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	e := entry(job, key, Done)
	e.SourceOrigin = origin
	if _, err := l.Add(context.Background(), []Entry{e}); err != nil {
		t.Fatal(err)
	}
}

func (l *Ledger) column(t *testing.T, q string) []string {
	t.Helper()
	rows, err := l.db.Query(q)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// A name stored before the folds is renamed everywhere it is the old
// panel's name, so later runs of that panel match it; a run again
// changes nothing.
func TestFoldOriginsRenamesOldNames(t *testing.T) {
	ctx := context.Background()
	l := open(t)
	foldRows(t, l, 1, "https://old.example.com.", "ali")
	foldRows(t, l, 2, "https://old.example.com:0443//panel", "bob")
	foldRows(t, l, 3, "https://other.example.com", "cyrus")
	if _, err := l.db.Exec("INSERT INTO legacy_sites (name, panel_id, source_kind, source_url, old_path, listen_port, mode, " +
		"created_at, updated_at) VALUES ('typed', 1, 'marzban', 'https://old.example.com.', 'sub', 9100, 'manual', 1, 1)"); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		kept, err := FoldOrigins(ctx, l.db)
		if err != nil || len(kept) != 0 {
			t.Fatalf("kept %v, %v", kept, err)
		}
		want := []string{"https://old.example.com", "https://old.example.com/panel", "https://other.example.com"}
		for _, q := range []string{
			"SELECT DISTINCT source_origin FROM jobs ORDER BY 1",
			"SELECT DISTINCT source_origin FROM ledger ORDER BY 1",
			"SELECT DISTINCT source_origin FROM batches ORDER BY 1",
			"SELECT source_url FROM legacy_sites WHERE mode = 'batches' ORDER BY 1",
		} {
			if got := l.column(t, q); !slices.Equal(got, want) {
				t.Fatalf("%s: %v", q, got)
			}
		}
		if got := l.column(t, "SELECT source_url FROM legacy_sites WHERE mode = 'manual'"); !slices.Equal(got, []string{"https://old.example.com."}) {
			t.Fatalf("a manual rule keeps the address as typed: %v", got)
		}
	}
}

// A name whose live rows would clash with rows already under the new
// name keeps its spelling and is reported; the others are renamed.
func TestFoldOriginsKeepsAClash(t *testing.T) {
	ctx := context.Background()
	l := open(t)
	foldRows(t, l, 1, "https://old.example.com", "ali")
	foldRows(t, l, 2, "https://old.example.com.", "ali-again")
	if _, err := l.db.Exec("UPDATE ledger SET source_key = 'ali' WHERE source_origin = 'https://old.example.com.'"); err != nil {
		t.Fatal(err)
	}
	foldRows(t, l, 3, "https://other.example.com.", "bob")
	kept, err := FoldOrigins(ctx, l.db)
	if err != nil || !slices.Equal(kept, []string{"https://old.example.com."}) {
		t.Fatalf("kept %v, %v", kept, err)
	}
	want := []string{"https://old.example.com", "https://old.example.com.", "https://other.example.com"}
	if got := l.column(t, "SELECT DISTINCT source_origin FROM ledger ORDER BY 1"); !slices.Equal(got, want) {
		t.Fatalf("ledger: %v", got)
	}
	if got := l.column(t, "SELECT DISTINCT source_origin FROM jobs ORDER BY 1"); !slices.Equal(got, want) {
		t.Fatalf("jobs: %v", got)
	}
}
