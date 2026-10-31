package bridge

import (
	"context"
	"net"
	"net/netip"
	"regexp"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzbanapi"
	"github.com/PdYrust/karvan/internal/panel/pasarguardapi"
	"github.com/PdYrust/karvan/internal/panel/source"
)

// OldPanel answers "whose token is this?" for one site's old panel.
type OldPanel interface {
	// Owner is the user /{path}/{token}/info names. A token the panel does
	// not know is a source error with CodeNotFound.
	Owner(ctx context.Context, token string) (panel.User, error)
}

// tokenRe is every character either panel puts in a token (base64url,
// "." and "=" padding). Anything else never reaches the old panel.
var tokenRe = regexp.MustCompile(`^[A-Za-z0-9_\-.=]{15,1000}$`)

// lazyTimeout bounds one check; the client is waiting on it.
const lazyTimeout = 8 * time.Second

type oldPanel struct {
	kind   panel.Kind
	path   string
	client *source.Client
}

// pinnedResolver answers every lookup with the address the migration
// used, because the old domain may point at this server by now.
type pinnedResolver struct{ addr netip.Addr }

func (p pinnedResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return []netip.Addr{p.addr}, nil
}

// openOldPanel builds the client for a site's old panel: one attempt, a
// short timeout, the stored certificate pin and address.
func openOldPanel(s Site) (OldPanel, error) {
	o := source.Options{BaseURL: s.SourceURL, PinSHA256: s.SourcePin, Timeout: lazyTimeout, Attempts: 1}
	if s.SourceIP != "" {
		o.Resolver = pinnedResolver{netip.MustParseAddr(s.SourceIP)}
	} else {
		o.Resolver = net.DefaultResolver
	}
	c, err := source.New(o)
	if err != nil {
		return nil, err
	}
	return &oldPanel{kind: s.SourceKind, path: s.OldPath, client: c}, nil
}

func (o *oldPanel) Owner(ctx context.Context, token string) (panel.User, error) {
	if o.kind == panel.PasarGuard {
		return pasarguardapi.SubscriptionUser(ctx, o.client, o.path, token)
	}
	return marzbanapi.SubscriptionUser(ctx, o.client, o.path, token)
}
