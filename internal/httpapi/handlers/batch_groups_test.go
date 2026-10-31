package handlers_test

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/batch"
)

// The group list has one entry per old panel with its runs.
func TestBatchGroupsList(t *testing.T) {
	m := newMigrationEnv(t, 2)
	_, b := doneBatch(t, m)
	var page struct {
		Groups []batch.Group `json:"groups"`
		Next   string        `json:"next"`
	}
	if res := m.do(http.MethodGet, api+"/batch-groups?runs=1", nil); res.status != http.StatusOK {
		t.Fatalf("list: %d %s", res.status, res.body)
	} else {
		res.json(t, &page)
	}
	if len(page.Groups) != 1 || page.Next != "" {
		t.Fatalf("groups %+v", page)
	}
	g := page.Groups[0]
	if g.ID != b.ID || g.LatestID != b.ID || g.Source.Origin != b.Source.Origin || g.Counts.Runs != 1 || g.Counts.Users != 2 ||
		len(g.Runs) != 1 || g.Runs[0].ID != b.ID {
		t.Fatalf("group %+v", g)
	}
	for _, q := range []string{"?state=deleted", "?runs=6", "?runs=-1", "?limit=0", "?panel=x", "?cursor=!"} {
		wantError(t, m.do(http.MethodGet, api+"/batch-groups"+q, nil), http.StatusBadRequest, "bad_request")
	}
	wantError(t, m.do(http.MethodGet, api+"/batch-groups/999", nil), http.StatusNotFound, "not_found")
	wantError(t, m.do(http.MethodGet, api+"/batch-groups/x", nil), http.StatusBadRequest, "bad_request")
}

// Any batch of a group reads it, and the batch list keeps one group's runs.
func TestBatchGroupGetAndRuns(t *testing.T) {
	m := newMigrationEnv(t, 1)
	_, b := doneBatch(t, m)
	var one batch.Group
	m.do(http.MethodGet, api+"/batch-groups/"+strconv.FormatInt(b.ID, 10)+"?runs=0", nil).json(t, &one)
	if one.ID != b.ID || len(one.Runs) != 0 {
		t.Fatalf("one %+v", one)
	}
	var runs struct {
		Batches []batch.Batch `json:"batches"`
	}
	m.do(http.MethodGet, api+"/batches?group="+strconv.FormatInt(b.ID, 10), nil).json(t, &runs)
	if len(runs.Batches) != 1 || runs.Batches[0].ID != b.ID {
		t.Fatalf("runs %+v", runs)
	}
}
