package bridge

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PdYrust/karvan/internal/secret"
)

// certSnap is a certificates row as Reload read it, without its key: a
// cached certificate is used only while it was read from the same
// values.
type certSnap struct {
	certFile, keyFile, pem string
	updated                int64
}

// shakeFacts is what a handshake needs besides the routes, read at Reload.
type shakeFacts struct {
	domains map[string]shakeDomain   // by name
	pins    map[int]map[string]int64 // port, domain name: certificate id
	certs   map[int64]certSnap       // certificates rows by id
}

// shakeDomain is a domain's certificate (0: none) and whether the domain
// fill has given it one yet.
type shakeDomain struct {
	certID int64
	filled bool
}

// portShake is how one TLS port picks its certificate.
type portShake struct {
	// legacy: a rule on the port is served from its own certificate, or a
	// domain its rules hold is not filled yet, so every handshake takes
	// the table-order pick, as before the move.
	legacy bool
	names  map[string]int64 // the domains its HTTPS rules hold: certificate id, the port's pin first
}

// shakeFacts reads every domain's certificate, every pin and every
// certificates row.
func (st *store) shakeFacts(ctx context.Context) (shakeFacts, error) {
	f := shakeFacts{domains: map[string]shakeDomain{}, pins: map[int]map[string]int64{}, certs: map[int64]certSnap{}}
	err := st.each(ctx, "SELECT name, COALESCE(certificate_id, 0), cert_filled FROM domains", func(r *sql.Rows) error {
		var name string
		var d shakeDomain
		err := r.Scan(&name, &d.certID, &d.filled)
		f.domains[name] = d
		return err
	})
	if err != nil {
		return f, fmt.Errorf("read domain certificates: %w", err)
	}
	err = st.each(ctx, "SELECT p.port, d.name, p.certificate_id FROM domain_pins p JOIN domains d ON d.id = p.domain_id",
		func(r *sql.Rows) error {
			var port int
			var name string
			var id int64
			err := r.Scan(&port, &name, &id)
			if f.pins[port] == nil {
				f.pins[port] = map[string]int64{}
			}
			f.pins[port][name] = id
			return err
		})
	if err != nil {
		return f, fmt.Errorf("read pins: %w", err)
	}
	err = st.each(ctx, "SELECT id, cert_file, key_file, cert_pem, updated_at FROM certificates", func(r *sql.Rows) error {
		var id int64
		var c certSnap
		err := r.Scan(&id, &c.certFile, &c.keyFile, &c.pem, &c.updated)
		f.certs[id] = c
		return err
	})
	if err != nil {
		return f, fmt.Errorf("read certificates: %w", err)
	}
	return f, nil
}

// each runs q with args and calls fn on every row.
func (st *store) each(ctx context.Context, q string, fn func(*sql.Rows) error, args ...any) error {
	rows, err := st.db.QueryContext(ctx, q, args...)
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

// buildShakes is the certificate pick of every TLS port of t.
func buildShakes(t table, f shakeFacts) map[int]portShake {
	out := map[int]portShake{}
	for port, routes := range t {
		sh := portShake{names: map[string]int64{}}
		for _, r := range routes {
			if r.site != nil {
				sh.legacy = true
			}
			for _, name := range r.domains {
				d, ok := f.domains[name]
				if !ok || !d.filled {
					sh.legacy = true
					continue
				}
				id := d.certID
				if pin, ok := f.pins[port][name]; ok {
					id = pin
				}
				if r.tls && id != 0 {
					sh.names[name] = id
				}
			}
		}
		out[port] = sh
	}
	return out
}

// certSource is what a handshake serves: a certificates row, or a panel
// rule's own certificate columns.
type certSource struct {
	id   int64
	snap certSnap
	site *Site
}

// pickCert picks the certificate for the SNI name (lower case, "" for
// none) on port: the domain the port's HTTPS rules hold with that
// exact name, else the "*." wildcard that covers it; the port's pin of
// that domain first, else its certificate. With no SNI or no such
// domain, and on a port not moved yet, it is the table-order pick.
func (b *Bridge) pickCert(port int, name string) (certSource, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if sh := b.shakes[port]; !sh.legacy && name != "" {
		id, ok := sh.names[name]
		if i := strings.IndexByte(name, '.'); !ok && i > 0 {
			id, ok = sh.names["*."+name[i+1:]]
		}
		if ok {
			return b.rowSource(id)
		}
	}
	r, ok := d53(b.routes[port], name)
	switch {
	case !ok:
		return certSource{}, false
	case r.certID != 0:
		return b.rowSource(r.certID)
	case r.site != nil:
		s := *r.site
		return certSource{site: &s}, true
	}
	return certSource{}, false
}

// rowSource is certificates row id as Reload read it. The caller holds mu.
func (b *Bridge) rowSource(id int64) (certSource, bool) {
	c, ok := b.certRows[id]
	return certSource{id: id, snap: c}, ok
}

// d53 is the table-order pick on one port's routes: the first in table order
// (panel rules by id, then redirect rules by id) whose domains cover
// name; else the first with no domains; else the first (a client without
// SNI, or an IP address).
func d53(routes []route, name string) (route, bool) {
	catchAll := -1
	for i, r := range routes {
		switch {
		case len(r.domains) == 0:
			if catchAll < 0 {
				catchAll = i
			}
		case name != "" && r.hostRank(name) != hostNone:
			return r, true
		}
	}
	switch {
	case catchAll >= 0:
		return routes[catchAll], true
	case len(routes) > 0:
		return routes[0], true
	}
	return route{}, false
}

// certCache holds loaded certificates. A row's certificate is read again
// when Reload read other values for it, and a file pair when the file
// changes on disk, so a renewal needs no restart.
type certCache struct {
	mu sync.Mutex
	m  map[string]cachedCert // "c" and the row id, or "s" and the site id
}

type cachedCert struct {
	cert    *tls.Certificate
	snap    certSnap
	modTime time.Time
}

// certificate is the GetCertificate of a TLS listener on port.
func (b *Bridge) certificate(port int) func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	cache := &certCache{m: map[string]cachedCert{}}
	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		name := strings.ToLower(hello.ServerName)
		src, ok := b.pickCert(port, name)
		switch {
		case !ok:
			if c, ok := b.panelCertificate(hello.Context(), port, name); ok {
				return c, nil
			}
			return nil, errors.New("no certificate for this name on this port")
		case src.site != nil:
			s := *src.site
			return cache.load(fmt.Sprintf("s%d", s.ID), certSnap{certFile: s.CertFile, keyFile: s.KeyFile, pem: s.CertPEM},
				func() (*tls.Certificate, certSnap, error) {
					c, err := b.readSiteCert(hello.Context(), s)
					return c, certSnap{certFile: s.CertFile, keyFile: s.KeyFile, pem: s.CertPEM}, err
				})
		}
		return cache.load(fmt.Sprintf("c%d", src.id), src.snap, func() (*tls.Certificate, certSnap, error) {
			return b.st.rowCert(hello.Context(), src.id)
		})
	}
}

// load is the cached certificate under key while it was read from snap
// and its file did not change, else read() again.
func (c *certCache) load(key string, snap certSnap, read func() (*tls.Certificate, certSnap, error)) (*tls.Certificate, error) {
	var mod time.Time
	if snap.certFile != "" {
		info, err := os.Stat(snap.certFile)
		if err != nil {
			return nil, err
		}
		mod = info.ModTime()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.m[key]; ok && e.snap == snap && e.modTime.Equal(mod) {
		return e.cert, nil
	}
	cert, got, err := read()
	if err != nil {
		return nil, err
	}
	c.m[key] = cachedCert{cert: cert, snap: got, modTime: mod}
	return cert, nil
}

// rowCert reads certificates row id: its certificate and key come from
// one read of the row, so a rewritten row never pairs an old certificate
// with a new key.
func (st *store) rowCert(ctx context.Context, id int64) (*tls.Certificate, certSnap, error) {
	var c certSnap
	var sealed []byte
	err := st.db.QueryRowContext(ctx, "SELECT cert_file, key_file, cert_pem, key_sealed, updated_at FROM certificates WHERE id = ?",
		id).Scan(&c.certFile, &c.keyFile, &c.pem, &sealed, &c.updated)
	if err != nil {
		return nil, c, fmt.Errorf("read certificate: %w", err)
	}
	if c.certFile != "" {
		cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)
		return &cert, c, err
	}
	if sealed == nil {
		return nil, c, errors.New("certificate has no key")
	}
	key, err := st.key.Open(sealed, secret.AAD("certificates", "key_sealed", id))
	if err != nil {
		return nil, c, fmt.Errorf("open certificate key: %w", err)
	}
	cert, err := tls.X509KeyPair([]byte(c.pem), key)
	return &cert, c, err
}

// readSiteCert reads a panel rule's own certificate, as stored before its
// adoption.
func (b *Bridge) readSiteCert(ctx context.Context, s Site) (*tls.Certificate, error) {
	if s.CertFile != "" {
		c, err := tls.LoadX509KeyPair(s.CertFile, s.KeyFile)
		return &c, err
	}
	key, err := b.st.siteKey(ctx, s.ID)
	if err != nil {
		return nil, err
	}
	c, err := tls.X509KeyPair([]byte(s.CertPEM), key)
	return &c, err
}
