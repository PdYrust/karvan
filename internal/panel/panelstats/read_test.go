package panelstats_test

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
	"github.com/PdYrust/karvan/internal/panel/panelstats"
)

func TestReadCountsUsersAdminsAndNodes(t *testing.T) {
	ctx := context.Background()
	for _, k := range kinds {
		t.Run(k.name, func(t *testing.T) {
			paneldbtest.Each(t, k.dir, k.rev, func(t *testing.T, tg paneldb.Target) {
				seed(t, tg, k)
				view(t, tg, func(q paneldb.Querier, _ paneldb.Server) error {
					d, err := panelstats.Read(ctx, q, tg.Dialect, k.schema, now)
					if err != nil {
						return err
					}
					want := panelstats.Data{
						Users: 5, ByStatus: map[string]int64{"active": 2, "disabled": 1, "expired": 1, "on_hold": 1},
						OnlineNow: 1, OnlineDay: 2, NewDay: 2, NewWeek: 3, ExpiringWeek: 1, UsedTraffic: 350, Admins: 2,
						TopAdmins: []panelstats.Admin{
							{Username: "reseller1", Users: 3, Active: 2, UsedTraffic: 300},
							{Username: "reseller2", Users: 2, Active: 0, UsedTraffic: 50},
						},
						Nodes:  []panelstats.Node{{Name: "de-1", Status: "connected", Uplink: 40, Downlink: 60}},
						Uplink: 10, Downlink: 20,
					}
					if !reflect.DeepEqual(d, want) {
						t.Errorf("got  %+v\nwant %+v", d, want)
					}
					return nil
				})
			})
		})
	}
}

func TestHistoryTrafficSignUpsAndApps(t *testing.T) {
	ctx := context.Background()
	for _, k := range kinds {
		t.Run(k.name, func(t *testing.T) {
			paneldbtest.Each(t, k.dir, k.rev, func(t *testing.T, tg paneldb.Target) {
				seed(t, tg, k)
				view(t, tg, func(q paneldb.Querier, _ paneldb.Server) error {
					hours, err := panelstats.Traffic(ctx, q, tg.Dialect, now.Add(-24*time.Hour))
					if err != nil {
						return err
					}
					// Two hours; the master's row (no node) adds to its hour.
					if len(hours) != 2 || !hours[0].At.Equal(now.Add(-3*time.Hour)) || hours[1].Uplink != 8 || hours[1].Downlink != 4 {
						t.Errorf("traffic %+v", hours)
					}
					signups, err := panelstats.NewUsers(ctx, q, tg.Dialect, now.Add(-24*time.Hour))
					if err != nil {
						return err
					}
					if len(signups) != 2 || signups[0].N != 1 || !signups[1].At.Equal(now.Add(-time.Hour)) {
						t.Errorf("sign-ups %+v", signups)
					}
					apps, err := panelstats.Clients(ctx, q, tg.Dialect, k.schema, now.Add(-7*24*time.Hour))
					if err != nil {
						return err
					}
					// Marzban knows each user's last app; PasarGuard counts every
					// update, so the first user's two updates come first.
					want := []panelstats.Client{{Name: "Hiddify", Count: 1}, {Name: "v2rayNG", Count: 1}}
					if k.schema.Updates {
						want = []panelstats.Client{{Name: "v2rayNG", Count: 2}, {Name: "Hiddify", Count: 1}}
					}
					if !slices.Equal(apps, want) {
						t.Errorf("apps %+v, want %+v", apps, want)
					}
					return nil
				})
			})
		})
	}
}

func TestEngineStatsAndTables(t *testing.T) {
	ctx := context.Background()
	for _, k := range kinds {
		t.Run(k.name, func(t *testing.T) {
			paneldbtest.Each(t, k.dir, k.rev, func(t *testing.T, tg paneldb.Target) {
				seed(t, tg, k)
				view(t, tg, func(q paneldb.Querier, srv paneldb.Server) error {
					e, err := paneldb.Engine(ctx, q, tg, srv)
					if err != nil {
						return err
					}
					checkEngine(t, tg.Dialect, srv, e)
					tables, err := paneldb.Tables(ctx, q, tg.Dialect, 50)
					if err != nil {
						return err
					}
					i := slices.IndexFunc(tables, func(x paneldb.TableSize) bool { return x.Name == "users" })
					if i < 0 || tables[i].Bytes <= 0 || (tg.Dialect == paneldb.SQLite && tables[i].Rows != 5) {
						t.Errorf("tables %+v", tables)
					}
					return nil
				})
			})
		})
	}
}

// checkEngine wants SQLite's page facts, and a server's live counters.
func checkEngine(t *testing.T, d paneldb.Dialect, srv paneldb.Server, e paneldb.EngineStats) {
	t.Helper()
	ok := e.Size > 0 && e.Version != "" && e.Flavor == srv.Flavor
	if d == paneldb.SQLite {
		ok = ok && e.PageSize > 0 && e.Pages > 0 && e.JournalMode == "wal"
	} else {
		ok = ok && e.MaxConnections > 0 && e.Connections > 0 && e.Queries > 0
	}
	if !ok {
		t.Errorf("%s engine %+v", d, e)
	}
}

func TestClientNames(t *testing.T) {
	for ua, want := range map[string]string{
		"v2rayNG/1.8.5":                    "v2rayNG",
		"v2rayN/6.42":                      "v2rayN",
		"HiddifyNext/2.5.7 (android)":      "Hiddify",
		"Shadowrocket/2070 CFNetwork/1410": "Shadowrocket",
		"SFA/1.9.3 (sing-box 1.9.3)":       "sing-box",
		"ClashMeta/1.18":                   "Clash",
		"Mozilla/5.0 (Windows NT 10.0)":    "Browser",
		"SomethingNew/1.0":                 panelstats.OtherClient,
	} {
		if got := panelstats.ClientName(ua); got != want {
			t.Errorf("%q: got %q, want %q", ua, got, want)
		}
	}
}
