package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/certorder"
	"github.com/PdYrust/karvan/internal/settings"
)

// domainWait is how long init waits for Let's Encrypt before it leaves
// the order to the service.
var domainWait = 3 * time.Minute

// errOrderSlow means the order was still running when init stopped
// waiting; the service resumes it.
var errOrderSlow = errors.New("the Let's Encrypt order is still running")

// orderFailed is a Let's Encrypt order that ended without a certificate.
type orderFailed struct{ detail string }

func (e *orderFailed) Error() string { return "the Let's Encrypt order failed: " + e.detail }

// panelDomain registers domain name and gets its certificate from Let's
// Encrypt by HTTP-01, the way the Domains page does, with the bridge and
// the order service run only for this. It returns the domain's id once
// the certificate is stored.
func (a *App) panelDomain(ctx context.Context, name, email string, panelPort int) (int64, error) {
	if email != "" {
		if err := a.Settings.SetACME(ctx, settings.ACME{Email: email}); err != nil {
			return 0, err
		}
	}
	ctx, cancel := context.WithCancel(ctx)
	br := bridge.New(bridge.Deps{DB: a.DB, Key: a.Key, Log: a.Logger, DataDir: a.Config.DataDir, KeyFile: a.Config.KeyFile,
		EnvRoots: scanRoots(), PanelPorts: func() []int { return []int{panelPort} }, ChallengePort: a.Config.ACMEHTTPPort})
	orders := a.newOrders(br)
	var wg sync.WaitGroup
	defer func() { cancel(); wg.Wait() }()
	wg.Go(func() {
		if err := br.Run(ctx); err != nil {
			a.Logger.Warn("subscription bridge", "err", err)
		}
	})
	select {
	case <-br.Started():
	case <-ctx.Done():
		return 0, ctx.Err()
	}
	wg.Go(func() { orders.Run(ctx) })
	d, err := br.AddDomain(ctx, name, bridge.CertChoice{Kind: bridge.ChoiceNone, PlainOK: true})
	if err != nil {
		return 0, err
	}
	o, err := orders.Start(ctx, d.ID, certorder.HTTP01, true)
	if err != nil {
		return d.ID, err
	}
	deadline := time.Now().Add(domainWait)
	for o.Running() {
		if time.Now().After(deadline) {
			return d.ID, errOrderSlow
		}
		select {
		case <-ctx.Done():
			return d.ID, ctx.Err()
		case <-time.After(time.Second):
		}
		if o, err = br.Order(ctx, o.ID); err != nil {
			return d.ID, err
		}
	}
	if o.State != "done" {
		return d.ID, &orderFailed{detail: orderReason(o.Error)}
	}
	return d.ID, nil
}

// orderReason is why an order failed, for people to read.
func orderReason(e *bridge.ACMEError) string {
	switch {
	case e == nil:
		return "no reason was given"
	case e.Detail != "":
		return e.Detail
	case e.Code == certorder.CodeUnreachable:
		return "Let's Encrypt could not be reached from this server"
	case e.Code == certorder.CodeRateLimited:
		return "too many certificates were asked for; try again later"
	case e.Code == certorder.CodeChallengeFailed:
		return "Let's Encrypt could not reach this server on the domain"
	}
	return "no reason was given"
}

// domainNotice says in plain words why the panel did not get the
// domain's certificate and what to do.
func domainNotice(name string, port int, err error) string {
	var failed *orderFailed
	var why string
	switch {
	case errors.Is(err, bridge.ErrPortBusy):
		why = fmt.Sprintf("Port %d is in use by another program, so Let's Encrypt cannot check %s. Stop that program or get the certificate on the Domains page with a DNS record.", port, name)
	case errors.Is(err, bridge.ErrPortDenied):
		why = fmt.Sprintf("Karvan may not listen on port %d, so Let's Encrypt cannot check %s.", port, name)
	case errors.Is(err, errOrderSlow):
		why = fmt.Sprintf("Let's Encrypt has not issued the certificate for %s yet. Karvan keeps trying; once it is issued, choose the domain in Settings, HTTPS.", name)
	case errors.As(err, &failed):
		why = fmt.Sprintf("Let's Encrypt did not issue a certificate for %s: %s. Check that the domain points at this server and port %d is open, then try again on the Domains page.", name, failed.detail, port)
	default:
		why = fmt.Sprintf("Could not get a certificate for %s: %v.", name, err)
	}
	return why + " Until then the panel uses Karvan's own certificate."
}
