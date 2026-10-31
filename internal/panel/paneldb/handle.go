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

// ErrLocked means another Karvan (or another job) holds the destination.
var ErrLocked = errors.New("another job is writing to this database")

// Lock takes Karvan's exclusive lock on the destination itself, so
// two Karvan installs, or two ENV files naming one database, cannot write
// at once. It never waits. SQLite uses flock on the database file, which
// is separate from SQLite's own POSIX locks on Linux; MySQL and MariaDB use
// GET_LOCK and PostgreSQL an advisory lock, each held by one connection.
func (w *Writer) Lock(ctx context.Context, t Target) (release func(), err error) {
	if t.Dialect == SQLite {
		return flock(t.Path)
	}
	conn, err := w.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	name := "karvan:" + t.Database
	var ok bool
	if t.Dialect == MySQL {
		var got sql.NullInt64
		err = conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", name).Scan(&got)
		ok = got.Int64 == 1
	} else {
		err = conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock(hashtext($1))", name).Scan(&ok)
	}
	if err == nil && !ok {
		err = ErrLocked
	}
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return func() {
		// Closing the connection drops the lock even if the unlock fails.
		bg := context.Background()
		if t.Dialect == MySQL {
			_, _ = conn.ExecContext(bg, "SELECT RELEASE_LOCK(?)", name)
		} else {
			_, _ = conn.ExecContext(bg, "SELECT pg_advisory_unlock(hashtext($1))", name)
		}
		_ = conn.Close()
	}, nil
}

// Writer is a read-write handle. Writes happen only inside Chunk.
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

// Chunk runs fn in one transaction and commits only if fn returns nil.
func (w *Writer) Chunk(ctx context.Context, fn func(x Execer) error) error {
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

// Revision reads the alembic revision. Several heads (a database stuck
// mid-merge) come back joined with "+", so they never match a known one.
func Revision(ctx context.Context, q Querier) (string, error) {
	rows, err := q.QueryContext(ctx, "SELECT version_num FROM alembic_version ORDER BY version_num")
	if err != nil {
		return "", fmt.Errorf("read alembic_version: %w", err)
	}
	defer rows.Close()
	var revs []string
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			return "", err
		}
		revs = append(revs, r)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return strings.Join(revs, "+"), nil
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

// InsertID runs an INSERT and returns the new row's id. Each row is its
// own statement: MySQL hands out ids for a multi-row insert that need not
// be contiguous when the panel inserts at the same time.
func InsertID(ctx context.Context, x Execer, d Dialect, query string, args ...any) (int64, error) {
	if d == Postgres {
		var id int64
		err := x.QueryRowContext(ctx, query+" RETURNING id", args...).Scan(&id)
		return id, err
	}
	res, err := x.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// In returns "?, ?, ..." for n values and the values as arguments.
func In[T any](vals []T) (string, []any) {
	args := make([]any, len(vals))
	for i, v := range vals {
		args[i] = v
	}
	return strings.TrimSuffix(strings.Repeat("?, ", len(vals)), ", "), args
}

// LockRows takes row locks on table's ids for the rest of the transaction
// on servers that have them; SQLite has one writer, so it needs none.
func LockRows(ctx context.Context, x Execer, d Dialect, table string, ids []int64) error {
	if d == SQLite || len(ids) == 0 {
		return nil
	}
	in, args := In(ids)
	rows, err := x.QueryContext(ctx, "SELECT id FROM "+Ident(d, table)+" WHERE id IN ("+in+") FOR UPDATE", args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	locked := 0
	for rows.Next() {
		locked++
	}
	return rows.Err()
}

// Ident quotes a table or column name for d. Needed for names that are
// reserved words somewhere, like PasarGuard's "groups" on MySQL 8.
func Ident(d Dialect, name string) string {
	if d == MySQL {
		return "`" + name + "`"
	}
	return `"` + name + `"`
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
