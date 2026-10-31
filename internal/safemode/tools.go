// Package safemode takes the destination backup before a migration writes
// anything: SQLite through VACUUM INTO, MySQL and MariaDB through
// their dump tool, PostgreSQL through pg_dump. Backups are encrypted as
// they are written and kept outside any rotation.
package safemode

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/PdYrust/karvan/internal/panel/paneldb"
)

// ToolError means the dump tool for a database is missing or too old.
// The dry run reports it, so the operator can install Package first.
type ToolError struct {
	Tool, Package, Reason string
}

func (e *ToolError) Error() string {
	return fmt.Sprintf("%s: %s (install %s)", e.Tool, e.Reason, e.Package)
}

// Runner starts external commands; tests swap it.
type Runner func(ctx context.Context, name string, args ...string) *exec.Cmd

// DefaultRunner runs commands from PATH.
func DefaultRunner(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...) //nolint:gosec // tool names are fixed; arguments are built here
}

// dumpTool picks the dump command for a server.
func dumpTool(s paneldb.Server) (tool, pkg string) {
	switch s.Flavor {
	case "mariadb":
		return "mariadb-dump", "mariadb-client"
	case "mysql":
		return "mysqldump", "mysql-client"
	case "postgres":
		return "pg_dump", "postgresql-client-" + major(s.Version)
	}
	return "", ""
}

var leadingNumber = regexp.MustCompile(`\d+`)

func major(version string) string { return leadingNumber.FindString(version) }

// Probe checks that a backup of s can be taken on this host.
func Probe(ctx context.Context, s paneldb.Server, run Runner) error {
	if s.Dialect == paneldb.SQLite {
		return nil
	}
	tool, pkg := dumpTool(s)
	out, err := run(ctx, tool, "--version").Output()
	if err != nil {
		var ee *exec.Error
		if errors.As(err, &ee) {
			return &ToolError{Tool: tool, Package: pkg, Reason: "not installed"}
		}
		return &ToolError{Tool: tool, Package: pkg, Reason: "does not run"}
	}
	if s.Flavor == "postgres" {
		// pg_dump refuses servers newer than itself.
		have, _ := strconv.Atoi(major(strings.TrimPrefix(string(out), "pg_dump (PostgreSQL) ")))
		want, _ := strconv.Atoi(major(s.Version))
		if have < want {
			return &ToolError{Tool: tool, Package: pkg, Reason: fmt.Sprintf("version %d is older than the server (%d)", have, want)}
		}
	}
	return nil
}
