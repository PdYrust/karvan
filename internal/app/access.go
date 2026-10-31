package app

import (
	"context"

	"github.com/PdYrust/karvan/internal/audit"
)

// OpenIP turns "Only open Karvan on its domain" off, until it is turned on
// again from the panel: every name and the server's IP open the panel. A
// running server takes it within a few seconds. It reports whether the
// switch was on.
func (a *App) OpenIP(ctx context.Context) (bool, error) {
	h, err := a.Settings.HTTP(ctx)
	if err != nil {
		return false, err
	}
	t := h.TLS
	if t.OpenIP {
		return false, nil
	}
	t.OpenIP, t.Armed = true, false
	if err := a.Settings.SetKarvan(ctx, t); err != nil {
		return false, err
	}
	return true, a.Audit.Record(ctx, audit.Event{Actor: audit.ActorCLI, Action: audit.ActionAccessOpenIP, Outcome: audit.Success})
}
