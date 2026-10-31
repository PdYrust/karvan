package certorder

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"golang.org/x/crypto/acme"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/bridge"
)

// errKeep means the certificate could not be stored: the order stays in
// issuing, with its key sealed, and resumes at the next start.
var errKeep = errors.New("the certificate is kept for the next start")

// drive works order id until it is done, failed or cancelled, or waits
// for its DNS record. At most talking orders run a step at once.
func (s *Service) drive(ctx context.Context, id int64) {
	o, err := s.load(ctx, id)
	if err != nil || o.state == stateWaitingDNS || !isRunning(o.state) {
		if err != nil && !errors.Is(err, errGone) && ctx.Err() == nil {
			s.Log.Error("certorder: read order", "order", id, "err", err)
		}
		return
	}
	select {
	case s.sem <- struct{}{}:
	case <-ctx.Done():
		return
	}
	defer func() { <-s.sem }()
	a, err := s.loadAccount(ctx, o.accountID)
	if err != nil {
		s.fail(ctx, o, err)
		return
	}
	c := s.client(a)
	for isRunning(o.state) && o.state != stateWaitingDNS {
		if !s.settle(ctx, o, s.step(ctx, c, &a, &o)) {
			return
		}
	}
}

// step runs the work of o's state.
func (s *Service) step(ctx context.Context, c *acme.Client, a *account, o *orderRow) error {
	switch o.state {
	case stateStarting:
		return s.begin(ctx, c, a, o)
	case stateValidating:
		return s.validate(ctx, c, o)
	}
	return s.issue(ctx, c, o)
}

// settle handles the end of a step and reports whether to go on.
func (s *Service) settle(ctx context.Context, o orderRow, err error) bool {
	switch {
	case ctx.Err() != nil, errors.Is(err, errGone):
		return false
	case errors.Is(err, errKeep):
		s.Log.Error("certorder: store certificate", "domain", o.domain, "err", err)
		return false
	case err != nil:
		s.fail(ctx, o, err)
		return false
	}
	return true
}

func isRunning(state string) bool {
	return slices.Contains([]string{stateStarting, stateWaitingDNS, stateValidating, stateIssuing}, state)
}

// begin registers the account if needed, orders the name and reads its
// authorization: HTTP-01 goes on to validating, DNS-01 waits for its
// record, an authorization the CA already holds goes to issuing.
func (s *Service) begin(ctx context.Context, c *acme.Client, a *account, o *orderRow) error {
	if err := s.register(ctx, c, a); err != nil {
		return err
	}
	ord, err := timed(ctx, s.call, func(ctx context.Context) (*acme.Order, error) {
		return c.AuthorizeOrder(ctx, acme.DomainIDs(o.domain))
	})
	if err != nil {
		return err
	}
	if len(ord.AuthzURLs) == 0 {
		return errors.New("the CA gave no authorization")
	}
	o.orderURL, o.authzURL = ord.URI, ord.AuthzURLs[0]
	if !ord.Expires.IsZero() {
		o.expiresAt = ord.Expires.Unix()
	}
	az, err := timed(ctx, s.call, func(ctx context.Context) (*acme.Authorization, error) { return c.GetAuthorization(ctx, o.authzURL) })
	if err != nil {
		return err
	}
	next := stateIssuing
	if az.Status != acme.StatusValid {
		ch := challengeOf(az, o.method)
		if ch == nil {
			return fmt.Errorf("the CA offers no %s challenge", o.method)
		}
		o.challengeURL, o.token, next = ch.URI, ch.Token, stateValidating
		if o.method == DNS01 {
			if o.dnsValue, err = c.DNS01ChallengeRecord(ch.Token); err != nil {
				return err
			}
			o.dnsName, next = "_acme-challenge."+strings.TrimPrefix(o.domain, "*."), stateWaitingDNS
		}
	}
	if err := s.advance(ctx, o.id, stateStarting, "UPDATE acme_orders SET state = ?, order_url = ?, authz_url = ?, challenge_url = ?, "+
		"token = ?, dns_name = ?, dns_value = ?, expires_at = ?"+setWhere, next, o.orderURL, o.authzURL, o.challengeURL, o.token, o.dnsName, o.dnsValue,
		o.expiresAt); err != nil {
		return err
	}
	o.state = next
	return nil
}

func challengeOf(az *acme.Authorization, method string) *acme.Challenge {
	for _, ch := range az.Challenges {
		if ch.Type == method {
			return ch
		}
	}
	return nil
}

// validate answers the challenge: HTTP-01 serves its token through the
// bridge while the CA checks it. Then the order goes to issuing and the
// challenge port is let go.
func (s *Service) validate(ctx context.Context, c *acme.Client, o *orderRow) error {
	if o.method == HTTP01 {
		ka, err := c.HTTP01ChallengeResponse(o.token)
		if err != nil {
			return err
		}
		s.Bridge.AddChallenge(o.token, o.domain, ka)
		defer s.Bridge.RemoveChallenge(o.token)
	}
	az, err := timed(ctx, s.call, func(ctx context.Context) (*acme.Authorization, error) { return c.GetAuthorization(ctx, o.authzURL) })
	if err != nil {
		return err
	}
	if ch := challengeOf(az, o.method); az.Status == acme.StatusPending && ch != nil && ch.Status == acme.StatusPending {
		if _, err := timed(ctx, s.call, func(ctx context.Context) (*acme.Challenge, error) {
			return c.Accept(ctx, &acme.Challenge{Type: o.method, URI: o.challengeURL, Token: o.token})
		}); err != nil {
			return err
		}
	}
	if _, err := timed(ctx, s.wait, func(ctx context.Context) (*acme.Authorization, error) {
		return c.WaitAuthorization(ctx, o.authzURL)
	}); err != nil {
		return err
	}
	if err := s.advance(ctx, o.id, stateValidating, "UPDATE acme_orders SET state = 'issuing'"+setWhere); err != nil {
		return err
	}
	o.state = stateIssuing
	s.letGo(o.id)
	return nil
}

// issue finalizes the order with its sealed key, downloads the chain and
// stores it through the bridge, then reloads so it is served.
func (s *Service) issue(ctx context.Context, c *acme.Client, o *orderRow) error {
	key, keyPEM, err := s.orderKey(ctx, o)
	if err != nil {
		return err
	}
	ord, err := timed(ctx, s.wait, func(ctx context.Context) (*acme.Order, error) { return c.WaitOrder(ctx, o.orderURL) })
	if err != nil {
		return err
	}
	var der [][]byte
	if ord.Status == acme.StatusValid {
		der, err = timed(ctx, s.call, func(ctx context.Context) ([][]byte, error) { return c.FetchCert(ctx, ord.CertURL, true) })
	} else {
		ord.URI = o.orderURL // a read of the order has no Location
		der, err = s.finalize(ctx, c, ord, o.domain, key)
	}
	if err != nil {
		return err
	}
	var chain strings.Builder
	for _, b := range der {
		_ = pem.Encode(&chain, &pem.Block{Type: "CERTIFICATE", Bytes: b})
	}
	leaf, err := x509.ParseCertificate(der[0])
	if err != nil {
		return err
	}
	if _, err := tls.X509KeyPair([]byte(chain.String()), keyPEM); err != nil || !slices.Contains(leaf.DNSNames, o.domain) {
		return errors.Join(errors.New("the CA issued a certificate for another key or name"), err)
	}
	err = s.Bridge.StoreACMECert(ctx, bridge.ACMEIssued{DomainID: o.domainID, OrderID: o.id, Method: o.method,
		ChainPEM: chain.String(), Key: keyPEM, RenewAt: renewAt(leaf)})
	switch {
	case errors.Is(err, bridge.ErrOrderGone):
		return errGone
	case err != nil:
		return fmt.Errorf("%w: %w", errKeep, err)
	}
	o.state = stateDone
	if err := s.Bridge.Reload(context.WithoutCancel(ctx)); err != nil {
		s.Log.Error("certorder: reload after a new certificate", "domain", o.domain, "err", err)
	}
	s.Log.Info("certorder: certificate stored", "domain", o.domain, "renewal", o.renewal, "expires", leaf.NotAfter.UTC())
	s.record(ctx, audit.ActionACMEIssued, audit.Success, map[string]string{"domain": o.domain,
		"expiry": leaf.NotAfter.UTC().Format(time.RFC3339)})
	return nil
}

// finalize sends the CSR and waits for the chain. The order's own URL is
// passed on, since a CA may answer finalize still processing and without
// a Location (Pebble and Let's Encrypt do).
func (s *Service) finalize(ctx context.Context, c *acme.Client, ord *acme.Order, name string, key *ecdsa.PrivateKey) ([][]byte, error) {
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{DNSNames: []string{name}}, key)
	if err != nil {
		return nil, err
	}
	return timed(ctx, s.wait, func(ctx context.Context) ([][]byte, error) {
		der, _, err := c.CreateCertFromOrder(ctx, ord, csr, true)
		return der, err
	})
}

// orderKey is the certificate key of order o, made and sealed in the
// order before finalize so a restart can finish it.
func (s *Service) orderKey(ctx context.Context, o *orderRow) (*ecdsa.PrivateKey, []byte, error) {
	if o.keySealed == nil {
		sealed, err := s.sealNewKey("acme_orders", o.id)
		if err != nil {
			return nil, nil, errAccount{err}
		}
		if err := s.advance(ctx, o.id, stateIssuing, "UPDATE acme_orders SET key_sealed = ?"+setWhere, sealed); err != nil {
			return nil, nil, err
		}
		o.keySealed = sealed
	}
	k, raw, err := s.openKey(o.keySealed, "acme_orders", o.id)
	if err != nil {
		s.Log.Error("certorder: open order key", "domain", o.domain, "err", err)
		return nil, nil, errAccount{err}
	}
	return k, raw, nil
}

// letGo lets go of order id's hold on the challenge port.
func (s *Service) letGo(id int64) {
	s.mu.Lock()
	var h string
	if j := s.jobs[id]; j != nil {
		h, j.holder = j.holder, ""
	}
	s.mu.Unlock()
	s.release(h)
}
