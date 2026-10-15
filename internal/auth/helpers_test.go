package auth

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/secret"
	"github.com/PdYrust/karvan/internal/store"
)

type fixture struct {
	db        *sql.DB
	key       *secret.Key
	operators *Operators
	clock     *fakeClock
}

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dir := t.TempDir()
	db, err := store.Open(context.Background(), filepath.Join(dir, "data"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	keyPath := filepath.Join(dir, "secret.key")
	if err := secret.CreateKeyFile(keyPath); err != nil {
		t.Fatal(err)
	}
	key, err := secret.LoadKeyFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	clock := &fakeClock{t: time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)}
	ops := NewOperators(db, key)
	ops.now = clock.Now
	return &fixture{db: db, key: key, operators: ops, clock: clock}
}

// Cheap hash so tests that only need some stored hash stay fast.
const testHash = "$argon2id$v=19$m=8,t=1,p=1$c2FsdHNhbHQ$aGFzaGhhc2hoYXNoaGFzaA"

func (f *fixture) createOperator(t *testing.T, username string) Operator {
	t.Helper()
	o, err := f.operators.Create(context.Background(), username, testHash)
	if err != nil {
		t.Fatal(err)
	}
	return o
}
