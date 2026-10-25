package handlers_test

import (
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
)

const account = api + "/account"

// browser is a second browser on the same server: its own cookies.
func (e *env) browser() *env {
	b := *e
	jar, _ := cookiejar.New(nil)
	b.client = &http.Client{Jar: jar, CheckRedirect: e.client.CheckRedirect}
	b.csrf = ""
	return &b
}

type sessionsBody struct {
	Sessions []struct {
		ID        string `json:"id"`
		IP        string `json:"ip"`
		UserAgent string `json:"userAgent"`
		Current   bool   `json:"current"`
	} `json:"sessions"`
}

func (e *env) sessions() sessionsBody {
	e.t.Helper()
	var s sessionsBody
	res := e.do(http.MethodGet, account+"/sessions", nil)
	if res.status != http.StatusOK {
		e.t.Fatalf("sessions: %d %s", res.status, res.body)
	}
	res.json(e.t, &s)
	return s
}

func TestSessionsListAndEndOne(t *testing.T) {
	e := newEnv(t)
	e.login("")
	other := e.browser()
	other.login("")

	s := e.sessions()
	if len(s.Sessions) != 2 || s.Sessions[0].IP != "127.0.0.1" || s.Sessions[0].UserAgent == "" {
		t.Fatalf("sessions = %+v", s)
	}
	var otherID, ownID string
	for _, x := range s.Sessions {
		if x.Current {
			ownID = x.ID
		} else {
			otherID = x.ID
		}
	}
	if ownID == "" || otherID == "" {
		t.Fatalf("current not marked: %+v", s)
	}
	for id, want := range map[string]int{"short": http.StatusBadRequest, strings.Repeat("A", 16): http.StatusNotFound} {
		if res := e.do(http.MethodDelete, account+"/sessions/"+id, nil); res.status != want {
			t.Fatalf("end %s: %d, want %d", id, res.status, want)
		}
	}
	if res := e.do(http.MethodDelete, account+"/sessions/"+otherID, nil); res.status != http.StatusNoContent {
		t.Fatalf("end other: %d %s", res.status, res.body)
	}
	if res := other.do(http.MethodGet, account+"/sessions", nil); res.status != http.StatusUnauthorized {
		t.Fatalf("ended browser: %d", res.status)
	}
	if res := e.do(http.MethodDelete, account+"/sessions/"+ownID, nil); res.status != http.StatusNoContent || !strings.Contains(res.header.Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("end own: %d %v", res.status, res.header)
	}
	if res := e.do(http.MethodGet, account+"/sessions", nil); res.status != http.StatusUnauthorized {
		t.Fatalf("after ending its own session: %d", res.status)
	}
}

func TestSignOutOtherSessions(t *testing.T) {
	e := newEnv(t)
	e.login("")
	others := []*env{e.browser(), e.browser()}
	for _, o := range others {
		o.login("")
	}
	var ended struct {
		Ended int `json:"ended"`
	}
	res := e.do(http.MethodDelete, account+"/sessions", nil)
	res.json(t, &ended)
	if res.status != http.StatusOK || ended.Ended != 2 {
		t.Fatalf("sign out others: %d %s", res.status, res.body)
	}
	for _, o := range others {
		if res := o.do(http.MethodGet, account+"/sessions", nil); res.status != http.StatusUnauthorized {
			t.Fatalf("other browser: %d", res.status)
		}
	}
	if s := e.sessions(); len(s.Sessions) != 1 || !s.Sessions[0].Current {
		t.Fatalf("left: %+v", s)
	}
	if n := auditCount(t, e, "sessions.end", "success"); n != 1 {
		t.Fatalf("audited %d", n)
	}
}

// auditCount counts events with action and outcome through GET /audit.
func auditCount(t *testing.T, e *env, action, outcome string) int {
	t.Helper()
	var page auditPage
	res := e.do(http.MethodGet, api+"/audit?limit=200&action="+action+"&outcome="+outcome, nil)
	if res.status != http.StatusOK {
		t.Fatalf("audit: %d %s", res.status, res.body)
	}
	res.json(t, &page)
	return len(page.Events)
}
