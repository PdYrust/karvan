package run

import (
	"context"
	"fmt"
	"time"

	"github.com/PdYrust/karvan/internal/migration/engine"
)

// CheckSchema names the pre-flight report format (B7).
const CheckSchema = "karvan.check/v1"

// CheckReport is what a run of the current plan would do, for the
// operator to read, print or keep before anything runs (B7). Like the run
// report it holds no secret: no password, UUID or token.
type CheckReport struct {
	Schema      string    `json:"schema"`
	JobID       string    `json:"jobId"`
	PlanVersion int       `json:"planVersion"`
	CheckedAt   time.Time `json:"checkedAt"` // when the panels were read
	Source      struct {
		Kind   string `json:"kind"`
		Origin string `json:"origin"`
		Admin  string `json:"admin"`
	} `json:"source"`
	Destination struct {
		Kind string `json:"kind"`
		Name string `json:"name"`
	} `json:"destination"`
	Admin     engine.AdminSummary `json:"admin"`
	Totals    CheckTotals         `json:"totals"`
	Blockers  int                 `json:"blockers"`
	Stable    bool                `json:"stable"`
	Conflicts []engine.Conflict   `json:"conflicts"` // job-level
	Users     []CheckLine         `json:"users"`
}

// CheckTotals count the users by what the run would do with them.
type CheckTotals struct {
	Users         int `json:"users"`
	Moving        int `json:"moving"`
	Skipped       int `json:"skipped"`
	Adopted       int `json:"adopted"` // already on the destination: recorded, not written
	Renamed       int `json:"renamed"`
	ConfigChanged int `json:"configChanged"`
	Changed       int `json:"changed"` // moving with an operation or a fix applied
}

// CheckLine is one user as the run would leave them.
type CheckLine struct {
	Key            string            `json:"key"`
	SourceUsername string            `json:"sourceUsername"`
	Username       string            `json:"username"`
	Moving         bool              `json:"moving"`
	Adopted        bool              `json:"adopted"`
	Targets        []string          `json:"targets"`
	Applied        []string          `json:"applied"`
	Fixed          []string          `json:"fixed"`     // fixed conflicts, as "C02:suffix"
	Conflicts      []engine.Conflict `json:"conflicts"` // still open: type, severity and fix
	ChangesConfig  bool              `json:"changesConfig"`
	ChangesLink    bool              `json:"changesLink"`
}

// CheckReport is the pre-flight report of the job's current build.
func (s *Service) CheckReport(ctx context.Context, jobID string) (*CheckReport, error) {
	j, err := s.job(jobID)
	if err != nil {
		return nil, err
	}
	var name string
	if err := s.db.QueryRowContext(ctx, "SELECT name FROM panels WHERE id = ?", j.PanelID).Scan(&name); err != nil {
		return nil, fmt.Errorf("read the destination's name: %w", err)
	}
	if !j.lockIdle() {
		return nil, ErrBusy
	}
	defer j.mu.Unlock()
	if j.result == nil {
		return nil, ErrNotChecked
	}
	r := &CheckReport{Schema: CheckSchema, JobID: j.ID, PlanVersion: j.version, CheckedAt: j.checkedAt, Admin: j.result.Admin.Summary(),
		Blockers: j.result.Blockers, Stable: j.result.Stable, Conflicts: append([]engine.Conflict{}, j.result.Conflicts...), Users: []CheckLine{}}
	r.Source.Kind, r.Source.Origin = string(j.kind), j.origin
	if j.snapshot != nil {
		r.Source.Admin = j.snapshot.Admin.Username
	}
	r.Destination.Kind, r.Destination.Name = string(j.facts.Kind), name
	for _, it := range j.result.Items {
		r.Users = append(r.Users, checkLine(it))
		r.Totals.count(it)
	}
	return r, nil
}

func checkLine(it *engine.Item) CheckLine {
	l := CheckLine{Key: it.Key, SourceUsername: it.Source.Username, Username: it.User.Username, Moving: it.Include, Adopted: it.Adopt != nil,
		Targets: append([]string{}, it.Targets...), Applied: append([]string{}, it.Applied...), Fixed: []string{}, Conflicts: []engine.Conflict{},
		ChangesConfig: it.ChangesConfig, ChangesLink: it.ChangesLink}
	for _, c := range it.Resolved {
		l.Fixed = append(l.Fixed, c.Type+":"+c.Fix)
	}
	for _, c := range it.Conflicts {
		l.Conflicts = append(l.Conflicts, engine.Conflict{Type: c.Type, Severity: c.Severity, Fix: c.Fix})
	}
	return l
}

func (t *CheckTotals) count(it *engine.Item) {
	t.Users++
	switch {
	case it.Adopt != nil:
		t.Adopted++
	case !it.Include:
		t.Skipped++
	default:
		t.Moving++
	}
	if !it.Include {
		return
	}
	if it.ChangesConfig {
		t.ConfigChanged++
	}
	if it.User.Username != it.Source.Username {
		t.Renamed++
	}
	if len(it.Applied) > 0 {
		t.Changed++
	}
}
