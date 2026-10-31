package certorder

import (
	"context"
	"database/sql"
	"errors"

	"golang.org/x/crypto/acme"
)

// resume works every running order again after a start: from its
// order URL, pending goes back to validating (HTTP-01, with the port) or
// waiting_dns, ready or valid goes on to issuing, and an order the CA no
// longer has, or has made invalid or expired, fails.
func (s *Service) resume(ctx context.Context) {
	var list []orderRow
	err := s.each(ctx, orderRowColumns+" WHERE o.state IN "+runningStates+" ORDER BY o.id", func(r *sql.Rows) error {
		o, err := scanOrderRow(r)
		list = append(list, o)
		return err
	})
	if err != nil {
		s.Log.Warn("certorder: read running orders", "err", err)
	}
	for _, o := range list {
		if err := s.resumeOne(ctx, o); err != nil && !errors.Is(err, errGone) && ctx.Err() == nil {
			s.Log.Warn("certorder: resume order", "domain", o.domain, "err", err)
			s.fail(ctx, o, err)
		}
	}
}

func (s *Service) resumeOne(ctx context.Context, o orderRow) error {
	if o.state == stateWaitingDNS {
		return nil // it waits for Check; renewDue fails it once the CA expires it
	}
	next := o.state
	if o.orderURL != "" {
		a, err := s.loadAccount(ctx, o.accountID)
		if err != nil {
			return err
		}
		ord, err := timed(ctx, s.call, func(ctx context.Context) (*acme.Order, error) { return s.client(a).GetOrder(ctx, o.orderURL) })
		var ae *acme.Error
		switch {
		case errors.As(err, &ae):
			return errExpired // the order URL is gone
		case err != nil:
			return err
		}
		switch ord.Status {
		case acme.StatusPending:
			next = stateValidating
			if o.method == DNS01 {
				next = stateWaitingDNS
			}
		case acme.StatusReady, acme.StatusProcessing, acme.StatusValid:
			next = stateIssuing
		default:
			return errExpired
		}
	}
	if next != o.state {
		if err := s.advance(ctx, o.id, o.state, "UPDATE acme_orders SET state = ?"+setWhere, next); err != nil {
			return err
		}
	}
	if next == stateWaitingDNS {
		return nil
	}
	holder := ""
	if o.method == HTTP01 && next != stateIssuing {
		holder = s.holder(o.domainID)
		if err := s.Bridge.WantChallengePort(ctx, holder, true); err != nil {
			return err
		}
	}
	s.launch(o.id, o.domainID, holder)
	return nil
}
