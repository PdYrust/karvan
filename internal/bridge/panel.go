package bridge

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
)

// ErrNoPanelCertificate means the domain the panel is served for has no
// certificate.
var ErrNoPanelCertificate = errors.New("the panel's domain has no certificate")

// PanelCertificate is the certificate of domain id for the web panel.
// It is read on each handshake through the same kind of cache as the
// rules' certificates, so a renewal, a new upload or a changed file is
// served without a restart.
func (b *Bridge) PanelCertificate(ctx context.Context, id int64) (*tls.Certificate, error) {
	var certID int64
	var c certSnap
	err := b.st.db.QueryRowContext(ctx, "SELECT c.id, c.cert_file, c.key_file, c.cert_pem, c.updated_at FROM domains d "+
		"JOIN certificates c ON c.id = d.certificate_id WHERE d.id = ?", id).Scan(&certID, &c.certFile, &c.keyFile, &c.pem, &c.updated)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, ErrNoPanelCertificate
	case err != nil:
		return nil, fmt.Errorf("read the panel certificate: %w", err)
	}
	return b.panelCerts.load(fmt.Sprintf("c%d", certID), c, func() (*tls.Certificate, certSnap, error) {
		return b.st.rowCert(ctx, certID)
	})
}

// PanelAnswers says whether domain id answers host (lower case, no port):
// its name, or one label under its wildcard, like a rule. hasCert is
// whether it has a certificate. ErrNotFound when the domain is gone.
func (b *Bridge) PanelAnswers(ctx context.Context, id int64, host string) (answers, hasCert bool, err error) {
	var name string
	err = b.st.db.QueryRowContext(ctx, "SELECT name, certificate_id IS NOT NULL FROM domains WHERE id = ?", id).Scan(&name, &hasCert)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return false, false, ErrNotFound
	case err != nil:
		return false, false, fmt.Errorf("read the panel domain: %w", err)
	}
	return matches(name, host), hasCert, nil
}

// RulePort is a port one rule answers on, with the domain names it holds
// there when it answers HTTPS.
type RulePort struct {
	Port    int
	HTTPS   bool
	Domains []string
}

// RulePorts lists the port of every rule, for the port proposal.
func (b *Bridge) RulePorts(ctx context.Context) ([]RulePort, error) {
	rules, err := b.st.rules(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RulePort, 0, len(rules))
	for _, r := range rules {
		out = append(out, RulePort{Port: r.port, HTTPS: r.tls, Domains: r.domains})
	}
	return out, nil
}
