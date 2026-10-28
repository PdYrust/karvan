package bridge

import (
	"context"
	"io"
	"net"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/PdYrust/karvan/internal/migration/ledger"
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
// one lazy check against the old panel. Users in an inactive or deleted
// batch get the plain 404 before the panel is asked (D22).
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
	held, err := b.st.ruleHeld(r.Context(), q.site)
	if err != nil {
		b.Log.Warn("bridge: rule batches", "site", q.site.Name, "err", err)
		answer(w, http.StatusServiceUnavailable)
		return
	}
	if held {
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
	if b.fromVerified(w, r, q) || b.namesHeld(w, r, q) || b.passThrough(w, r, q) {
		return
	}
	b.lazy(w, r, q)
}

// route finds the one rule for a request (D52) and, for a panel rule,
// splits /{old_path}/{token}{rest}. A malformed token or rest on the
// winner is a 404; no other rule is tried. A redirect rule that wins
// gets the plain 404 here.
func (b *Bridge) route(port int, r *http.Request) (request, bool) {
	host := strings.ToLower(r.Host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	rt, rest, ok := b.routes.match(port, host, r.URL.EscapedPath())
	if !ok || rt.panel == nil {
		return request{}, false
	}
	token, tail, _ := strings.Cut(rest[1:], "/") // a panel rule's rest starts with its slash
	if tail != "" {
		tail = "/" + tail
	}
	if !tokenRe.MatchString(token) || !restRe.MatchString(tail) {
		return request{}, false
	}
	return request{site: *rt.panel, token: token, rest: tail}, true
}

// fromVerified serves a token proven earlier, if the user still passes.
// A token proven on one rule counts on every rule whose scope holds its
// row (D23): an old panel often serves its links on 80 and 443.
func (b *Bridge) fromVerified(w http.ResponseWriter, r *http.Request, q request) bool {
	ctx := r.Context()
	v, row, ok, err := b.st.token(ctx, q.mac, q.site, q.dest.Key())
	switch {
	case err != nil:
		// A passing fault must not cost a proof that may never be made again.
		b.Log.Warn("bridge: token", "site", q.site.Name, "err", err)
		answer(w, http.StatusServiceUnavailable)
		return true
	case !ok:
		return false
	case row.held:
		answer(w, http.StatusNotFound) // nothing asked, dropped or closed (D22)
		return true
	case row.State != ledger.Done && row.State != ledger.Adopted:
		_ = b.st.dropToken(ctx, q.mac) // the user was moved again or rolled back
		return false
	case !row.inScope:
		return false
	}
	verd, err := b.check(ctx, q.dest, row.Entry, v.Issued)
	if err != nil {
		answer(w, http.StatusServiceUnavailable)
		return true
	}
	b.apply(ctx, row.Entry, verd)
	if !verd.ok {
		return false
	}
	if b.Now().Sub(v.Checked) > recheckAfter {
		b.recheck(q, row.Entry)
	}
	b.serveAs(w, r, q, row.Entry)
	return true
}

// namesHeld answers 404 when the token's payload names a user of this
// rule in an inactive or deleted batch, before the panel is asked (D22).
func (b *Bridge) namesHeld(w http.ResponseWriter, r *http.Request, q request) bool {
	p, ok := subtoken.Parse(q.token)
	if !ok {
		return false
	}
	rows, err := b.st.named(r.Context(), q.site, q.dest.Key(), p)
	if err != nil {
		b.Log.Warn("bridge: ledger", "site", q.site.Name, "err", err)
		answer(w, http.StatusServiceUnavailable)
		return true
	}
	if slices.ContainsFunc(rows, q.site.holds) {
		answer(w, http.StatusNotFound)
		return true
	}
	return false
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
// The limits come before any work on the destination, so tokens forged to
// name a real user cannot make the bridge query it without bound.
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
	cands, status := b.candidates(ctx, q, p)
	var e ledger.Entry
	if status == 0 {
		e, status = b.verify(ctx, q, cands)
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

// candidate is a row in the rule's scope that a token's payload names.
type candidate struct {
	ledger.Entry
	ok bool // it passes the destination check
}

// candidates reads, fresh, every row in the rule's scope the payload
// names (D24) and runs the destination check on each. status is 0 when
// one passes, so the old panel may be asked; a payload naming a user in
// a held batch answers 404 with no candidate (D22).
func (b *Bridge) candidates(ctx context.Context, q request, p subtoken.Payload) ([]candidate, int) {
	rows, err := b.st.named(ctx, q.site, q.dest.Key(), p)
	switch {
	case err != nil:
		return nil, http.StatusServiceUnavailable
	case slices.ContainsFunc(rows, q.site.holds):
		return nil, http.StatusNotFound
	}
	var out []candidate
	status := http.StatusNotFound
	for _, r := range rows {
		if !r.inScope {
			continue
		}
		verd, err := b.check(ctx, q.dest, r.Entry, p.Issued)
		if err != nil {
			return nil, http.StatusServiceUnavailable
		}
		b.apply(ctx, r.Entry, verd)
		out = append(out, candidate{Entry: r.Entry, ok: verd.ok})
		if verd.ok {
			status = 0
		}
	}
	return out, status
}

// verify asks the old panel whose token this is and stores it when the
// answer matches exactly one candidate that passed its check (D24). Zero
// or two matches answer 404. status is 0 on success.
func (b *Bridge) verify(ctx context.Context, q request, cands []candidate) (ledger.Entry, int) {
	o, err := b.old(q.site)
	if err != nil {
		return ledger.Entry{}, http.StatusServiceUnavailable
	}
	u, err := o.Owner(ctx, q.token)
	switch {
	case source.CodeOf(err) == source.CodeNotFound:
		b.neg.add(q.mac)
		return ledger.Entry{}, http.StatusNotFound
	case err != nil:
		return ledger.Entry{}, http.StatusServiceUnavailable
	}
	var match []candidate
	for _, c := range cands {
		if sameUser(q.site.SourceKind, u, c.Entry) {
			match = append(match, c)
		}
	}
	switch {
	case len(match) == 0:
		b.Log.Warn("bridge: old panel names another user", "site", q.site.Name, "ledger", ledgerIDs(cands))
		return ledger.Entry{}, http.StatusNotFound
	case len(match) > 1:
		b.Log.Warn("bridge: an old link matches more than one moved user", "site", q.site.Name, "ledger", ledgerIDs(match))
		return ledger.Entry{}, http.StatusNotFound
	case !match[0].ok:
		return ledger.Entry{}, http.StatusNotFound
	}
	e := match[0].Entry
	p, _ := subtoken.Parse(q.token)
	stored, err := b.st.putToken(ctx, q.mac, verified{SiteID: q.site.ID, LedgerID: e.ID, Issued: p.Issued})
	if err != nil {
		return e, http.StatusServiceUnavailable
	}
	if !stored {
		return e, http.StatusNotFound // closed while the check ran
	}
	return e, 0
}

func ledgerIDs(cs []candidate) []int64 {
	ids := make([]int64, len(cs))
	for i, c := range cs {
		ids[i] = c.ID
	}
	return ids
}
