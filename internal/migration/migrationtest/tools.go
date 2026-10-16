//go:build !dbtest

package migrationtest

import "github.com/PdYrust/karvan/internal/safemode"

// Tools runs dump tools; the fast gate only needs SQLite, which uses none.
var Tools safemode.Runner = safemode.DefaultRunner
