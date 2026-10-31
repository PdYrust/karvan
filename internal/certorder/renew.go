package certorder

import (
	"context"
	"crypto/x509"
	"database/sql"
	"time"
)

// renewBefore is how long before expiry a certificate renews, unless a
// third of its life is shorter.
const renewBefore = 30 * 24 * time.Hour

// maxBackoff is the longest wait between two renewal tries.
const maxBackoff = 24 * time.Hour

// renewAt is when leaf renews: expiry minus 30 days, or minus a third of
// its life when that is shorter, so a short-lived certificate never
// renews in a loop.
func renewAt(leaf *x509.Certificate) time.Time {
	return leaf.NotAfter.Add(-min(renewBefore, leaf.NotAfter.Sub(leaf.NotBefore)/3))
}

// backoff is the wait after the tries-th failed renewal: 1, 2, 4 and so
// on hours, at most 24.
func backoff(tries int) time.Duration {
	d := time.Hour
	for i := 1; i < tries && d < maxBackoff; i++ {
		d *= 2
	}
	return min(d, maxBackoff)
}

// renewDue fails DNS-01 orders the CA has expired while they waited for
// Check, then starts a renewal for every due domain without a running
// order. A start that fails counts as a failed renewal.
func (s *Service) renewDue(ctx context.Context) {
	now := s.Now().Unix()
	var expired []orderRow
	err := s.each(ctx, orderRowColumns+" WHERE o.state = 'waiting_dns' AND o.expires_at > 0 AND o.expires_at <= ?", func(r *sql.Rows) error {
		o, err := scanOrderRow(r)
		expired = append(expired, o)
		return err
	}, now)
	if err != nil {
		s.Log.Warn("certorder: read waiting orders", "err", err)
	}
	for _, o := range expired {
		s.fail(ctx, o, errExpired)
	}
	type due struct {
		id           int64
		name, method string
	}
	var list []due
	err = s.each(ctx, "SELECT id, name, acme_method FROM domains d WHERE acme_method <> '' AND renew_at > 0 AND renew_at <= ? "+
		"AND NOT EXISTS (SELECT 1 FROM acme_orders o WHERE o.domain_id = d.id AND o.state IN "+runningStates+") ORDER BY id",
		func(r *sql.Rows) error {
			var d due
			err := r.Scan(&d.id, &d.name, &d.method)
			list = append(list, d)
			return err
		}, now)
	if err != nil {
		s.Log.Warn("certorder: read due renewals", "err", err)
	}
	for _, d := range list {
		if _, err := s.start(ctx, d.id, d.name, d.method, false, true); err != nil && ctx.Err() == nil {
			e, _ := classify(err, s.Now())
			s.renewFailed(ctx, d.id, d.name, e)
		}
	}
}

// each runs q with args and calls fn on every row.
func (s *Service) each(ctx context.Context, q string, fn func(*sql.Rows) error, args ...any) error {
	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := fn(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}
