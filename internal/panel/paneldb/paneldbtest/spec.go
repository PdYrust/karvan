package paneldbtest

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/PdYrust/karvan/internal/panel/paneldb"
)

// KnownSpec reads the shape of the tables in columns from tg, a database
// at a known revision, and checks it against the spec for tg's server
// flavor in file (see paneldb.ParseSpecs). It also checks that the shape
// passes its own Diff, so no column the writer leaves out is required.
// With update it writes the shape into file instead.
func KnownSpec(t *testing.T, tg paneldb.Target, file string, columns map[string][]string, inserts []string, update bool) {
	t.Helper()
	ctx := context.Background()
	r, err := paneldb.OpenReader(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var live paneldb.Spec
	err = r.View(ctx, func(q paneldb.Querier) error {
		var err error
		live, err = paneldb.ReadSpec(ctx, q, tg.Dialect, slices.Sorted(maps.Keys(columns)))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	got, flavor := live.Only(columns), r.Server().Flavor
	specs := map[string]paneldb.Spec{}
	if b, err := os.ReadFile(filepath.Clean(file)); err == nil {
		if specs, err = paneldb.ParseSpecs(b); err != nil {
			t.Fatal(err)
		}
	}
	if update {
		specs[flavor] = got
		var b bytes.Buffer
		e := json.NewEncoder(&b)
		e.SetEscapeHTML(false)
		e.SetIndent("", "  ")
		err := e.Encode(specs)
		if err == nil {
			err = os.WriteFile(file, b.Bytes(), 0o600)
		}
		if err != nil {
			t.Fatal(err)
		}
		return
	}
	if want := specs[flavor]; !reflect.DeepEqual(want, got) {
		t.Errorf("%s differs from %s: lost %q, gained %q", flavor, file, paneldb.Diff(want, got, nil), paneldb.Diff(got, want, nil))
	}
	if d := paneldb.Diff(got, live, inserts); d != nil {
		t.Errorf("the known schema fails its own check: %q", d)
	}
}
