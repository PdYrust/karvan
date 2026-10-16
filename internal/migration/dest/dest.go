// Package dest opens a discovered destination panel and reads, in one
// read-only snapshot, everything a check needs: rules, existing users and
// credentials, admins, targets and the protocols each target serves.
package dest

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/PdYrust/karvan/internal/discovery"
	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzban"
	"github.com/PdYrust/karvan/internal/panel/marzbandb"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/pasarguard"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
)

// ErrNotReady means discovery found something missing for this panel.
var ErrNotReady = errors.New("the destination panel is not ready")

// Panel is an opened destination. Its DSN came from the panel's ENV just
// now and lives only in memory.
type Panel struct {
	Found  discovery.Panel
	Target paneldb.Target
}

// Open reads the panel's ENV again and refuses a panel with findings.
func Open(ctx context.Context, envPath string) (*Panel, error) {
	found := discovery.Inspect(ctx, envPath)
	if !found.Ready() {
		return &Panel{Found: found}, fmt.Errorf("%w: %v", ErrNotReady, found.Findings)
	}
	t, err := paneldb.ParseURL(found.Settings.DatabaseURL)
	if err != nil {
		return nil, err
	}
	return &Panel{Found: found, Target: t}, nil
}

// Facts reads the destination for a check. adminNames are the admin
// usernames the plan may use (the source name and any rename).
func (p *Panel) Facts(ctx context.Context, adminNames []string) (engine.Dest, error) {
	r, err := paneldb.OpenReader(ctx, p.Target)
	if err != nil {
		return engine.Dest{}, err
	}
	defer r.Close()
	var d engine.Dest
	err = r.View(ctx, func(q paneldb.Querier) error {
		var err error
		switch p.Found.Kind {
		case panel.Marzban:
			d, err = p.marzbanFacts(ctx, q, r.Server(), adminNames)
		case panel.PasarGuard:
			d, err = pasarguardFacts(ctx, q, r.Server(), adminNames)
		default:
			err = fmt.Errorf("unknown panel kind %q", p.Found.Kind)
		}
		return err
	})
	return d, err
}

func (p *Panel) marzbanFacts(ctx context.Context, q paneldb.Querier, srv paneldb.Server, admins []string) (engine.Dest, error) {
	s, err := marzbandb.Inspect(ctx, q, srv)
	if err != nil {
		return engine.Dest{}, err
	}
	rules := marzban.NewRules(s.UsersFoldCase)
	x, err := marzbandb.LoadIndex(ctx, q, rules)
	if err != nil {
		return engine.Dest{}, err
	}
	found, err := marzbandb.FindAdmins(ctx, q, admins)
	if err != nil {
		return engine.Dest{}, err
	}
	targets, err := marzban.Inbounds(p.Found.Settings.XrayConfig, p.Found.Settings.ExcludeInbounds)
	if err != nil {
		return engine.Dest{}, err
	}
	protos := map[string][]string{}
	for _, t := range targets {
		protos[t.ID] = []string{t.Protocol}
	}
	return engine.Dest{Kind: panel.Marzban, Rules: rules, Index: x, Admins: found, Targets: targets, TargetProtocols: protos}, nil
}

func pasarguardFacts(ctx context.Context, q paneldb.Querier, srv paneldb.Server, admins []string) (engine.Dest, error) {
	if _, err := pasarguarddb.Inspect(ctx, q, srv); err != nil {
		return engine.Dest{}, err
	}
	x, err := pasarguarddb.LoadIndex(ctx, q)
	if err != nil {
		return engine.Dest{}, err
	}
	found, err := pasarguarddb.FindAdmins(ctx, q, admins)
	if err != nil {
		return engine.Dest{}, err
	}
	targets, err := pasarguarddb.Targets(ctx, q, srv.Dialect)
	if err != nil {
		return engine.Dest{}, err
	}
	byTag, err := pasarguarddb.InboundProtocols(ctx, q)
	if err != nil {
		return engine.Dest{}, err
	}
	protos := map[string][]string{}
	for _, t := range targets {
		for _, tag := range t.Inbounds {
			if p := byTag[tag]; p != "" && !slices.Contains(protos[t.ID], p) {
				protos[t.ID] = append(protos[t.ID], p)
			}
		}
	}
	return engine.Dest{Kind: panel.PasarGuard, Rules: pasarguard.Rules{}, Index: x, Admins: found, Targets: targets, TargetProtocols: protos}, nil
}
