package certorder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/netip"
	"strings"

	"golang.org/x/crypto/acme"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/bridge"
)

// orderRow is an acme_orders row with its domain's name.
type orderRow struct {
	id, domainID, accountID                                    int64
	domain, method, state                                      string
	renewal                                                    bool
	orderURL, authzURL, challengeURL, token, dnsName, dnsValue string
	keySealed                                                  []byte
	expiresAt                                                  int64
}

const orderRowColumns = "SELECT o.id, o.domain_id, o.account_id, d.name, o.method, o.state, o.renewal, o.order_url, o.authz_url, " +
	"o.challenge_url, o.token, o.dns_name, o.dns_value, o.key_sealed, o.expires_at FROM acme_orders o JOIN domains d ON d.id = o.domain_id"

func scanOrderRow(row interface{ Scan(...any) error }) (orderRow, error) {
	var o orderRow
	err := row.Scan(&o.id, &o.domainID, &o.accountID, &o.domain, &o.method, &o.state, &o.renewal, &o.orderURL, &o.authzURL,
		&o.challengeURL, &o.token, &o.dnsName, &o.dnsValue, &o.keySealed, &o.expiresAt)
	return o, err
}

func (s *Service) load(ctx context.Context, id int64) (orderRow, error) {
	o, err := scanOrderRow(s.DB.QueryRowContext(ctx, orderRowColumns+" WHERE o.id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return o, errGone
	}
	return o, err
}

// checkMethod checks a method for the domain name (D66).
func checkMethod(name, method string) error {
	switch {
	case method != HTTP01 && method != DNS01:
		return &bridge.ACMEInvalidError{Field: "method"}
	case isIP(name):
		return &bridge.ACMEInvalidError{Field: "name", Reason: "ip"}
	case method == HTTP01 && strings.HasPrefix(name, "*."):
		return &bridge.ACMEInvalidError{Field: "method", Reason: "wildcard_needs_dns"}
	}
	return nil
}

func isIP(name string) bool {
	_, err := netip.ParseAddr(name)
	return err == nil
}

// Start starts an order for domain id now, a first certificate or a
// renewal ahead of time (POST /domains/{id}/acme). HTTP-01 opens the
// challenge port first; a port failure starts nothing.
func (s *Service) Start(ctx context.Context, id int64, method string, agree bool) (bridge.ACMEOrder, error) {
	name, err := s.domainName(ctx, id)
	if err != nil {
		return bridge.ACMEOrder{}, err
	}
	oid, err := s.start(ctx, id, name, method, agree, false)
	if err != nil {
		return bridge.ACMEOrder{}, err
	}
	return s.Bridge.Order(ctx, oid)
}

// start checks the method, readies the account, holds the challenge port
// for HTTP-01 and stores the order in starting, then works it in the
// background.
func (s *Service) start(ctx context.Context, domainID int64, name, method string, agree, renewal bool) (int64, error) {
	if err := checkMethod(name, method); err != nil {
		return 0, err
	}
	var running bool
	if err := s.DB.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM acme_orders WHERE domain_id = ? AND state IN "+runningStates+")",
		domainID).Scan(&running); err != nil {
		return 0, fmt.Errorf("read orders: %w", err)
	}
	if running {
		return 0, ErrBusy
	}
	acct, err := s.orderAccount(ctx, domainID, agree, renewal)
	if err != nil {
		return 0, err
	}
	holder := ""
	if method == HTTP01 {
		holder = s.holder(domainID)
		if err := s.Bridge.WantChallengePort(ctx, holder, true); err != nil {
			return 0, err
		}
	}
	now := s.Now().Unix()
	res, err := s.DB.ExecContext(ctx, "INSERT INTO acme_orders (domain_id, account_id, method, renewal, state, created_at, updated_at) "+
		"VALUES (?, ?, ?, ?, 'starting', ?, ?)", domainID, acct, method, renewal, now, now)
	if err != nil {
		s.release(holder)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return 0, ErrBusy
		}
		return 0, fmt.Errorf("add order: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		s.release(holder)
		return 0, err
	}
	s.launch(id, domainID, holder)
	return id, nil
}

// orderAccount is the account a new order uses: a renewal keeps the
// account, and so the directory, of the domain's last certificate, since
// the staging switch changes new orders only (D73); a new order uses the
// directory in use now.
func (s *Service) orderAccount(ctx context.Context, domainID int64, agree, renewal bool) (int64, error) {
	if renewal {
		var id int64
		err := s.DB.QueryRowContext(ctx, "SELECT account_id FROM acme_orders WHERE domain_id = ? AND state = 'done' ORDER BY id DESC "+
			"LIMIT 1", domainID).Scan(&id)
		if !errors.Is(err, sql.ErrNoRows) {
			return id, err
		}
	}
	dir, err := s.directory(ctx)
	if err != nil {
		return 0, err
	}
	return s.prepareAccount(ctx, dir, agree)
}

// AddDomain adds a domain whose certificate comes from Let's Encrypt
// (CertChoice acme, D61): plain until the order stores it.
func (s *Service) AddDomain(ctx context.Context, name, method string, agree bool) (bridge.Domain, error) {
	n := bridge.NormalDomain(name)
	if n == "" {
		return bridge.Domain{}, &bridge.DomainInvalidError{Field: "name"}
	}
	if err := checkMethod(n, method); err != nil {
		return bridge.Domain{}, err
	}
	if method == HTTP01 { // a port failure creates nothing
		probe := s.holder(0)
		if err := s.Bridge.WantChallengePort(ctx, probe, true); err != nil {
			return bridge.Domain{}, err
		}
		defer s.release(probe)
	}
	d, err := s.Bridge.AddDomain(ctx, n, bridge.CertChoice{Kind: bridge.ChoiceNone, PlainOK: true})
	if err != nil {
		return d, err
	}
	if _, err := s.start(ctx, d.ID, n, method, agree, false); err != nil {
		if derr := s.Bridge.DeleteDomain(context.WithoutCancel(ctx), d.ID); derr != nil {
			s.Log.Warn("certorder: remove domain after a failed start", "domain", n, "err", derr)
		}
		return bridge.Domain{}, err
	}
	return s.Bridge.Domain(ctx, d.ID)
}

// Cancel cancels domain id's running order (DELETE /domains/{id}/acme)
// and, best effort, its authorization at the CA.
func (s *Service) Cancel(ctx context.Context, id int64) error {
	o, err := scanOrderRow(s.DB.QueryRowContext(ctx, orderRowColumns+" WHERE o.domain_id = ? AND o.state IN "+runningStates, id))
	if errors.Is(err, sql.ErrNoRows) {
		return bridge.ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := s.advance(ctx, o.id, o.state, "UPDATE acme_orders SET state = 'cancelled', key_sealed = NULL"+setWhere); err != nil {
		if errors.Is(err, errGone) {
			return bridge.ErrNotFound
		}
		return err
	}
	s.Forget(id)
	if o.authzURL == "" {
		return nil
	}
	s.wg.Go(func() {
		a, err := s.loadAccount(s.bg, o.accountID)
		if err != nil {
			return
		}
		c := s.client(a)
		if _, err := timed(s.bg, s.call, func(ctx context.Context) (struct{}, error) {
			return struct{}{}, c.RevokeAuthorization(ctx, o.authzURL)
		}); err != nil {
			s.Log.Info("certorder: revoke authorization", "domain", o.domain, "err", err)
		}
	})
	return nil
}

// setWhere ends every update of advance: the time, the order and the
// state it must still be in.
const setWhere = ", updated_at = ? WHERE id = ? AND state = ?"

// advance runs update q (an UPDATE acme_orders SET ... ending in setWhere)
// on order id while it is in state from, or returns errGone when it is
// no longer: cancelled or deleted.
func (s *Service) advance(ctx context.Context, id int64, from, q string, args ...any) error {
	res, err := s.DB.ExecContext(ctx, q, append(args, s.Now().Unix(), id, from)...)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errGone
	}
	return nil
}

// fail stores err on order o in words (D71). A failed renewal also counts
// on the domain and retries with the backoff (D70).
func (s *Service) fail(ctx context.Context, o orderRow, err error) {
	e, problem := classify(err, s.Now())
	e.Detail = s.redact(ctx, o, e.Detail)
	s.Log.Warn("certorder: order failed", "domain", o.domain, "code", e.Code, "problem", problem, "detail", e.Detail)
	var retry int64
	if e.RetryAt != nil {
		retry = e.RetryAt.Unix()
	}
	res, err := s.DB.ExecContext(ctx, "UPDATE acme_orders SET state = 'failed', error_code = ?, error_detail = ?, retry_at = ?, "+
		"key_sealed = NULL, updated_at = ? WHERE id = ? AND state IN "+runningStates, e.Code, e.Detail, retry, s.Now().Unix(), o.id)
	if err != nil {
		s.Log.Error("certorder: store order failure", "domain", o.domain, "err", err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 || !o.renewal {
		return
	}
	s.renewFailed(ctx, o.domainID, o.domain, e)
}

// redact leaves the order's account URL and key authorization, which a
// CA may quote, out of its detail (D71).
func (s *Service) redact(ctx context.Context, o orderRow, detail string) string {
	a, err := s.loadAccount(ctx, o.accountID)
	if err != nil || detail == "" {
		return detail
	}
	if a.url != "" {
		detail = strings.ReplaceAll(detail, a.url, "[account]")
	}
	if thumb, err := acme.JWKThumbprint(a.key.Public()); err == nil {
		detail = strings.ReplaceAll(detail, thumb, "[key]")
	}
	return detail
}

// renewFailed counts a failed renewal of the domain (D70).
func (s *Service) renewFailed(ctx context.Context, id int64, name string, e bridge.ACMEError) {
	if err := s.Bridge.RenewFailed(ctx, id, e, backoff); err != nil {
		s.Log.Error("certorder: store renewal failure", "domain", name, "err", err)
	}
	s.Log.Warn("certorder: renewal failed", "domain", name, "code", e.Code, "port", s.Bridge.ChallengePort)
	s.record(ctx, audit.ActionACMERenewFailed, audit.Failure, map[string]string{"domain": name, "code": e.Code})
}
