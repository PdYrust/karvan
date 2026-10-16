package pasarguarddb

import (
	"context"
	"testing"

	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
)

func TestFixturesAreAtTheSupportedRevision(t *testing.T) {
	paneldbtest.Each(t, "testdata", Revision, func(t *testing.T, tg paneldb.Target) {
		ctx := context.Background()
		r, err := paneldb.OpenReader(ctx, tg)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		err = r.View(ctx, func(q paneldb.Querier) error {
			rev, err := paneldb.Revision(ctx, q)
			if err == nil && rev != Revision {
				t.Errorf("revision %q, want %q", rev, Revision)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}
