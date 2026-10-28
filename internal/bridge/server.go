package bridge

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// listener is one port the bridge serves.
type listener struct {
	srv *http.Server
	tls bool
}

// Run adopts the sites' own certificates, then serves every site and
// sweeps hourly until ctx ends.
func (b *Bridge) Run(ctx context.Context) error {
	b.adoptCertificates(ctx)
	if err := b.Reload(ctx); err != nil {
		return err
	}
	t := time.NewTicker(SweepEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			b.stopBg()
			b.stopAll()
			b.Wait()
			return nil
		case <-t.C:
			if n, err := b.Sweep(ctx); err != nil {
				b.Log.Warn("bridge: sweep", "err", err)
			} else if n > 0 {
				b.Log.Info("bridge: sweep closed users", "count", n)
			}
		}
	}
}

func siteRules(sites []Site) []rule {
	out := make([]rule, len(sites))
	for i, s := range sites {
		out[i] = ruleOf(s)
	}
	return out
}

// Reload reads the panel and redirect rules again and listens on exactly
// their ports.
func (b *Bridge) Reload(ctx context.Context) error {
	b.reloadMu.Lock()
	defer b.reloadMu.Unlock()
	sites, err := b.st.Sites(ctx)
	if err != nil {
		return err
	}
	redirects, err := b.st.redirectRules(ctx)
	if err != nil {
		return err
	}
	want := map[int]bool{} // port -> TLS
	for _, r := range append(siteRules(sites), redirects...) {
		if tlsOn, ok := want[r.port]; ok && tlsOn != r.tls {
			return fmt.Errorf("port %d has rules with and without a certificate", r.port)
		}
		want[r.port] = r.tls
	}
	b.mu.Lock()
	b.load(sites, redirects)
	b.olds = map[int64]OldPanel{}
	routes := b.routes
	for port := range b.errs {
		if _, ok := want[port]; !ok {
			delete(b.errs, port) // no rule uses the port any more
		}
	}
	var stop []*listener
	for port, l := range b.servers {
		if tlsOn, ok := want[port]; !ok || tlsOn != l.tls {
			stop = append(stop, l)
			delete(b.servers, port)
		}
	}
	b.mu.Unlock()
	routes.logConflicts(b.Log)
	for _, l := range stop {
		shutdown(l)
	}
	for port, tlsOn := range want {
		b.listen(port, tlsOn)
	}
	return nil
}

// Status lists ports that could not listen, with the reason.
func (b *Bridge) Status() map[int]string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make(map[int]string, len(b.errs))
	for k, v := range b.errs {
		out[k] = v
	}
	return out
}

func (b *Bridge) listen(port int, tlsOn bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.servers[port] != nil {
		return
	}
	srv := &http.Server{Handler: b.Handler(port), ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second,
		WriteTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		b.errs[port] = err.Error()
		b.Log.Warn("bridge: listen", "port", port, "err", err)
		return
	}
	delete(b.errs, port)
	if tlsOn {
		srv.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12, GetCertificate: b.certificate(port)}
		ln = tls.NewListener(ln, srv.TLSConfig)
	}
	b.servers[port] = &listener{srv: srv, tls: tlsOn}
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			b.Log.Warn("bridge: serve", "port", port, "err", err)
		}
	}()
}

func (b *Bridge) stopAll() {
	b.reloadMu.Lock()
	defer b.reloadMu.Unlock()
	b.mu.Lock()
	all := b.servers
	b.servers = map[int]*listener{}
	b.mu.Unlock()
	for _, l := range all {
		shutdown(l)
	}
}

func shutdown(l *listener) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = l.srv.Shutdown(ctx)
}

// certCache holds loaded certificates; a file is read again when it
// changes on disk (a renewed certificate needs no restart).
type certCache struct {
	mu sync.Mutex
	m  map[string]cachedCert // by the rule's certKey
}

type cachedCert struct {
	cert    *tls.Certificate
	modTime time.Time
	pem     string
	files   string // the certificate and key paths: a changed pair is read again
}

// certificate picks the rule certificate for the SNI name on port.
func (b *Bridge) certificate(port int) func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	cache := &certCache{m: map[string]cachedCert{}}
	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		rt, ok := b.routeForName(port, strings.ToLower(hello.ServerName))
		if !ok {
			return nil, errors.New("no rule on this port")
		}
		return b.loadCert(hello.Context(), cache, rt.certKey(), rt.tlsSite())
	}
}

// routeForName is the first rule in table order on port (panel rules by
// id, then redirect rules by id) whose domains cover name; else the
// port's first rule with no domains, else its first rule (a client
// without SNI, or an IP address). The save check keeps this pick
// unambiguous (D53).
func (b *Bridge) routeForName(port int, name string) (route, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	routes := b.routes[port]
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

// tlsSite is what a handshake reads the route's certificate from: the
// panel rule, or the redirect's certificates row.
func (r route) tlsSite() Site {
	if r.panel != nil {
		return *r.panel
	}
	s := Site{Name: r.ref.Name, CertificateID: r.certID}
	if r.redirect != nil {
		c := r.redirect.cert
		s.CertFile, s.KeyFile, s.CertPEM = c.certFile, c.keyFile, c.pem
	}
	return s
}

func (b *Bridge) loadCert(ctx context.Context, cache *certCache, key string, s Site) (*tls.Certificate, error) {
	var mod time.Time
	if s.CertFile != "" {
		info, err := os.Stat(s.CertFile)
		if err != nil {
			return nil, err
		}
		mod = info.ModTime()
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	files := s.CertFile + "\x00" + s.KeyFile
	if c, ok := cache.m[key]; ok && c.modTime.Equal(mod) && c.pem == s.CertPEM && c.files == files {
		return c.cert, nil
	}
	cert, err := b.readCert(ctx, s)
	if err != nil {
		return nil, err
	}
	cache.m[key] = cachedCert{cert: cert, modTime: mod, pem: s.CertPEM, files: files}
	return cert, nil
}

// readCert reads a pasted key from the certificates row when the rule
// has one, else from the site's own columns (D54).
func (b *Bridge) readCert(ctx context.Context, s Site) (*tls.Certificate, error) {
	if s.CertFile != "" {
		c, err := tls.LoadX509KeyPair(s.CertFile, s.KeyFile)
		return &c, err
	}
	var key []byte
	var err error
	if s.CertificateID != 0 {
		key, err = b.st.certificateKey(ctx, s.CertificateID)
	} else {
		key, err = b.st.siteKey(ctx, s.ID)
	}
	if err != nil {
		return nil, err
	}
	c, err := tls.X509KeyPair([]byte(s.CertPEM), key)
	return &c, err
}
