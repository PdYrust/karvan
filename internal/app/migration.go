package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/discovery"
	"github.com/PdYrust/karvan/internal/migration/reload"
	"github.com/PdYrust/karvan/internal/migration/run"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
)

// ErrDataDirInUse means another Karvan process works on the data directory.
var ErrDataDirInUse = errors.New("another Karvan process is using the data directory; stop it first")

// credentialSweep is how often idle migration logins are dropped.
const credentialSweep = 10 * time.Minute

// lockDataDir takes the data directory for this process (D30).
func (a *App) lockDataDir() (func(), error) {
	unlock, err := paneldb.LockFile(filepath.Join(a.Config.DataDir, "karvan.lock"))
	if errors.Is(err, paneldb.ErrLocked) {
		return nil, ErrDataDirInUse
	}
	return unlock, err
}

func (a *App) newRuns() *run.Service {
	return run.New(run.Deps{DB: a.DB, Key: a.Key, DataDir: a.Config.DataDir, Reloaders: reload.Methods()})
}

// services are the migration service and the bridge that serve runs
// next to the web panel.
type services struct {
	runs   *run.Service
	bridge *bridge.Bridge
	done   chan struct{}
}

// startServices settles migrations a crash interrupted, then starts the
// bridge and the login sweep until ctx ends.
func (a *App) startServices(ctx context.Context) *services {
	s := &services{runs: a.newRuns(), bridge: bridge.New(bridge.Deps{DB: a.DB, Key: a.Key, Log: a.Logger}), done: make(chan struct{})}
	if err := s.runs.Recover(ctx); err != nil {
		a.Logger.Warn("settle interrupted migrations", "err", err)
	}
	go func() {
		defer close(s.done)
		if err := s.bridge.Run(ctx); err != nil {
			a.Logger.Warn("subscription bridge", "err", err)
		}
	}()
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

// stop ends running jobs (Safe Mode undoes them) and waits for the bridge.
func (s *services) stop() {
	s.runs.Shutdown()
	<-s.done
}

// scanRoots are where panels are looked for.
func scanRoots() []string { return discovery.DefaultRoots }

// RestoreBackup puts a Safe Mode backup back. The operator stops the panel
// first; Karvan must not be running, so no job writes meanwhile.
func (a *App) RestoreBackup(ctx context.Context, id string) error {
	unlock, err := a.lockDataDir()
	if err != nil {
		return err
	}
	defer unlock()
	err = a.newRuns().RestoreBackup(ctx, id)
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
