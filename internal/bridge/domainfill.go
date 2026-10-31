package bridge

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
)

// fillResult is what one domain fill did.
type fillResult struct {
	filled, pins int
	left         []leftDomain
}

// leftDomain is a domain the fill left, and the rule served from its own
// certificate that keeps it so.
type leftDomain struct{ domain, rule string }

// fillDomains gives every domain from before 0012 the certificate its
// rules serve it with, after the adoption and before the first
// Reload. A domain held by a rule still served from its own certificate
// stays unfilled, so its ports keep today's pick, and is tried again at
// the next start.
func (b *Bridge) fillDomains(ctx context.Context) {
	res, err := b.st.fillDomains(ctx)
	if err != nil {
		b.Log.Warn("bridge: fill domain certificates", "err", err)
		return
	}
	for _, l := range res.left {
		b.Log.Warn("bridge: domain certificate not filled", "domain", l.domain, "rule", l.rule)
	}
	if res.filled > 0 {
		b.Log.Info("bridge: domain certificates filled", "domains", res.filled, "pins", res.pins)
	}
}

// fillDomains fills every domain with cert_filled 0 in one transaction:
// its certificate is the table-order pick for its name on the TLS port
// of the first rule (table order) that holds it, and every other TLS port
// whose table-order pick differs gets a pin, so no handshake changes. Rules keep their
// stored certificate_id. Nothing else is written, so a second run with
// nothing left changes nothing.
func (st *store) fillDomains(ctx context.Context) (fillResult, error) {
	var res fillResult
	st.saveMu.Lock()
	defer st.saveMu.Unlock()
	tx, err := st.db.BeginTx(ctx, nil) // immediate: no other write until it ends
	if err != nil {
		return res, err
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	rules, err := st.rules(ctx)
	if err != nil {
		return res, err
	}
	type todo struct {
		id   int64
		name string
	}
	var list []todo
	err = st.each(ctx, "SELECT id, name FROM domains WHERE cert_filled = 0 ORDER BY id", func(r *sql.Rows) error {
		var d todo
		err := r.Scan(&d.id, &d.name)
		list = append(list, d)
		return err
	})
	if err != nil {
		return res, fmt.Errorf("read domains to fill: %w", err)
	}
	t := tableOf(rules)
	for _, d := range list {
		cert, pins, left := fillPlan(rules, t, d.name)
		if left != "" {
			res.left = append(res.left, leftDomain{domain: d.name, rule: left})
			continue
		}
		if _, err := tx.ExecContext(ctx, "UPDATE domains SET certificate_id = ?, cert_filled = 1 WHERE id = ? AND cert_filled = 0",
			nullID(cert), d.id); err != nil {
			return res, fmt.Errorf("fill domain: %w", err)
		}
		for port, id := range pins {
			if _, err := tx.ExecContext(ctx, "INSERT INTO domain_pins (port, domain_id, certificate_id) VALUES (?, ?, ?) "+
				"ON CONFLICT DO NOTHING", port, d.id, id); err != nil {
				return res, fmt.Errorf("pin domain: %w", err)
			}
		}
		res.filled, res.pins = res.filled+1, res.pins+len(pins)
	}
	return res, tx.Commit()
}

// fillPlan is the certificate and pins of the domain name, or the name of
// the rule that leaves it unfilled: one served from its own certificate
// that holds it or that a port holding it picks.
func fillPlan(rules []rule, t table, name string) (int64, map[int]int64, string) {
	var ports []int
	for _, r := range rules { // table order: panel rules by id, then redirect rules by id
		if !slices.Contains(r.domains, name) {
			continue
		}
		if r.site != nil {
			return 0, nil, r.ref.Name
		}
		if r.tls && !slices.Contains(ports, r.port) {
			ports = append(ports, r.port)
		}
	}
	var cert int64
	pins := map[int]int64{}
	for i, port := range ports {
		pick, _ := d53(t[port], sampleName(name, t[port]))
		switch {
		case pick.certID == 0:
			return 0, nil, pick.ref.Name
		case i == 0:
			cert = pick.certID
		case pick.certID != cert:
			pins[port] = pick.certID
		}
	}
	return cert, pins, ""
}

// nullID is id for a nullable id column: 0 is NULL.
func nullID(id int64) sql.NullInt64 { return sql.NullInt64{Int64: id, Valid: id != 0} }

// sampleName is a name that only the domain name covers among routes: the
// name itself, or for a "*." wildcard a host under it that no route
// holds by its exact name.
func sampleName(name string, routes []route) string {
	rest, ok := strings.CutPrefix(name, "*.")
	if !ok {
		return name
	}
	for i := 0; ; i++ {
		n := fmt.Sprintf("karvan-%d.%s", i, rest)
		if !slices.ContainsFunc(routes, func(r route) bool { return slices.Contains(r.domains, n) }) {
			return n
		}
	}
}

// tableOf is the routing table of rules, in their order.
func tableOf(rules []rule) table {
	t := table{}
	for _, r := range rules {
		t[r.port] = append(t[r.port], route{rule: r})
	}
	return t
}
