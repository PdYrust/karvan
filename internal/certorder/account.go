package certorder

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/crypto/acme"

	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/secret"
)

// account is an acme_accounts row with its key opened.
type account struct {
	id             int64
	directory, url string
	key            *ecdsa.PrivateKey
}

// errAccount wraps a sealing or key failure of an account, which the
// order shows as acme_account (D71).
type errAccount struct{ err error }

func (e errAccount) Error() string { return "account key: " + e.err.Error() }
func (e errAccount) Unwrap() error { return e.err }

// prepareAccount returns the account of dir, making its row and key the
// first time: inserted with a NULL key, then sealed under its new id in
// the same transaction (D66). Until the CA has registered it, an order
// must agree to the subscriber agreement.
func (s *Service) prepareAccount(ctx context.Context, dir string, agree bool) (int64, error) {
	var id int64
	var url string
	var sealed []byte
	err := s.DB.QueryRowContext(ctx, "SELECT id, url, key_sealed FROM acme_accounts WHERE directory = ?", dir).Scan(&id, &url, &sealed)
	switch {
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		return 0, fmt.Errorf("read account: %w", err)
	case sealed != nil && url != "":
		return id, nil
	case !agree:
		return 0, &bridge.ACMEInvalidError{Field: "agreeTos"}
	case sealed != nil:
		return id, nil
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	if id == 0 {
		res, err := tx.ExecContext(ctx, "INSERT INTO acme_accounts (directory, created_at) VALUES (?, ?)", dir, s.Now().Unix())
		if err != nil {
			return 0, fmt.Errorf("add account: %w", err)
		}
		if id, err = res.LastInsertId(); err != nil {
			return 0, err
		}
	}
	if sealed, err = s.sealNewKey("acme_accounts", id); err != nil {
		return 0, errAccount{err}
	}
	if _, err := tx.ExecContext(ctx, "UPDATE acme_accounts SET key_sealed = ?, url = '' WHERE id = ?", sealed, id); err != nil {
		return 0, fmt.Errorf("store account key: %w", err)
	}
	return id, tx.Commit()
}

// sealNewKey makes an ECDSA P-256 key and seals its PEM under (table,
// key_sealed, id).
func (s *Service) sealNewKey(table string, id int64) ([]byte, error) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	der, err := x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		return nil, err
	}
	return s.Key.Seal(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), secret.AAD(table, "key_sealed", id))
}

// openKey opens a key sealed by sealNewKey.
func (s *Service) openKey(sealed []byte, table string, id int64) (*ecdsa.PrivateKey, []byte, error) {
	raw, err := s.Key.Open(sealed, secret.AAD(table, "key_sealed", id))
	if err != nil {
		return nil, nil, err
	}
	b, _ := pem.Decode(raw)
	if b == nil {
		return nil, nil, errors.New("no key in the sealed value")
	}
	k, err := x509.ParsePKCS8PrivateKey(b.Bytes)
	if err != nil {
		return nil, nil, err
	}
	ec, ok := k.(*ecdsa.PrivateKey)
	if !ok {
		return nil, nil, errors.New("the sealed key is not ECDSA")
	}
	return ec, raw, nil
}

// loadAccount reads account id with its key; a row whose key is missing
// or cannot be opened fails the order as acme_account.
func (s *Service) loadAccount(ctx context.Context, id int64) (account, error) {
	a := account{id: id}
	var sealed []byte
	err := s.DB.QueryRowContext(ctx, "SELECT directory, url, key_sealed FROM acme_accounts WHERE id = ?", id).
		Scan(&a.directory, &a.url, &sealed)
	if err != nil {
		return a, errAccount{fmt.Errorf("read account %d: %w", id, err)}
	}
	if sealed == nil {
		return a, errAccount{errors.New("the account has no key")}
	}
	if a.key, _, err = s.openKey(sealed, "acme_accounts", id); err != nil {
		s.Log.Error("certorder: open account key", "account", id, "err", err)
		return a, errAccount{err}
	}
	return a, nil
}

// client talks to a's directory as a.
func (s *Service) client(a account) *acme.Client {
	c := &acme.Client{Key: a.key, DirectoryURL: a.directory, HTTPClient: s.HTTPClient, UserAgent: "karvan",
		RetryBackoff: func(n int, _ *http.Request, res *http.Response) time.Duration {
			if n > 3 || (res != nil && res.StatusCode == http.StatusTooManyRequests) {
				return -1 // a rate limit is shown in words with its time, not waited out
			}
			return time.Duration(n) * s.retry
		}}
	if a.url != "" {
		c.KID = acme.KeyID(a.url)
	}
	return c
}

// register makes the CA know account a, agreeing to its subscriber
// agreement, which the first order of a directory asked for (D66).
func (s *Service) register(ctx context.Context, c *acme.Client, a *account) error {
	if a.url != "" {
		return nil
	}
	st, err := s.Settings.ACME(ctx)
	if err != nil {
		return err
	}
	acct := &acme.Account{}
	if st.Email != "" {
		acct.Contact = []string{"mailto:" + st.Email}
	}
	got, err := timed(ctx, s.call, func(ctx context.Context) (*acme.Account, error) { return c.Register(ctx, acct, acme.AcceptTOS) })
	switch {
	case errors.Is(err, acme.ErrAccountAlreadyExists):
		a.url = string(c.KID)
	case err != nil:
		return err
	default:
		a.url = got.URI
	}
	if a.url == "" {
		return errAccount{errors.New("the CA gave no account URL")}
	}
	c.KID = acme.KeyID(a.url)
	if _, err := s.DB.ExecContext(ctx, "UPDATE acme_accounts SET url = ?, email = ? WHERE id = ?", a.url, st.Email, a.id); err != nil {
		return fmt.Errorf("store account: %w", err)
	}
	return nil
}
