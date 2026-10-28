package bridge

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"
)

// Let's Encrypt warnings on a domain (D50, D70).
const (
	WarnRenewalFailed     = "renewal_failed"      // the last renewal failed; the old certificate is served
	WarnRenewalWaitingDNS = "renewal_waiting_dns" // a DNS-01 renewal waits for its new TXT record and Check
)

// ErrOrderGone means the order was cancelled or deleted while it ran, so
// what it got is not stored.
var ErrOrderGone = errors.New("the Let's Encrypt order was cancelled")

// ACMEError is a Let's Encrypt failure in words: a D71 code and the CA's
// cleaned detail.
type ACMEError struct {
	Code    string     `json:"code"`
	Detail  string     `json:"detail"`
	RetryAt *time.Time `json:"retryAt,omitempty"`
}

// DNSRecord is the TXT record a DNS-01 order waits for (D68).
type DNSRecord struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ACMEOrder is an acme_orders row as the API shows it (D69).
type ACMEOrder struct {
	ID        int64      `json:"id"`
	Method    string     `json:"method"`
	Renewal   bool       `json:"renewal"`
	State     string     `json:"state"`
	DNS       *DNSRecord `json:"dns"`
	Error     *ACMEError `json:"error"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	ExpiresAt *time.Time `json:"expiresAt"`
	domainID  int64
}

// Running reports whether the order still works toward a certificate.
func (o ACMEOrder) Running() bool {
	return slices.Contains([]string{"starting", "waiting_dns", "validating", "issuing"}, o.State)
}

// Renewal is how a domain's Let's Encrypt certificate renews (D70).
type Renewal struct {
	Method    string     `json:"method"`
	NextAt    *time.Time `json:"nextAt"`
	Tries     int        `json:"tries"`
	LastError *ACMEError `json:"lastError"`
}

// DomainACME is a domain's Let's Encrypt state on the domains list.
type DomainACME struct {
	Method      string     `json:"method"`
	NextRenewal *time.Time `json:"nextRenewal"`
	Tries       int        `json:"tries"`
	LastError   *ACMEError `json:"lastError"`
	Order       *ACMEOrder `json:"order"`
}

// ACMEState is what GET /domains/{id}/acme answers: the latest order and
// the renewal, each null when there is none.
type ACMEState struct {
	Order   *ACMEOrder `json:"order"`
	Renewal *Renewal   `json:"renewal"`
}

// orderColumns read an acme_orders row as an ACMEOrder.
const orderColumns = "SELECT id, domain_id, method, renewal, state, dns_name, dns_value, error_code, error_detail, retry_at, " +
	"created_at, updated_at, expires_at FROM acme_orders"

func scanOrder(row interface{ Scan(...any) error }) (ACMEOrder, error) {
	var o ACMEOrder
	var dnsName, dnsValue, code, detail string
	var retry, created, updated, expires int64
	err := row.Scan(&o.ID, &o.domainID, &o.Method, &o.Renewal, &o.State, &dnsName, &dnsValue, &code, &detail, &retry, &created,
		&updated, &expires)
	if dnsName != "" {
		o.DNS = &DNSRecord{Name: dnsName, Value: dnsValue}
	}
	if code != "" {
		o.Error = &ACMEError{Code: code, Detail: detail, RetryAt: unixOrNil(retry)}
	}
	o.CreatedAt, o.UpdatedAt, o.ExpiresAt = time.Unix(created, 0).UTC(), time.Unix(updated, 0).UTC(), unixOrNil(expires)
	return o, err
}

func unixOrNil(s int64) *time.Time {
	if s == 0 {
		return nil
	}
	t := time.Unix(s, 0).UTC()
	return &t
}

// acmeRow is a domain's renewal columns and its latest order.
type acmeRow struct {
	renewal Renewal
	order   *ACMEOrder
}

// acmeRows reads the renewal columns of every domain, or of domain id,
// with each one's latest order.
func (st *store) acmeRows(ctx context.Context, id int64) (map[int64]acmeRow, error) {
	out := map[int64]acmeRow{}
	err := st.each(ctx, "SELECT id, acme_method, renew_at, renew_tries, renew_error FROM domains WHERE ? IN (0, id)",
		func(r *sql.Rows) error {
			var did, at int64
			var a acmeRow
			var lastErr string
			err := r.Scan(&did, &a.renewal.Method, &at, &a.renewal.Tries, &lastErr)
			a.renewal.NextAt, a.renewal.LastError = unixOrNil(at), decodeRenewError(lastErr)
			out[did] = a
			return err
		}, id)
	if err != nil {
		return nil, fmt.Errorf("read renewals: %w", err)
	}
	err = st.each(ctx, orderColumns+" o WHERE ? IN (0, domain_id) AND id = (SELECT max(id) FROM acme_orders WHERE domain_id = o.domain_id)",
		func(r *sql.Rows) error {
			o, err := scanOrder(r)
			a := out[o.domainID]
			a.order = &o
			out[o.domainID] = a
			return err
		}, id)
	if err != nil {
		return nil, fmt.Errorf("read orders: %w", err)
	}
	return out, nil
}

// decodeRenewError reads renew_error, stored as an ACMEError in JSON.
func decodeRenewError(s string) *ACMEError {
	if s == "" {
		return nil
	}
	var e ACMEError
	if json.Unmarshal([]byte(s), &e) != nil || e.Code == "" {
		return &ACMEError{Code: "acme_failed"}
	}
	return &e
}

// addACME sets d's Let's Encrypt state and warnings: shown while its
// certificate is from Let's Encrypt, or its latest order runs or is all
// it has.
func addACME(d *Domain, a acmeRow) {
	if a.renewal.Method == "" && (a.order == nil || (d.certID != 0 && !a.order.Running())) {
		return
	}
	d.ACME = &DomainACME{Method: a.renewal.Method, Tries: a.renewal.Tries, LastError: a.renewal.LastError, Order: a.order}
	if a.renewal.Method != "" {
		d.ACME.NextRenewal = a.renewal.NextAt
		if a.renewal.LastError != nil {
			d.Warnings = append(d.Warnings, WarnRenewalFailed)
		}
	}
	if o := a.order; o != nil && o.Renewal && o.State == "waiting_dns" {
		d.Warnings = append(d.Warnings, WarnRenewalWaitingDNS)
	}
	slices.Sort(d.Warnings)
}

// ACME is a domain's latest order and its renewal (D69, D70).
func (b *Bridge) ACME(ctx context.Context, id int64) (ACMEState, error) {
	rows, err := b.st.acmeRows(ctx, id)
	if err != nil {
		return ACMEState{}, err
	}
	a, ok := rows[id]
	if !ok {
		return ACMEState{}, ErrNotFound
	}
	out := ACMEState{Order: a.order}
	if a.renewal.Method != "" {
		r := a.renewal
		out.Renewal = &r
	}
	return out, nil
}

// Order reads one order.
func (b *Bridge) Order(ctx context.Context, id int64) (ACMEOrder, error) {
	o, err := scanOrder(b.DB.QueryRowContext(ctx, orderColumns+" WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return o, ErrNotFound
	}
	return o, err
}

// ACMEIssued is a certificate a Let's Encrypt order got.
type ACMEIssued struct {
	DomainID, OrderID int64
	Method            string // how it was got, for the renewal
	ChainPEM          string
	Key               []byte // PEM
	RenewAt           time.Time
}

// StoreACMECert stores what order in.OrderID got, in its done transaction
// (D69): a renewal rewrites the domain's Let's Encrypt row (cert_pem,
// key_sealed, updated_at); a first certificate is a new row with origin
// acme set as the domain's certificate as D65 does. The caller reloads.
func (b *Bridge) StoreACMECert(ctx context.Context, in ACMEIssued) error {
	if _, err := tls.X509KeyPair([]byte(in.ChainPEM), in.Key); err != nil {
		return fmt.Errorf("issued certificate: %w", err)
	}
	st := b.st
	st.saveMu.Lock()
	defer st.saveMu.Unlock()
	tx, err := st.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	var name, origin string
	var certID int64
	err = tx.QueryRowContext(ctx, "SELECT d.name, COALESCE(d.certificate_id, 0), COALESCE(c.origin, '') FROM domains d "+
		"LEFT JOIN certificates c ON c.id = d.certificate_id WHERE d.id = ?", in.DomainID).Scan(&name, &certID, &origin)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrOrderGone
	}
	if err != nil {
		return err
	}
	if leaf := firstCert([]byte(in.ChainPEM)); leaf == nil || !certCovers(leaf, name) {
		return fmt.Errorf("issued certificate does not cover %s", name)
	}
	now := st.now().Unix()
	if origin == OriginACME {
		err = st.rewriteACMECert(ctx, tx, certID, in, now)
	} else {
		err = st.newACMECert(ctx, tx, in, name)
	}
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE domains SET acme_method = ?, renew_at = ?, renew_tries = 0, renew_error = '', "+
		"updated_at = ? WHERE id = ?", in.Method, in.RenewAt.Unix(), now, in.DomainID); err != nil {
		return fmt.Errorf("set renewal: %w", err)
	}
	res, err := tx.ExecContext(ctx, "UPDATE acme_orders SET state = 'done', key_sealed = NULL, error_code = '', error_detail = '', "+
		"updated_at = ? WHERE id = ? AND state = 'issuing'", now, in.OrderID)
	if err != nil {
		return fmt.Errorf("finish order: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrOrderGone
	}
	return tx.Commit()
}

func (st *store) rewriteACMECert(ctx context.Context, tx *sql.Tx, id int64, in ACMEIssued, now int64) error {
	if _, err := tx.ExecContext(ctx, "UPDATE certificates SET cert_pem = ?, updated_at = ? WHERE id = ?", in.ChainPEM, now, id); err != nil {
		return fmt.Errorf("renew certificate: %w", err)
	}
	return st.sealCertKey(ctx, tx, id, in.Key)
}

// newACMECert adds the row and sets it as the domain's certificate (D65):
// its pins go and the HTTPS rules that hold it take it.
func (st *store) newACMECert(ctx context.Context, tx *sql.Tx, in ACMEIssued, domain string) error {
	id, err := st.insertChosen(ctx, tx, domain, "", certParts{certPEM: in.ChainPEM, key: in.Key})
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE certificates SET origin = 'acme' WHERE id = ?", id); err != nil {
		return fmt.Errorf("mark certificate: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE domains SET certificate_id = ?, cert_filled = 1 WHERE id = ?", id, in.DomainID); err != nil {
		return fmt.Errorf("set domain certificate: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM domain_pins WHERE domain_id = ?", in.DomainID); err != nil {
		return fmt.Errorf("delete pins: %w", err)
	}
	rules, err := st.rules(ctx)
	if err != nil {
		return err
	}
	for _, r := range rules {
		if r.tls && slices.Contains(r.domains, domain) {
			if err := rederive(ctx, tx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

// RenewFailed records a failed renewal of domain id (D70): one more try,
// the error, and the next try after backoff(tries).
func (b *Bridge) RenewFailed(ctx context.Context, id int64, e ACMEError, backoff func(tries int) time.Duration) error {
	raw, err := json.Marshal(ACMEError{Code: e.Code, Detail: e.Detail})
	if err != nil {
		return err
	}
	b.st.saveMu.Lock()
	defer b.st.saveMu.Unlock()
	var tries int
	if err := b.DB.QueryRowContext(ctx, "SELECT renew_tries FROM domains WHERE id = ?", id).Scan(&tries); err != nil {
		return fmt.Errorf("read renewal: %w", err)
	}
	now := b.Now()
	_, err = b.DB.ExecContext(ctx, "UPDATE domains SET renew_tries = ?, renew_error = ?, renew_at = ?, updated_at = ? WHERE id = ?",
		tries+1, string(raw), now.Add(backoff(tries+1)).Unix(), now.Unix(), id)
	if err != nil {
		return fmt.Errorf("record renewal: %w", err)
	}
	return nil
}
