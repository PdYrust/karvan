package run

import (
	"time"

	"github.com/PdYrust/karvan/internal/migration/dest"
	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
)

// periods are the panels' reset periods in days (jobs/reset_user_data_usage.py).
var periods = map[panel.ResetStrategy]int{
	panel.ResetDay: 1, panel.ResetWeek: 7, panel.ResetMonth: 30, panel.ResetYear: 365,
}

// usageLog decides the one usage log row for a moved user. Both
// panels take the last reset from the newest log row (else created_at), so
// the row keeps the reset schedule and carries the traffic used before the
// last reset, keeping lifetime traffic right.
func usageLog(u panel.User, schedule string, now time.Time) *dest.UsageLog {
	carried := max(u.LifetimeUsed-u.UsedTraffic, 0)
	days, periodic := periods[u.ResetStrategy]
	if !periodic {
		if carried == 0 {
			return nil
		}
		return &dest.UsageLog{Traffic: carried, ResetAt: u.CreatedAt}
	}
	if schedule == plan.ResetRestart {
		return &dest.UsageLog{Traffic: carried, ResetAt: now}
	}
	// The panel resets when (now - last).days >= period, run hourly, so
	// the last reset is about created_at plus whole periods.
	period := time.Duration(days) * 24 * time.Hour
	k := now.Sub(u.CreatedAt) / period
	if k < 0 {
		k = 0
	}
	return &dest.UsageLog{Traffic: carried, ResetAt: u.CreatedAt.Add(k * period)}
}
