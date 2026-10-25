package bridge

import (
	"bytes"
	"context"
	"errors"
	"time"
)

// uploadTimeout bounds the checks of one token list in the background.
const uploadTimeout = 2 * time.Hour

// Upload states.
const (
	UploadNone    = "none"    // no list uploaded since Karvan started
	UploadRunning = "running" // checking
	UploadDone    = "done"    // every token checked
	UploadStopped = "stopped" // Karvan stopped, or the checks ran out of time
)

// ErrUploadBusy means a token list for the site is still being checked.
var ErrUploadBusy = errors.New("a token list for this site is still being checked")

// UploadProgress is a token list checked in the background. Checked counts
// the tokens done so far, out of Found.
type UploadProgress struct {
	State string `json:"state"`
	UploadResult
	Checked   int       `json:"checked"`
	StartedAt time.Time `json:"startedAt,omitzero"`
	EndedAt   time.Time `json:"endedAt,omitzero"`
}

// StartUpload reads a token list and checks it in the background, one
// list per site at a time. UploadStatus follows it. The list itself is
// not kept; only the progress, until Karvan stops.
func (b *Bridge) StartUpload(siteID int64, list []byte) (UploadProgress, error) {
	u, err := b.prepareUpload(b.bg, siteID, bytes.NewReader(list))
	if err != nil {
		return UploadProgress{}, err
	}
	p := &UploadProgress{State: UploadRunning, StartedAt: b.Now()}
	p.Found, p.TooLarge = len(u.tokens), u.tooLarge
	b.upMu.Lock()
	if old := b.ups[siteID]; old != nil && old.State == UploadRunning {
		b.upMu.Unlock()
		return UploadProgress{}, ErrUploadBusy
	}
	b.ups[siteID] = p
	first := *p
	b.upMu.Unlock()
	b.upWG.Add(1)
	go func() {
		defer b.upWG.Done()
		ctx, cancel := context.WithTimeout(b.bg, uploadTimeout)
		defer cancel()
		b.checkTokens(ctx, u, func(o uploadOutcome) {
			b.upMu.Lock()
			count(&p.UploadResult, o)
			p.Checked++
			b.upMu.Unlock()
		})
		b.upMu.Lock()
		p.State, p.EndedAt = UploadDone, b.Now()
		if ctx.Err() != nil {
			p.State = UploadStopped
		}
		res := p.UploadResult
		b.upMu.Unlock()
		b.Log.Info("bridge: token list checked", "site", u.site.Name, "found", res.Found, "stored", res.Stored,
			"failed", res.Failed)
	}()
	return first, nil
}

// UploadStatus is the latest token list of a loaded site.
func (b *Bridge) UploadStatus(siteID int64) (UploadProgress, error) {
	if _, ok := b.site(siteID); !ok {
		return UploadProgress{}, ErrNotFound
	}
	b.upMu.Lock()
	defer b.upMu.Unlock()
	if p := b.ups[siteID]; p != nil {
		return *p, nil
	}
	return UploadProgress{State: UploadNone}, nil
}

// forgetUpload drops a deleted site's progress, since its id can come back.
func (b *Bridge) forgetUpload(siteID int64) {
	b.upMu.Lock()
	delete(b.ups, siteID)
	b.upMu.Unlock()
}
