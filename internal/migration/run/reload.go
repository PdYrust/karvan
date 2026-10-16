package run

import (
	"context"

	"github.com/PdYrust/karvan/internal/panel"
)

// Reload methods (D46). The operator picks one on the final step.
const (
	ReloadManual = "manual"       // show the restart command
	ReloadTouch  = "touch"        // a no-op edit per user through the panel API
	ReloadCore   = "core_restart" // one core restart through the panel API
)

// ReloadTarget is a written user the panel must pick up.
type ReloadTarget struct {
	ID       int64
	Username string
}

// ReloadResult says what the panel picked up. A reload never rolls back
// data (D48).
type ReloadResult struct {
	Method  string   `json:"method"`
	OK      int      `json:"ok"`
	Failed  []string `json:"failed,omitempty"`
	Command string   `json:"command,omitempty"` // for the operator to run
}

// ReloadInput is what a reloader gets.
type ReloadInput struct {
	Kind    panel.Kind
	EnvPath string
	Admin   ReloadTarget // the moved admin, whose token the API calls use
	Users   []ReloadTarget
}

// Reloader makes a running panel pick up rows written behind its back.
type Reloader interface {
	Reload(ctx context.Context, in ReloadInput) ReloadResult
}

type manualReload struct{}

func (manualReload) Reload(_ context.Context, in ReloadInput) ReloadResult {
	cmd := "marzban restart"
	if in.Kind == panel.PasarGuard {
		cmd = "pasarguard restart"
	}
	return ReloadResult{Method: ReloadManual, Command: cmd}
}

// reload runs the chosen method on what the job wrote. Its result goes to
// the report; a failure never undoes data.
func (s *Service) reload(ctx context.Context, j *Job, rep *Report, method string) {
	_ = s.stage(j, rep, StageReload, func() error {
		rows, err := s.ledger.Job(ctx, j.ID, "done")
		if err != nil {
			return err
		}
		in := ReloadInput{Kind: j.facts.Kind, EnvPath: j.envPath,
			Admin: ReloadTarget{ID: rep.Admin.ID, Username: rep.Admin.Username}}
		for _, e := range rows {
			if e.Kind == "user" {
				in.Users = append(in.Users, ReloadTarget{ID: e.Row.ID, Username: e.Username})
			}
		}
		rep.Reload = s.Reloaders[method].Reload(ctx, in)
		rep.NotUndoable = append(rep.NotUndoable, UndoAfterReload)
		return nil
	})
}
