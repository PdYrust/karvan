// Package buildinfo holds version data that the build sets with -ldflags.
package buildinfo

import "fmt"

// Set at link time, for example:
// -ldflags "-X github.com/PdYrust/karvan/internal/buildinfo.Version=1.0.0".
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// String returns a one-line description of the running build.
func String() string {
	return fmt.Sprintf("karvan %s (commit %s, built %s)", Version, Commit, Date)
}
