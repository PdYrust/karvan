package handlers_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/hoststats"
	"github.com/PdYrust/karvan/internal/migration/migrationtest"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
)

const analytics = "/" + base + "/api/v1/analytics"

func TestServerAnalytics(t *testing.T) {
	e := newEnv(t)
	if res := e.do(http.MethodGet, analytics+"/server", nil); res.status != http.StatusUnauthorized {
		t.Fatalf("signed out: %d", res.status)
	}
	e.login("")
	ctx := context.Background()
	e.host.Tick(ctx)
	time.Sleep(20 * time.Millisecond)
	e.host.Tick(ctx)

	var live struct {
		Available bool               `json:"available"`
		Step      int64              `json:"step"`
		Seq       int64              `json:"seq"`
		Facts     hoststats.Facts    `json:"facts"`
		Now       *hoststats.Sample  `json:"now"`
		Samples   []hoststats.Sample `json:"samples"`
		Points    []hoststats.Minute `json:"points"`
	}
	res := e.do(http.MethodGet, analytics+"/server", nil)
	if res.status != http.StatusOK {
		t.Fatalf("live: %d %s", res.status, res.body)
	}
	res.json(t, &live)
	if !live.Available || live.Now == nil {
		t.Fatalf("live = %+v", live)
	}
	for name, ok := range map[string]bool{
		"step":    live.Step == 5,
		"samples": len(live.Samples) == 1,
		"cores":   live.Facts.Cores >= 1,
		"memory":  live.Now.Mem.Total > 0,
	} {
		if !ok {
			t.Errorf("live %s: %+v", name, live)
		}
	}
	res = e.do(http.MethodGet, analytics+"/server?after="+strconv.FormatInt(live.Seq, 10), nil)
	live.Samples = nil
	res.json(t, &live)
	if len(live.Samples) != 0 {
		t.Fatalf("after the latest seq: %d samples", len(live.Samples))
	}

	minute := time.Now().Add(-10*time.Minute).Unix() / 60 * 60
	if err := e.host.Store.Put(ctx, hoststats.Minute{TS: minute, CPU: 0.25, MemTotal: 100}); err != nil {
		t.Fatal(err)
	}
	res = e.do(http.MethodGet, analytics+"/server?range=24h", nil)
	res.json(t, &live)
	if res.status != http.StatusOK || live.Step != 300 || len(live.Points) != 1 || live.Points[0].CPU != 0.25 {
		t.Fatalf("24h: %d %+v", res.status, live.Points)
	}
	if res := e.do(http.MethodGet, analytics+"/server?range=2y", nil); res.status != http.StatusBadRequest {
		t.Fatalf("unknown range: %d", res.status)
	}
}

type liveJSON struct {
	Seq      int64               `json:"seq"`
	Samples  []hoststats.Sample  `json:"samples"`
	Backfill *[]hoststats.Minute `json:"backfill"`
}

func (e *env) live(t *testing.T, query string) liveJSON {
	t.Helper()
	var out liveJSON
	res := e.do(http.MethodGet, analytics+"/server"+query, nil)
	if res.status != http.StatusOK {
		t.Fatalf("live%s: %d %s", query, res.status, res.body)
	}
	res.json(t, &out)
	return out
}

func TestServerLiveBackfill(t *testing.T) {
	e := newEnv(t)
	e.login("")
	ctx := context.Background()
	minute := time.Now().Add(-10*time.Minute).Unix() / 60 * 60
	if err := e.host.Store.Put(ctx, hoststats.Minute{TS: minute, CPU: 0.25, MemTotal: 100}); err != nil {
		t.Fatal(err)
	}
	e.host.Tick(ctx)
	time.Sleep(20 * time.Millisecond)
	e.host.Tick(ctx)
	first := e.live(t, "")
	if first.Backfill == nil || len(*first.Backfill) != 1 || (*first.Backfill)[0].TS != minute || len(first.Samples) != 1 {
		t.Fatalf("first answer: %+v", first)
	}
	if next := e.live(t, "?after="+strconv.FormatInt(first.Seq, 10)); next.Backfill != nil || len(next.Samples) != 0 {
		t.Fatalf("a poll that follows on: %+v", next)
	}
	// A seq from before a restart: every sample again, with the minutes.
	if again := e.live(t, "?after="+strconv.FormatInt(first.Seq+100, 10)); again.Backfill == nil || len(again.Samples) != 1 {
		t.Fatalf("after a restart: %+v", again)
	}
	if over := e.live(t, "?range=1h"); over.Backfill != nil {
		t.Fatalf("a range has no backfill: %+v", over)
	}
}

type panelJSON struct {
	PanelID int64 `json:"panelId"`
	Pending bool  `json:"pending"`
	Data    *struct {
		Users int64 `json:"users"`
	} `json:"data"`
	Engine *struct {
		Flavor string `json:"flavor"`
	} `json:"engine"`
	Points []struct {
		Size int64 `json:"size"`
	} `json:"points"`
}

// statsPanel registers the PasarGuard SQLite fixture as a panel.
func statsPanel(t *testing.T, e *env) int64 {
	t.Helper()
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/pasarguarddb/testdata/" + pasarguarddb.Revision + ".sqlite.sql"})
	id, err := e.runs.RegisterPanel(context.Background(), migrationtest.PanelEnv(t, tg, panel.PasarGuard), "pasarguard", string(panel.PasarGuard))
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestPanelsAnalytics(t *testing.T) {
	e := newEnv(t)
	e.login("")
	id := statsPanel(t, e)
	var list struct {
		Panels []panelJSON `json:"panels"`
	}
	e.do(http.MethodGet, analytics+"/panels", nil).json(t, &list)
	if len(list.Panels) != 1 || !list.Panels[0].Pending || list.Panels[0].PanelID != id {
		t.Fatalf("before the first reading: %+v", list.Panels)
	}
	e.dbstats.Tick(context.Background())
	res := e.do(http.MethodGet, analytics+"/panels?range=1h", nil)
	list.Panels = nil
	res.json(t, &list)
	if res.status != http.StatusOK || len(list.Panels) != 1 {
		t.Fatalf("after a reading: %d %s", res.status, res.body)
	}
	if p := list.Panels[0]; p.Pending || p.Data == nil || p.Engine == nil || p.Engine.Flavor != "sqlite" || len(p.Points) != 1 {
		t.Fatalf("after a reading: %s", res.body)
	}
	if res := e.do(http.MethodGet, analytics+"/panels?range=x", nil); res.status != http.StatusBadRequest {
		t.Fatalf("unknown range: %d", res.status)
	}
}

func TestPanelAnalytics(t *testing.T) {
	e := newEnv(t)
	e.login("")
	id := statsPanel(t, e)
	e.dbstats.Tick(context.Background())
	var one struct {
		Panel  panelJSON `json:"panel"`
		Detail *struct {
			Tables []struct {
				Name string `json:"name"`
			} `json:"tables"`
		} `json:"detail"`
	}
	res := e.do(http.MethodGet, analytics+"/panels/"+strconv.FormatInt(id, 10)+"?range=7d", nil)
	res.json(t, &one)
	if res.status != http.StatusOK || one.Panel.PanelID != id || one.Detail == nil || len(one.Detail.Tables) == 0 {
		t.Fatalf("one panel: %d %s", res.status, res.body)
	}
	for path, status := range map[string]int{"/panels/999": http.StatusNotFound, "/panels/x": http.StatusBadRequest} {
		if res := e.do(http.MethodGet, analytics+path, nil); res.status != status {
			t.Errorf("%s: %d, want %d", path, res.status, status)
		}
	}
}
