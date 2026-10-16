package run

import (
	"context"
	"errors"

	"github.com/PdYrust/karvan/internal/migration/dest"
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
	ID        int64
	Username  string
	WireGuard bool // PasarGuard gives it a peer IP only on its own edit path
}

// ReloadResult says what the panel picked up. A reload never rolls back
// data (D48).
type ReloadResult struct {
	Method  string   `json:"method"`
	OK      int      `json:"ok"`
	Failed  []string `json:"failed,omitempty"`
	Error   string   `json:"error,omitempty"`   // why the method could not run
	Command string   `json:"command,omitempty"` // for the operator to run
}

// ReloadInput is what a reloader gets.
type ReloadInput struct {
	Kind  panel.Kind
	API   dest.API // empty for the manual method
	Users []ReloadTarget
}

// Reloader makes a running panel pick up rows written behind its back.
type Reloader interface {
	Reload(ctx context.Context, in ReloadInput) ReloadResult
}

type manualReload struct{}

func (manualReload) Reload(_ context.Context, in ReloadInput) ReloadResult {
	return ReloadResult{Method: ReloadManual, Command: RestartCommand(in.Kind)}
}

// RestartCommand is the panel's own restart command.
func RestartCommand(k panel.Kind) string {
	if k == panel.PasarGuard {
		return "pasarguard restart"
	}
	return "marzban restart"
}

// reload runs the chosen method on what the job wrote. Its result goes to
// the report; a failure never undoes data.
func (s *Service) reload(ctx context.Context, j *Job, rep *Report, method string) {
	_ = s.stage(j, rep, StageReload, func() error {
		rows, err := s.ledger.Job(ctx, j.ID, "done")
		if err != nil {
			return err
		}
		wg := map[string]bool{}
		for _, it := range j.result.Items {
			wg[it.Key] = it.User.Credentials.WireGuard != nil
		}
		in := ReloadInput{Kind: j.facts.Kind}
		for _, e := range rows {
			if e.Kind == "user" {
				in.Users = append(in.Users, ReloadTarget{ID: e.Row.ID, Username: e.Username, WireGuard: wg[e.SourceKey]})
			}
		}
		if method != ReloadManual {
			api, err := s.panelAPI(ctx, j.envPath)
			if err != nil {
				rep.Reload = ReloadResult{Method: method, Error: err.Error(), Command: RestartCommand(in.Kind)}
				return nil
			}
			in.API = api
		}
		rep.Reload = s.Reloaders[method].Reload(ctx, in)
		rep.NotUndoable = append(rep.NotUndoable, UndoAfterReload)
		return nil
	})
}

// ReloadOption is one reload method as the final step shows it (owner
// decision Q5: every method is shown and the operator picks).
type ReloadOption struct {
	Method    string   `json:"method"`
	Available bool     `json:"available"`
	Reason    string   `json:"reason,omitempty"`  // why it is not available, a code
	Detail    string   `json:"detail,omitempty"`  // the error behind Reason
	Effects   []string `json:"effects"`           // codes the web panel explains
	Command   string   `json:"command,omitempty"` // manual only
}

// Effect codes, checked on the running panels (test/panels).
const (
	EffectOperatorRuns     = "operator_runs_command"   // nothing happens until someone restarts the panel
	EffectWireGuardNoIP    = "wireguard_users_no_ip"   // PasarGuard gives peer IPs only on its own edit path
	EffectEditPerUser      = "one_edit_per_user"       // a no-op edit through the API, a few per second
	EffectNotifyPerUser    = "notification_per_user"   // the panel sends "user modified" for each one
	EffectShortDisconnect  = "short_disconnect_all"    // every user drops for the seconds a restart takes
	EffectWireGuardTouched = "wireguard_users_touched" // PasarGuard WireGuard users get the edit first
)

// ReloadOptions lists every reload method for a checked job, with what
// each does on this panel and whether Karvan can run it.
func (s *Service) ReloadOptions(ctx context.Context, jobID string) ([]ReloadOption, error) {
	j, err := s.job(jobID)
	if err != nil {
		return nil, err
	}
	if j.running.Load() {
		return nil, ErrBusy
	}
	j.mu.Lock()
	kind, env, wg := j.facts.Kind, j.envPath, false
	if j.result != nil {
		for _, it := range j.result.Items {
			wg = wg || (it.Include && it.Adopt == nil && it.User.Credentials.WireGuard != nil)
		}
	}
	j.mu.Unlock()
	wg = wg && kind == panel.PasarGuard
	manual := ReloadOption{Method: ReloadManual, Available: true, Command: RestartCommand(kind), Effects: []string{EffectOperatorRuns}}
	touch := ReloadOption{Method: ReloadTouch, Effects: []string{EffectEditPerUser, EffectNotifyPerUser}}
	core := ReloadOption{Method: ReloadCore, Effects: []string{EffectShortDisconnect}}
	if wg {
		manual.Effects = append(manual.Effects, EffectWireGuardNoIP)
		core.Effects = append(core.Effects, EffectWireGuardTouched, EffectNotifyPerUser)
	}
	_, apiErr := s.panelAPI(ctx, env)
	for _, o := range []*ReloadOption{&touch, &core} {
		_, ok := s.Reloaders[o.Method]
		switch {
		case !ok:
			o.Reason = "not_built_in"
		case errors.Is(apiErr, dest.ErrNoFullAdmin):
			o.Reason = "no_full_admin"
		case apiErr != nil:
			o.Reason, o.Detail = "panel_unreadable", apiErr.Error()
		default:
			o.Available = true
		}
	}
	return []ReloadOption{manual, touch, core}, nil
}

func (s *Service) panelAPI(ctx context.Context, envPath string) (dest.API, error) {
	d, err := dest.Open(ctx, envPath)
	if err != nil {
		return dest.API{}, err
	}
	return d.API(ctx)
}
