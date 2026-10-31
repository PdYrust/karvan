package safemode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/secret"
)

// Restorer puts a backup back. The panel must be stopped first: restoring
// under a running panel mixes its writes into the restored data.
type Restorer struct {
	Key *secret.Key
	Run Runner
}

// Restore replaces the database at t with backup b.
func (r Restorer) Restore(ctx context.Context, t paneldb.Target, s paneldb.Server, b Backup) error {
	in, err := Open(r.Key, b)
	if err != nil {
		return err
	}
	defer in.Close()
	if t.Dialect == paneldb.SQLite {
		return restoreSQLite(t.Path, in)
	}
	return r.restoreServer(ctx, t, s, in)
}

// restoreSQLite writes the copy next to the database, then renames it into
// place, so a crash leaves either the old file or the new one. The WAL and
// shared-memory files belong to the old file and go with it.
func restoreSQLite(path string, in io.Reader) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".karvan-restore-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("decrypt backup: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if info, err := os.Stat(path); err == nil {
		_ = os.Chmod(tmp.Name(), info.Mode().Perm())
	}
	for _, side := range []string{path + "-wal", path + "-shm"} {
		if err := os.Remove(side); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return os.Rename(tmp.Name(), path)
}

func (r Restorer) restoreServer(ctx context.Context, t paneldb.Target, s paneldb.Server, in io.Reader) error {
	if s.Dialect == paneldb.MySQL {
		tool := "mysql"
		if s.Flavor == "mariadb" {
			tool = "mariadb"
		}
		args := []string{"-u", t.User, "-h", t.Host, "-P", strconv.Itoa(t.Port), "--protocol=TCP", t.Database}
		return r.run(ctx, tool, args, append(os.Environ(), "MYSQL_PWD="+t.Password.Reveal()), in, nil)
	}
	return r.restorePostgres(ctx, t, in)
}

// restorePostgres feeds the dump to psql. With TimescaleDB it follows the
// extension's own procedure: timescaledb_pre_restore() stops its background
// workers, which would otherwise write their catalog rows (the telemetry
// job's exported_uuid) while the dump is copying the same rows back, and
// timescaledb_post_restore() starts them again afterwards.
func (r Restorer) restorePostgres(ctx context.Context, t paneldb.Target, in io.Reader) error {
	schema, err := r.psql(ctx, t, "SELECT quote_ident(n.nspname) FROM pg_extension e JOIN pg_namespace n ON n.oid = e.extnamespace WHERE e.extname = 'timescaledb'")
	if err != nil {
		return err
	}
	if schema != "" {
		// The setting applies to new sessions, so the restore runs in its own.
		if _, err := r.psql(ctx, t, "SELECT "+schema+".timescaledb_pre_restore()"); err != nil {
			return err
		}
	}
	err = r.run(ctx, "psql", r.psqlArgs(t), r.psqlEnv(t), in, nil)
	if schema != "" {
		if _, perr := r.psql(ctx, t, "SELECT "+schema+".timescaledb_post_restore()"); perr != nil {
			err = errors.Join(err, perr)
		}
	}
	return err
}

func (r Restorer) psqlArgs(t paneldb.Target) []string {
	return []string{"-v", "ON_ERROR_STOP=1", "-q", "-h", t.Host, "-p", strconv.Itoa(t.Port), "-U", t.User, "-d", t.Database}
}

func (r Restorer) psqlEnv(t paneldb.Target) []string {
	return append(os.Environ(), "PGPASSWORD="+t.Password.Reveal())
}

// psql runs one statement and returns its output, trimmed.
func (r Restorer) psql(ctx context.Context, t paneldb.Target, query string) (string, error) {
	var out strings.Builder
	err := r.run(ctx, "psql", append(r.psqlArgs(t), "-tAX", "-c", query), r.psqlEnv(t), nil, &out)
	return strings.TrimSpace(out.String()), err
}

func (r Restorer) run(ctx context.Context, tool string, args, env []string, in io.Reader, out io.Writer) error {
	cmd := r.Run(ctx, tool, args...)
	var stderr limited
	cmd.Env, cmd.Stdin, cmd.Stdout, cmd.Stderr = env, in, out, &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w: %s", tool, err, stderr.String())
	}
	return nil
}
