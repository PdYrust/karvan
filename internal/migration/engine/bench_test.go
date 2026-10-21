package engine

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
)

// benchInputs is the scale from the phase file: 100k source users against
// 200k destination users, 5% of names and 1% of UUIDs already taken.
func benchInputs(b *testing.B) (plan.Plan, Inputs) {
	b.Helper()
	const sourceUsers, destUsers = 100_000, 200_000
	var existing []destUser
	for i := range destUsers {
		u := mUser(fmt.Sprintf("dest%06d", i), 1_000_000+i)
		existing = append(existing, destUser{int64(i + 1), u.Username, u})
	}
	d := dest(panel.Marzban, existing...)
	users := make([]panel.User, sourceUsers)
	for i := range users {
		name := fmt.Sprintf("src%06d", i)
		if i%20 == 0 {
			name = fmt.Sprintf("dest%06d", i) // taken name
		}
		u := mUser(name, i)
		if i%100 == 1 {
			u.Credentials.VLESS.ID = uuid('b', 1_000_000+i) // taken UUID
		}
		users[i] = u
	}
	p := basePlan()
	p.Tree = []plan.Node{
		{ID: "big", Conditions: []plan.Condition{{Field: "used_gb", Op: "gt", Value: "50000"}}, Ops: []plan.Op{{Kind: "traffic.reset_used"}}},
		{ID: "all", Ops: []plan.Op{{Kind: "time.add", Params: map[string]string{"days": "3"}}}},
	}
	p.Resolutions = map[string]plan.Resolution{NameTaken: {Option: OptPrefix, Params: map[string]string{"text": "shop_"}}}
	return p, inputs(panel.Marzban, d, users...)
}

func BenchmarkBuild100k(b *testing.B) {
	p, in := benchInputs(b)
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	b.ResetTimer()
	for range b.N {
		r, err := Build(p, in)
		if err != nil || !r.Stable {
			b.Fatal(err, r.Stable)
		}
		_ = Diff(r, r)
	}
	b.StopTimer()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	b.ReportMetric(float64(before.HeapInuse)/(1<<20), "inputMiB")
	b.ReportMetric(float64(after.HeapInuse-min(before.HeapInuse, after.HeapInuse))/(1<<20), "buildMiB")
}
