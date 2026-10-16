package settings

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/PdYrust/karvan/internal/store"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	db, err := store.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return New(db)
}

func TestHTTPBeforeInitIsNotInitialized(t *testing.T) {
	_, err := newStore(t).HTTP(context.Background())
	if !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("err = %v, want ErrNotInitialized", err)
	}
}

func TestSetAndReadHTTP(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	want := HTTP{Bind: "", Port: 34567, BasePath: "abcDEF2345ghijkm"}
	if err := s.SetHTTP(ctx, want); err != nil {
		t.Fatal(err)
	}
	want.Port = 40000
	want.Bind = "127.0.0.1"
	if err := s.SetHTTP(ctx, want); err != nil {
		t.Fatal(err)
	}
	got, err := s.HTTP(ctx)
	if err != nil || got != want {
		t.Fatalf("HTTP = %+v, %v; want %+v", got, err, want)
	}
	if got.Addr() != "127.0.0.1:40000" {
		t.Errorf("Addr = %q", got.Addr())
	}
}

func TestHTTPRefusesADamagedStoredPath(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	if err := s.SetHTTP(ctx, HTTP{Port: 34567, BasePath: "abcDEF2345ghijkm"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`UPDATE settings SET value = '' WHERE key = ?`, keyBasePath); err != nil {
		t.Fatal(err)
	}
	if _, err := s.HTTP(ctx); err == nil {
		t.Fatal("an empty stored path was accepted")
	}
}

func TestSetHTTPRejectsInvalidValues(t *testing.T) {
	s := newStore(t)
	for name, h := range map[string]HTTP{
		"port zero":    {Port: 0, BasePath: "abcdefgh"},
		"port too big": {Port: 70000, BasePath: "abcdefgh"},
		"short path":   {Port: 3000, BasePath: "abc"},
		"slash path":   {Port: 3000, BasePath: "abc/defgh"},
		"bad bind":     {Bind: "example.com", Port: 3000, BasePath: "abcdefgh"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := s.SetHTTP(context.Background(), h); err == nil {
				t.Fatal("SetHTTP accepted an invalid value")
			}
		})
	}
}

func TestNewBasePathIsRandomAndValid(t *testing.T) {
	re := regexp.MustCompile(`^[a-zA-Z2-9]{16}$`)
	a, err := NewBasePath()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := NewBasePath()
	if !re.MatchString(a) || a == b {
		t.Fatalf("paths %q, %q", a, b)
	}
	if err := (HTTP{Port: 1, BasePath: a}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestRandomPortInRange(t *testing.T) {
	for range 200 {
		p, err := RandomPort()
		if err != nil || p < MinRandomPort || p > MaxRandomPort {
			t.Fatalf("port %d, %v", p, err)
		}
	}
}
