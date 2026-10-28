package certorder

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"time"

	"github.com/PdYrust/karvan/internal/bridge"
)

// lookupTimeout bounds the TXT lookup of a Check (D68).
const lookupTimeout = 5 * time.Second

// maxSeen is how many found TXT values a refused Check shows.
const maxSeen = 5

// DNSNotFoundError means the TXT record does not hold the order's value
// yet (acme_dns_not_found); Seen are the values found, at most five.
type DNSNotFoundError struct{ Seen []string }

func (e *DNSNotFoundError) Error() string { return "the TXT record does not hold the value yet" }

// Check finishes a DNS-01 order once its TXT record is there (POST
// /domains/{id}/acme/check, D68). It looks the record up first, so a
// value not there yet costs no failed validation at the CA; force skips
// the lookup for split DNS.
func (s *Service) Check(ctx context.Context, id int64, force bool) (bridge.ACMEOrder, error) {
	if _, err := s.domainName(ctx, id); err != nil {
		return bridge.ACMEOrder{}, err
	}
	o, err := scanOrderRow(s.DB.QueryRowContext(ctx, orderRowColumns+" WHERE o.domain_id = ? AND o.state = 'waiting_dns'", id))
	if errors.Is(err, sql.ErrNoRows) {
		return bridge.ACMEOrder{}, ErrState
	}
	if err != nil {
		return bridge.ACMEOrder{}, err
	}
	if !force {
		seen := s.lookup(ctx, o.dnsName)
		if !slices.Contains(seen, o.dnsValue) {
			return bridge.ACMEOrder{}, &DNSNotFoundError{Seen: seen[:min(len(seen), maxSeen)]}
		}
	}
	if err := s.advance(ctx, o.id, stateWaitingDNS, "UPDATE acme_orders SET state = 'validating'"+setWhere); err != nil {
		if errors.Is(err, errGone) {
			return bridge.ACMEOrder{}, ErrState
		}
		return bridge.ACMEOrder{}, err
	}
	s.launch(o.id, id, "")
	return s.Bridge.Order(ctx, o.id)
}

// lookup reads the TXT values of name; a name without records, or a
// lookup that fails, has none.
func (s *Service) lookup(ctx context.Context, name string) []string {
	ctx, cancel := context.WithTimeout(ctx, lookupTimeout)
	defer cancel()
	vals, err := s.LookupTXT(ctx, name)
	if err != nil {
		return []string{}
	}
	return vals
}
