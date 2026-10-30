package respond

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

type listItem struct {
	Names []string        `json:"names"`
	Raw   json.RawMessage `json:"raw"`
	Bytes []byte          `json:"bytes"`
	When  time.Time       `json:"when"`
	Next  *listItem       `json:"next"`
	Tags  map[string]int  `json:"tags"`
	Any   any             `json:"any"`
	Kept  []int           `json:"kept,omitempty"`
}

// Nil slices become [] at any depth; nil pointers, maps, []byte, types
// that encode themselves and the value passed in stay as they are.
func TestListsWritesEmptySlices(t *testing.T) {
	in := map[string]any{
		"none":  []listItem(nil),
		"items": []listItem{{Next: &listItem{}, Any: []string(nil)}},
		"nil":   nil,
	}
	rec := httptest.NewRecorder()
	Lists(rec, 200, in)
	want := `{"items":[{"names":[],"raw":null,"bytes":null,"when":"0001-01-01T00:00:00Z","next":{"names":[],"raw":null,` +
		`"bytes":null,"when":"0001-01-01T00:00:00Z","next":null,"tags":null,"any":null},"tags":null,"any":[]}],"nil":null,"none":[]}` + "\n"
	if got := rec.Body.String(); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	if in["none"].([]listItem) != nil || in["items"].([]listItem)[0].Names != nil {
		t.Fatal("the value passed in changed")
	}
	if EmptyLists(nil) != nil {
		t.Fatal("nil is not nil")
	}
}
