package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/PdYrust/karvan/internal/buildinfo"
	"github.com/PdYrust/karvan/internal/store"
)

// versionJSON is what "karvan version --json" prints for the installer:
// schema is the store schema version this build migrates to (D24).
type versionJSON struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Schema  int    `json:"schema"`
}

func runVersion(_ context.Context, streams IO, args []string) error {
	fs := newFlagSet("version", streams)
	asJSON := fs.Bool("json", false, "print the version, commit and store schema version as JSON")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if !*asJSON {
		_, err := fmt.Fprintln(streams.Stdout, buildinfo.String())
		return err
	}
	schema, err := store.SchemaVersion()
	if err != nil {
		return err
	}
	return json.NewEncoder(streams.Stdout).Encode(versionJSON{
		Version: buildinfo.Version, Commit: buildinfo.Commit, Date: buildinfo.Date, Schema: schema,
	})
}
