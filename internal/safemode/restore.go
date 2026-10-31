package safemode

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

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
	tool, args, env := "psql", []string{"-v", "ON_ERROR_STOP=1", "-q", "-h", t.Host, "-p", strconv.Itoa(t.Port), "-U", t.User, "-d", t.Database},
		append(os.Environ(), "PGPASSWORD="+t.Password.Reveal())
	if s.Dialect == paneldb.MySQL {
		tool = "mysql"
		if s.Flavor == "mariadb" {
			tool = "mariadb"
		}
		args = []string{"-u", t.User, "-h", t.Host, "-P", strconv.Itoa(t.Port), "--protocol=TCP", t.Database}
		env = append(os.Environ(), "MYSQL_PWD="+t.Password.Reveal())
	}
	cmd := r.Run(ctx, tool, args...)
	var stderr limited
	cmd.Env, cmd.Stdin, cmd.Stderr = env, in, &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w: %s", tool, err, stderr.String())
	}
	return nil
}
