package safemode

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/secret"
)

// Backup is one encrypted destination backup.
type Backup struct {
	ID        string
	Path      string
	Size      int64
	Method    string // "vacuum-into", "mysqldump", "mariadb-dump", "pg_dump"
	CreatedAt time.Time
}

// Taker takes backups into Dir.
type Taker struct {
	Dir string // <data>/safe-mode, 0700
	Key *secret.Key
	Run Runner
	Now func() time.Time
}

// AAD binds a backup file to its id, so files cannot be swapped.
func AAD(id string) string { return "safe-mode/" + id }

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Take backs up t (on server s) and returns the encrypted file. A failed
// backup leaves no file behind.
func (k Taker) Take(ctx context.Context, t paneldb.Target, s paneldb.Server) (Backup, error) {
	if err := os.MkdirAll(k.Dir, 0o700); err != nil {
		return Backup{}, fmt.Errorf("create backup folder: %w", err)
	}
	b := Backup{ID: newID(), CreatedAt: k.Now().UTC()}
	b.Path = filepath.Join(k.Dir, b.ID+".kvb")
	f, err := os.OpenFile(b.Path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return Backup{}, fmt.Errorf("create backup file: %w", err)
	}
	err = k.write(ctx, f, t, s, &b)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(b.Path)
		return Backup{}, err
	}
	info, err := os.Stat(b.Path)
	if err != nil {
		return Backup{}, err
	}
	b.Size = info.Size()
	return b, nil
}

func (k Taker) write(ctx context.Context, f *os.File, t paneldb.Target, s paneldb.Server, b *Backup) error {
	enc, err := k.Key.NewStreamWriter(f, AAD(b.ID))
	if err != nil {
		return err
	}
	switch t.Dialect {
	case paneldb.SQLite:
		b.Method = "vacuum-into"
		err = k.sqlite(ctx, t, enc)
	default:
		b.Method, _ = dumpTool(s)
		err = k.dump(ctx, t, s, enc)
	}
	if err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return f.Sync()
}

// sqlite copies a consistent snapshot (WAL included) with VACUUM INTO into
// a private temp file, then encrypts it and removes the plain copy.
func (k Taker) sqlite(ctx context.Context, t paneldb.Target, w io.Writer) error {
	tmp, err := os.MkdirTemp(k.Dir, "plain-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	plain := filepath.Join(tmp, "db.sqlite3")
	db, err := sql.Open("sqlite", (&url.URL{Scheme: "file", Path: t.Path, RawQuery: "mode=ro&_pragma=busy_timeout(5000)"}).String())
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", plain); err != nil {
		return fmt.Errorf("copy SQLite database: %w", err)
	}
	in, err := os.Open(plain) //nolint:gosec // our own temp file
	if err != nil {
		return err
	}
	defer in.Close()
	_, err = io.Copy(w, in)
	return err
}

// dump streams the server's dump tool output into w. The password goes
// through the environment, never the command line.
func (k Taker) dump(ctx context.Context, t paneldb.Target, s paneldb.Server, w io.Writer) error {
	tool, _ := dumpTool(s)
	var args []string
	env := os.Environ()
	if s.Dialect == paneldb.Postgres {
		args = []string{"--no-owner", "--no-privileges", "--clean", "--if-exists", "-h", t.Host, "-p", strconv.Itoa(t.Port), "-U", t.User, "-d", t.Database}
		env = append(env, "PGPASSWORD="+t.Password.Reveal())
	} else {
		args = []string{"--single-transaction", "--quick", "--no-tablespaces", "--routines", "-u", t.User}
		if t.Socket != "" {
			args = append(args, "--socket="+t.Socket)
		} else {
			args = append(args, "-h", t.Host, "-P", strconv.Itoa(t.Port), "--protocol=TCP")
		}
		args = append(args, t.Database)
		env = append(env, "MYSQL_PWD="+t.Password.Reveal())
	}
	cmd := k.Run(ctx, tool, args...)
	cmd.Env, cmd.Stdout = env, w
	var stderr limited
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w: %s", tool, err, stderr.String())
	}
	return nil
}

// limited keeps the first 4 KiB of a tool's error output for the log.
type limited struct{ b []byte }

func (l *limited) Write(p []byte) (int, error) {
	if room := 4096 - len(l.b); room > 0 {
		l.b = append(l.b, p[:min(room, len(p))]...)
	}
	return len(p), nil
}

func (l *limited) String() string { return string(l.b) }

// Open decrypts a backup for reading (restore, inspection).
func Open(key *secret.Key, b Backup) (io.ReadCloser, error) {
	f, err := os.Open(b.Path)
	if err != nil {
		return nil, err
	}
	r, err := key.NewStreamReader(f, AAD(b.ID))
	if err != nil {
		_ = f.Close()
		return nil, errors.Join(err, errors.New("backup "+b.ID))
	}
	return struct {
		io.Reader
		io.Closer
	}{r, f}, nil
}
