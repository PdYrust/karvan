package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/PdYrust/karvan/internal/audit"
)

type auditPage struct {
	Events []struct {
		ID      int64             `json:"id"`
		Actor   string            `json:"actor"`
		IP      string            `json:"ip"`
		Action  string            `json:"action"`
		Outcome string            `json:"outcome"`
		Detail  map[string]string `json:"detail"`
	} `json:"events"`
	Next *int64 `json:"next"`
}

func TestAuditLogPagesWithFilters(t *testing.T) {
	e := newEnv(t)
	e.login("")
	log := audit.New(e.db)
	for i := range 5 {
		outcome := audit.Success
		if i%2 == 1 {
			outcome = audit.Failure
		}
		if err := log.Record(context.Background(), audit.Event{Actor: "owner", IP: "203.0.113.1", Action: audit.ActionSafeModeOff, Outcome: outcome,
			Detail: map[string]string{"n": fmt.Sprint(i)}}); err != nil {
			t.Fatal(err)
		}
	}
	var first, second auditPage
	e.do(http.MethodGet, api+"/audit?limit=2&action=safe_mode.off&outcome=success", nil).json(t, &first)
	if len(first.Events) != 2 || first.Next == nil || first.Events[0].Detail["n"] != "4" || first.Events[1].Detail["n"] != "2" {
		t.Fatalf("first page = %+v", first)
	}
	e.do(http.MethodGet, fmt.Sprintf("%s/audit?limit=2&action=safe_mode.off&outcome=success&before=%d", api, *first.Next), nil).json(t, &second)
	if len(second.Events) != 1 || second.Next != nil || second.Events[0].Detail["n"] != "0" {
		t.Fatalf("second page = %+v", second)
	}

	var all auditPage
	e.do(http.MethodGet, api+"/audit", nil).json(t, &all)
	if len(all.Events) != 6 || all.Events[5].Action != audit.ActionLogin || all.Events[5].Detail == nil || all.Next != nil {
		t.Fatalf("all = %+v", all)
	}
}

func TestAuditLogRefusesBadQueries(t *testing.T) {
	e := newEnv(t)
	if res := e.do(http.MethodGet, api+"/audit", nil); res.status != http.StatusUnauthorized {
		t.Fatalf("signed out: %d", res.status)
	}
	e.login("")
	for _, q := range []string{"action=DROP%20TABLE", "outcome=maybe", "before=0", "before=x", "limit=0", "limit=201"} {
		if res := e.do(http.MethodGet, api+"/audit?"+q, nil); res.status != http.StatusBadRequest {
			t.Fatalf("%s: %d", q, res.status)
		}
	}
}
