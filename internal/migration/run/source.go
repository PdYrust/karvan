package run

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzbanapi"
	"github.com/PdYrust/karvan/internal/panel/pasarguardapi"
	"github.com/PdYrust/karvan/internal/panel/source"
)

// openSource builds the source reader, detecting the panel when the
// operator did not say which one it is.
func openSource(ctx context.Context, l Login, opts source.Options) (panel.Source, panel.Kind, string, *source.Client, error) {
	opts.BaseURL, opts.PinSHA256 = l.URL, l.PinSHA256
	c, err := source.New(opts)
	if err != nil {
		return nil, "", "", nil, err
	}
	kind := l.Kind
	if kind == "" {
		if kind, err = detect(ctx, c, l); err != nil {
			return nil, "", "", nil, err
		}
	}
	var src panel.Source
	switch kind {
	case panel.Marzban:
		src = marzbanapi.New(c, l.Username, l.Password)
	case panel.PasarGuard:
		src = pasarguardapi.New(c, pasarguardapi.Login{Username: l.Username, Password: l.Password, APIKey: l.APIKey})
	default:
		return nil, "", "", nil, fmt.Errorf("unknown panel %q", kind)
	}
	return src, kind, ledger.Origin(c.Base()), c, nil
}

// detect tells the panels apart by /api/admin: Marzban answers with
// users_usage, PasarGuard with used_traffic and an id. An API key only
// exists on PasarGuard.
func detect(ctx context.Context, c *source.Client, l Login) (panel.Kind, error) {
	if !l.APIKey.IsZero() {
		return panel.PasarGuard, nil
	}
	resp, err := c.PostForm(ctx, "/api/admin/token", url.Values{"username": {l.Username}, "password": {l.Password.Reveal()}}, nil)
	if err != nil {
		return "", err
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := source.DecodeJSON(resp.Body, &tok); err != nil {
		return "", err
	}
	resp, err = c.Get(ctx, "/api/admin", nil, http.Header{"Authorization": {"Bearer " + tok.AccessToken}})
	if err != nil {
		return "", err
	}
	var keys map[string]json.RawMessage
	if err := source.DecodeJSON(resp.Body, &keys); err != nil {
		return "", err
	}
	_, mz := keys["users_usage"]
	_, pg := keys["used_traffic"]
	switch {
	case pg:
		return panel.PasarGuard, nil
	case mz:
		return panel.Marzban, nil
	}
	return "", source.BadResponse("unrecognized panel")
}
