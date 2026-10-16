//go:build dbtest

package migrationtest

import (
	"context"
	"os/exec"
	"slices"

	"github.com/PdYrust/karvan/internal/safemode"
)

// containers runs each MySQL-family client tool in the container of its
// own server (the test host has none).
var containers = map[string]string{
	"mysqldump": "karvan-testdb-mysql-1", "mysql": "karvan-testdb-mysql-1",
	"mariadb-dump": "karvan-testdb-mariadb-1", "mariadb": "karvan-testdb-mariadb-1",
}

// Tools runs MySQL and MariaDB tools inside the test containers, where
// the server listens on 3306, and everything else from PATH.
var Tools safemode.Runner = func(ctx context.Context, name string, args ...string) *exec.Cmd {
	container, ok := containers[name]
	if !ok {
		return safemode.DefaultRunner(ctx, name, args...)
	}
	args = slices.Clone(args)
	if i := slices.Index(args, "-P"); i >= 0 && i+1 < len(args) {
		args[i+1] = "3306"
	}
	return exec.CommandContext(ctx, "docker", append([]string{"exec", "-i", "-e", "MYSQL_PWD", container, name}, args...)...)
}
