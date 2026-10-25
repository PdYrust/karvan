package bridge

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/secret"
)

// Deps are what a Bridge runs on. Tests replace the openers.
type Deps struct {
	DB     *sql.DB
	Key    *secret.Key
	Now    func() time.Time
	Log    *slog.Logger
	Ledger *ledger.Ledger
	// OpenDest opens a destination panel from its ENV path.
	OpenDest func(ctx context.Context, envPath string) (Destination, error)
	// OpenOld builds the client for a site's old panel.
	OpenOld func(s Site) (OldPanel, error)
}

// destTTL is how long an opened destination is reused before its ENV is
// read again (a changed DB password or secret shows up after this).
const destTTL = 10 * time.Minute

// recheckAfter is how old a token's last check against the old panel may
// get before the next request checks it again (D60).
const recheckAfter = 24 * time.Hour

// Bridge answers old links for every site.
type Bridge struct {
	Deps
	st     *store
	macKey []byte
	lim    *limits
	neg    *negative

	// reloadMu serializes Reload and stopAll, so two reloads cannot leave
	// a port in the wrong mode.
	reloadMu sync.Mutex
	mu       sync.Mutex
	sites    []Site
	dests    map[int64]*openDest // by panel id
	olds     map[int64]OldPanel  // by site id
	servers  map[int]*listener   // by port
	errs     map[int]string      // ports that could not listen
	wg       sync.WaitGroup
	// bg is cancelled when Run ends, which stops token lists in the background.
	bg     context.Context
	stopBg context.CancelFunc
	upMu   sync.Mutex
	ups    map[int64]*UploadProgress // by site id
	upWG   sync.WaitGroup            // token lists being checked
}

type openDest struct {
	d      Destination
	opened time.Time
}

// New returns a bridge. Call Reload to load the sites and listen.
func New(d Deps) *Bridge {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.Log == nil {
		d.Log = slog.Default()
	}
	if d.Ledger == nil {
		d.Ledger = ledger.New(d.DB)
	}
	if d.OpenDest == nil {
		d.OpenDest = openPanel
	}
	if d.OpenOld == nil {
		d.OpenOld = openOldPanel
	}
	bg, stop := context.WithCancel(context.Background())
	return &Bridge{Deps: d, st: &store{db: d.DB, key: d.Key, now: d.Now}, macKey: d.Key.Derive("bridge/token-mac"),
		lim: newLimits(d.Now), neg: newNegative(d.Now), dests: map[int64]*openDest{}, olds: map[int64]OldPanel{},
		servers: map[int]*listener{}, errs: map[int]string{}, bg: bg, stopBg: stop, ups: map[int64]*UploadProgress{}}
}

// Sites lists every site.
func (b *Bridge) Sites(ctx context.Context) ([]Site, error) { return b.st.Sites(ctx) }

// Site reads one site.
func (b *Bridge) Site(ctx context.Context, id int64) (Site, error) { return b.st.Site(ctx, id) }

// SaveSite adds or changes a site and starts serving it.
func (b *Bridge) SaveSite(ctx context.Context, s Site) (Site, error) {
	saved, err := b.st.Save(ctx, s)
	if err != nil {
		return saved, err
	}
	return saved, b.Reload(ctx)
}

// DeleteSite removes a site and the tokens verified for it.
func (b *Bridge) DeleteSite(ctx context.Context, id int64) error {
	if err := b.st.Delete(ctx, id); err != nil {
		return err
	}
	b.forgetUpload(id)
	return b.Reload(ctx)
}

// site returns the loaded site id.
func (b *Bridge) site(id int64) (Site, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, s := range b.sites {
		if s.ID == id {
			return s, true
		}
	}
	return Site{}, false
}

// dest returns the destination panel for a site, opened at most destTTL ago.
func (b *Bridge) dest(ctx context.Context, panelID int64) (Destination, error) {
	b.mu.Lock()
	o := b.dests[panelID]
	b.mu.Unlock()
	if o != nil && b.Now().Sub(o.opened) < destTTL {
		return o.d, nil
	}
	var env string
	if err := b.DB.QueryRowContext(ctx, "SELECT env_path FROM panels WHERE id = ?", panelID).Scan(&env); err != nil {
		return nil, fmt.Errorf("destination panel %d: %w", panelID, err)
	}
	d, err := b.OpenDest(ctx, env)
	if err != nil {
		return nil, err
	}
	b.mu.Lock()
	old := b.dests[panelID]
	b.dests[panelID] = &openDest{d: d, opened: b.Now()}
	b.mu.Unlock()
	if old != nil {
		time.AfterFunc(time.Minute, old.d.Close) // let requests still using it finish
	}
	return d, nil
}

// old returns the old panel client for a site.
func (b *Bridge) old(s Site) (OldPanel, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if o := b.olds[s.ID]; o != nil {
		return o, nil
	}
	o, err := b.OpenOld(s)
	if err != nil {
		return nil, err
	}
	b.olds[s.ID] = o
	return o, nil
}
