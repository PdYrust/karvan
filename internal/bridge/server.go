package bridge

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"
)

// listener is one port the bridge serves.
type listener struct {
	srv *http.Server
	tls bool
}

// Run adopts the sites' own certificates and fills the domains from them,
// then serves every site and sweeps hourly until ctx ends.
func (b *Bridge) Run(ctx context.Context) error {
	b.adoptCertificates(ctx)
	b.fillDomains(ctx)
	if err := b.Reload(ctx); err != nil {
		return err
	}
	b.startedOnce.Do(func() { close(b.started) })
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
	facts, err := b.st.shakeFacts(ctx)
	if err != nil {
		return err
	}
	want, err := portModes(append(siteRules(sites), redirects...))
	if err != nil {
		return err
	}
	b.keepSharedPorts(want)
	b.mu.Lock()
	if _, ok := want[b.ChallengePort]; !ok && len(b.chal.holders) > 0 {
		want[b.ChallengePort] = false // plain HTTP for Let's Encrypt only
	}
	b.load(sites, redirects)
	b.shakes, b.certRows = buildShakes(b.routes, facts), facts.certs
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

// keepSharedPorts adds the ports the panel is shared on to want, so the
// panel stays reachable with no rule left there.
func (b *Bridge) keepSharedPorts(want map[int]bool) {
	for _, port := range b.sharedPorts() {
		if _, ok := want[port]; !ok {
			want[port] = true
		}
	}
}

// portModes is whether each port of rules serves TLS.
func portModes(rules []rule) (map[int]bool, error) {
	want := map[int]bool{}
	for _, r := range rules {
		if tlsOn, ok := want[r.port]; ok && tlsOn != r.tls {
			return nil, fmt.Errorf("port %d has rules with and without a certificate", r.port)
		}
		want[r.port] = r.tls
	}
	return want, nil
}

// Status lists ports that could not listen, with the reason.
func (b *Bridge) Status() map[int]string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make(map[int]string, len(b.errs))
	for k, v := range b.errs {
		out[k] = v.Error()
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
		b.errs[port] = err
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
