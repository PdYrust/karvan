package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/dbstats"
	"github.com/PdYrust/karvan/internal/discovery"
	"github.com/PdYrust/karvan/internal/hoststats"
	"github.com/PdYrust/karvan/internal/migration/reload"
	"github.com/PdYrust/karvan/internal/migration/run"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panelctl"
)

// ErrDataDirInUse means another Karvan process works on the data directory.
var ErrDataDirInUse = errors.New("another Karvan process is using the data directory; stop it first")

// credentialSweep is how often idle migration logins are dropped.
const credentialSweep = 10 * time.Minute

// dailyTrim is how often the events of long ended jobs (B17) and old
// audit events are trimmed.
const dailyTrim = 24 * time.Hour

// lockDataDir takes the data directory for this process (D30).
func (a *App) lockDataDir() (func(), error) {
	unlock, err := paneldb.LockFile(filepath.Join(a.Config.DataDir, "karvan.lock"))
	if errors.Is(err, paneldb.ErrLocked) {
		return nil, ErrDataDirInUse
	}
	return unlock, err
}

func (a *App) newRuns() *run.Service {
	ctl := panelctl.Opener{Mode: a.Config.PanelControl}
	return run.New(run.Deps{DB: a.DB, Key: a.Key, DataDir: a.Config.DataDir, Reloaders: reload.Methods(),
		Control: func(ctx context.Context, envPath string) (run.PanelControl, error) {
			c, err := ctl.Open(ctx, filepath.Dir(envPath))
			if err != nil {
				return nil, err
			}
			return c, nil
		}})
}

// services are the migration service, the bridge and the analytics
// samplers that serve runs next to the web panel.
type services struct {
	runs      *run.Service
	bridge    *bridge.Bridge
	host      *hoststats.Sampler
	hostStore *hoststats.Store
	dbstats   *dbstats.Sampler
	wg        sync.WaitGroup
}

// startServices settles migrations a crash interrupted, then starts the
// bridge, the analytics samplers and the login sweep until ctx ends.
func (a *App) startServices(ctx context.Context) *services {
	s := &services{runs: a.newRuns(), bridge: bridge.New(bridge.Deps{DB: a.DB, Key: a.Key, Log: a.Logger}), hostStore: hoststats.NewStore(a.DB)}
	s.host = &hoststats.Sampler{Reader: hoststats.System, Store: s.hostStore, Log: a.Logger}
	s.dbstats = &dbstats.Sampler{Store: dbstats.NewStore(a.DB), Log: a.Logger, Panels: s.panelRefs}
	if err := s.runs.Recover(ctx); err != nil {
		a.Logger.Warn("settle interrupted migrations", "err", err)
	}
	s.wg.Go(func() {
		if err := s.bridge.Run(ctx); err != nil {
			a.Logger.Warn("subscription bridge", "err", err)
		}
	})
	s.wg.Go(func() { s.host.Run(ctx) })
	s.wg.Go(func() { s.dbstats.Run(ctx) })
	s.wg.Go(func() { a.trimDaily(ctx, s.runs) })
	go func() {
		t := time.NewTicker(credentialSweep)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.runs.Sweep()
			}
		}
	}()
	return s
}

// trimDaily trims the events of long ended jobs and the audit events
// past audit.Kept at start and then daily until ctx ends.
func (a *App) trimDaily(ctx context.Context, runs *run.Service) {
	t := time.NewTicker(dailyTrim)
	defer t.Stop()
	for {
		a.trimmed(ctx, "job events", runs.TrimEvents)
		a.trimmed(ctx, "audit events", a.Audit.Trim)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// trimmed runs one trim and logs what it did.
func (a *App) trimmed(ctx context.Context, what string, trim func(context.Context) (int64, error)) {
	if n, err := trim(ctx); err != nil && ctx.Err() == nil {
		a.Logger.Warn("trim "+what, "err", err)
	} else if n > 0 {
		a.Logger.Info("trimmed "+what, "rows", n)
	}
}

// stop ends running jobs (Safe Mode undoes them) and waits for the bridge
// and the samplers.
func (s *services) stop() {
	s.runs.Shutdown()
	s.wg.Wait()
}

// panelRefs lists the registered panels for the database sampler.
func (s *services) panelRefs(ctx context.Context) ([]dbstats.PanelRef, error) {
	rows, err := s.runs.Panels(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dbstats.PanelRef, len(rows))
	for i, p := range rows {
		out[i] = dbstats.PanelRef{ID: p.ID, EnvPath: p.EnvPath, Name: p.Name}
	}
	return out, nil
}

// scanRoots are where panels are looked for.
func scanRoots() []string { return discovery.DefaultRoots }

// RestoreBackup puts a Safe Mode backup back from the command line. Karvan
// stops and starts the panel itself unless stopped says the operator
// already did. Karvan's server must not be running meanwhile.
func (a *App) RestoreBackup(ctx context.Context, id string, stopped bool) error {
	unlock, err := a.lockDataDir()
	if err != nil {
		return err
	}
	defer unlock()
	err = a.newRuns().RestoreBackup(ctx, id, stopped)
	outcome := audit.Success
	if err != nil {
		outcome = audit.Failure
	}
	if aerr := a.Audit.Record(ctx, audit.Event{Actor: audit.ActorCLI, Action: audit.ActionBackupRestore, Outcome: outcome,
		Detail: map[string]string{"backup": id}}); aerr != nil && err == nil {
		return fmt.Errorf("restored, but the audit log failed: %w", aerr)
	}
	return err
}
