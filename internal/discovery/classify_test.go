package discovery

import (
	"slices"
	"testing"

	"github.com/PdYrust/karvan/internal/panel"
)

func TestNoteSchemaOfACheckedRevision(t *testing.T) {
	p := &Panel{Kind: panel.PasarGuard, Findings: []Finding{}}
	noteSchema(p, true, nil)
	if p.RevisionNote != panel.RevisionChecked || !p.Ready() {
		t.Fatalf("panel = %+v", p)
	}
}

func TestNoteSchemaOfARevisionThatDiffers(t *testing.T) {
	p := &Panel{Findings: []Finding{}}
	noteSchema(p, false, &panel.VersionError{Kind: panel.Marzban, Revision: "e56f1c781e46", Diffs: []string{"users.note missing"}})
	want := []Finding{{Code: UnknownRevision, Detail: "e56f1c781e46"}}
	if !slices.Equal(p.Findings, want) || !slices.Equal(p.SchemaDiffs, []string{"users.note missing"}) || p.RevisionNote != "" {
		t.Fatalf("panel = %+v", p)
	}
}
