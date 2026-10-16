package bridge

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/source"
	"github.com/PdYrust/karvan/internal/panel/subtoken"
)

// restRe is what may follow the token: a client type such as /clash,
// /sing-box or /info.
var restRe = regexp.MustCompile(`^(/[A-Za-z0-9_\-]{1,32})?$`)

// request is one old-link request routed to a site.
type request struct {
	site  Site
	dest  Destination
	token string
	rest  string
	mac   []byte
}

// Handler serves the sites on one port.
func (b *Bridge) Handler(port int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { b.serve(w, r, port) })
}

// serve runs the order of D59: verified set, then the panel itself, then
// one lazy check against the old panel.
func (b *Bridge) serve(w http.ResponseWriter, r *http.Request, port int) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		answer(w, http.StatusMethodNotAllowed)
		return
	}
	q, ok := b.route(port, r)
	if !ok {
		answer(w, http.StatusNotFound)
		return
	}
	d, err := b.dest(r.Context(), q.site.PanelID)
	if err != nil {
		b.Log.Warn("bridge: destination", "site", q.site.Name, "err", err)
		answer(w, http.StatusServiceUnavailable)
		return
	}
	q.dest, q.mac = d, mac(b.macKey, q.token)
	if b.fromVerified(w, r, q) || b.passThrough(w, r, q) {
		return
	}
	b.lazy(w, r, q)
}

// route finds the site for a request and splits /{old_path}/{token}{rest}.
func (b *Bridge) route(port int, r *http.Request) (request, bool) {
	host := strings.ToLower(r.Host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, s := range b.sites {
		rest, ok := strings.CutPrefix(r.URL.EscapedPath(), "/"+s.OldPath+"/")
		if s.Port != port || !ok || !s.covers(host) {
			continue
		}
		token, tail, _ := strings.Cut(rest, "/")
		if tail != "" {
			tail = "/" + tail
		}
		if !tokenRe.MatchString(token) || !restRe.MatchString(tail) {
			return request{}, false
		}
		return request{site: s, token: token, rest: tail}, true
	}
	return request{}, false
}

// fromVerified serves a token proven earlier, if the user still passes.
// A token proven on one site counts on every site of the same old panel
// and destination (an old panel often serves its links on 80 and 443).
func (b *Bridge) fromVerified(w http.ResponseWriter, r *http.Request, q request) bool {
	ctx := r.Context()
	v, ok, err := b.st.token(ctx, q.mac)
	if err != nil || !ok || !b.sameSource(v.SiteID, q.site) {
		return false
	}
	e, err := b.Ledger.Get(ctx, v.LedgerID)
	switch {
	case errors.Is(err, ledger.ErrNotFound), err == nil && (e.Dest != q.dest.Key() || (e.State != ledger.Done && e.State != ledger.Adopted)):
		_ = b.st.dropToken(ctx, q.mac) // the user was moved again or rolled back
		return false
	case err != nil:
		// A passing fault must not cost a proof that may never be made again.
		answer(w, http.StatusServiceUnavailable)
		return true
	}
	verd, err := b.check(ctx, q.dest, e, v.Issued)
	if err != nil {
		answer(w, http.StatusServiceUnavailable)
		return true
	}
	b.apply(ctx, e, verd)
	if !verd.ok {
		return false
	}
	if b.Now().Sub(v.Checked) > recheckAfter {
		b.recheck(q, e)
	}
	b.serveAs(w, r, q, e)
	return true
}

// sameSource reports whether site id reads the same old panel into the
// same destination as s.
func (b *Bridge) sameSource(id int64, s Site) bool {
	if id == s.ID {
		return true
	}
	o, ok := b.site(id)
	return ok && o.origin == s.origin && o.PanelID == s.PanelID
}

// passThrough lets the panel judge the token itself. It reports false
// only when the panel answered 404.
func (b *Bridge) passThrough(w http.ResponseWriter, r *http.Request, q request) bool {
	resp, err := forward(r, q.dest, q.token, q.rest)
	if err != nil {
		b.Log.Warn("bridge: panel", "site", q.site.Name, "err", err)
		answer(w, http.StatusServiceUnavailable)
		return true
	}
	if resp.StatusCode == http.StatusNotFound {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
		_ = resp.Body.Close()
		return false
	}
	relay(w, resp)
	return true
}

// serveAs answers with the panel's own response for the moved user.
func (b *Bridge) serveAs(w http.ResponseWriter, r *http.Request, q request, e ledger.Entry) {
	resp, err := forward(r, q.dest, q.dest.Token(e.Row.ID, e.Username, b.Now()), q.rest)
	if err != nil {
		answer(w, http.StatusServiceUnavailable)
		return
	}
	relay(w, resp)
}

// lazy verifies an unknown token once against the old panel (D59 step 3).
// The limits come before any database work, so tokens forged to name a
// real user cannot make the bridge query both databases without bound.
func (b *Bridge) lazy(w http.ResponseWriter, r *http.Request, q request) {
	ctx := r.Context()
	p, ok := subtoken.Parse(q.token)
	if !ok || b.neg.has(q.mac) || !q.site.Lazy || plainHTTPRefused(q.site) {
		answer(w, http.StatusNotFound)
		return
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	done, ok := b.lim.acquire(q.site.ID, ip)
	if !ok {
		answer(w, http.StatusServiceUnavailable)
		return
	}
	defer done()
	e, status := b.candidate(ctx, q, p)
	if status == 0 {
		status = b.verify(ctx, q, e)
	}
	if status != 0 {
		answer(w, status)
		return
	}
	b.serveAs(w, r, q, e)
}

// plainHTTPRefused reports whether s reads a plain-http old panel without
// the operator's opt-in: anyone on the path could answer for it.
func plainHTTPRefused(s Site) bool {
	return strings.HasPrefix(s.SourceURL, "http://") && !s.PlainHTTPOK
}

// candidate finds the ledger row a token claims and checks everything
// that needs no call to the old panel. status is 0 when a check may run.
func (b *Bridge) candidate(ctx context.Context, q request, p subtoken.Payload) (ledger.Entry, int) {
	e, err := b.claimed(ctx, q, p)
	switch {
	case errors.Is(err, ledger.ErrNotFound):
		return e, http.StatusNotFound
	case err != nil:
		return e, http.StatusServiceUnavailable
	}
	verd, err := b.check(ctx, q.dest, e, p.Issued)
	if err != nil {
		return e, http.StatusServiceUnavailable
	}
	b.apply(ctx, e, verd)
	if !verd.ok {
		return e, http.StatusNotFound
	}
	return e, 0
}

// claimed is the ledger row of the user a token names. The payload only
// picks the row; the old panel's answer must then match it.
func (b *Bridge) claimed(ctx context.Context, q request, p subtoken.Payload) (ledger.Entry, error) {
	switch {
	case q.site.SourceKind == panel.PasarGuard && p.UserID > 0:
		return b.Ledger.User(ctx, q.site.origin, strconv.FormatInt(p.UserID, 10), q.dest.Key())
	case p.Username != "":
		return b.Ledger.UserByName(ctx, q.site.origin, p.Username, q.dest.Key())
	}
	return ledger.Entry{}, ledger.ErrNotFound
}

// verify asks the old panel whose token this is and stores it when the
// answer is the ledger's user. status is 0 on success.
func (b *Bridge) verify(ctx context.Context, q request, e ledger.Entry) int {
	o, err := b.old(q.site)
	if err != nil {
		return http.StatusServiceUnavailable
	}
	u, err := o.Owner(ctx, q.token)
	switch {
	case source.CodeOf(err) == source.CodeNotFound:
		b.neg.add(q.mac)
		return http.StatusNotFound
	case err != nil:
		return http.StatusServiceUnavailable
	case !sameUser(q.site.SourceKind, u, e):
		b.Log.Warn("bridge: old panel names another user", "site", q.site.Name, "ledger", e.ID)
		return http.StatusNotFound
	}
	p, _ := subtoken.Parse(q.token)
	stored, err := b.st.putToken(ctx, q.mac, verified{SiteID: q.site.ID, LedgerID: e.ID, Issued: p.Issued})
	if err != nil {
		return http.StatusServiceUnavailable
	}
	if !stored {
		return http.StatusNotFound // closed while the check ran
	}
	return 0
}
