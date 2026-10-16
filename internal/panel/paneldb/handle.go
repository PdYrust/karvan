package paneldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Querier is the read side of a transaction.
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Execer can also write.
type Execer interface {
	Querier
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Reader is a read-only handle. Its type has no way to write, and every
// connection under it is read only on the server too, so a dry run cannot
// change the destination even through a bug.
type Reader struct {
	db     *sql.DB
	server Server
}

// OpenReader opens t read only.
func OpenReader(ctx context.Context, t Target) (*Reader, error) {
	db, s, err := open(ctx, t, readOnly)
	if err != nil {
		return nil, fmt.Errorf("open %s read only: %w", t, err)
	}
	return &Reader{db: db, server: s}, nil
}

// Server reports the database server.
func (r *Reader) Server() Server { return r.server }

// Close closes the pool.
func (r *Reader) Close() error { return r.db.Close() }

// View runs fn in one read-only transaction, so every query sees the same
// snapshot where the database offers one.
func (r *Reader) View(ctx context.Context, fn func(q Querier) error) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // read only; nothing to keep
	return fn(readOnlyTx{rebinder{tx: tx, pg: r.server.Dialect == Postgres}})
}

// readOnlyTx hides ExecContext, so a Querier from View cannot be asserted
// back into something that writes.
type readOnlyTx struct{ r rebinder }

func (t readOnlyTx) QueryContext(ctx context.Context, s string, args ...any) (*sql.Rows, error) {
	return t.r.QueryContext(ctx, s, args...)
}

func (t readOnlyTx) QueryRowContext(ctx context.Context, s string, args ...any) *sql.Row {
	return t.r.QueryRowContext(ctx, s, args...)
}

// Writer is a read-write handle. Writes happen only inside Batch.
type Writer struct {
	db     *sql.DB
	server Server
}

// OpenWriter opens t for writing. SQLite transactions take the write lock
// up front (BEGIN IMMEDIATE) and foreign keys are enforced.
func OpenWriter(ctx context.Context, t Target) (*Writer, error) {
	db, s, err := open(ctx, t, readWrite)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", t, err)
	}
	return &Writer{db: db, server: s}, nil
}

// Server reports the database server.
func (w *Writer) Server() Server { return w.server }

// Close closes the pool.
func (w *Writer) Close() error { return w.db.Close() }

// ErrCommitUnknown means the commit was sent but its outcome is not known
// (a lost connection, a timeout). The caller must read back before going on.
var ErrCommitUnknown = errors.New("commit outcome unknown")

// Batch runs fn in one transaction and commits only if fn returns nil.
func (w *Writer) Batch(ctx context.Context, fn func(x Execer) error) error {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(rebinder{tx: tx, pg: w.server.Dialect == Postgres}); err != nil {
		if rb := tx.Rollback(); rb != nil {
			return errors.Join(err, rb)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: %w", ErrCommitUnknown, err)
	}
	return nil
}

// rebinder lets every query use "?" placeholders; PostgreSQL gets $n.
type rebinder struct {
	tx *sql.Tx
	pg bool
}

func (r rebinder) q(s string) string {
	if !r.pg {
		return s
	}
	return Rebind(s)
}

func (r rebinder) QueryContext(ctx context.Context, s string, args ...any) (*sql.Rows, error) {
	return r.tx.QueryContext(ctx, r.q(s), args...)
}

func (r rebinder) QueryRowContext(ctx context.Context, s string, args ...any) *sql.Row {
	return r.tx.QueryRowContext(ctx, r.q(s), args...)
}

func (r rebinder) ExecContext(ctx context.Context, s string, args ...any) (sql.Result, error) {
	return r.tx.ExecContext(ctx, r.q(s), args...)
}

// Rebind turns "?" placeholders into PostgreSQL's $1, $2... Queries in
// Karvan never carry "?" inside string literals.
func Rebind(s string) string {
	var b strings.Builder
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
