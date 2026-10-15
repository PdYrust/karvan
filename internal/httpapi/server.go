// Package httpapi assembles Karvan's HTTP handler: the hidden base path,
// middleware, the JSON API and the embedded web panel.
package httpapi

import (
	"io/fs"
	"log/slog"
	"net/http"
	"net/netip"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/auth"
	"github.com/PdYrust/karvan/internal/httpapi/handlers"
	"github.com/PdYrust/karvan/internal/httpapi/middleware"
)

// staticFiles are top-level files from web/public served without a session.
var staticFiles = []string{"favicon.svg", "favicon.ico", "apple-touch-icon.png"}

// Deps is everything the handler needs.
type Deps struct {
	BasePath       string
	TrustedProxies []netip.Prefix
	Auth           *auth.Service
	Audit          *audit.Log
	Web            fs.FS
	Version        string
	Logger         *slog.Logger
}

// New builds the root handler.
func New(d Deps) (http.Handler, error) {
	spa, err := handlers.NewSPA(d.Web, d.BasePath)
	if err != nil {
		return nil, err
	}
	cookies := middleware.Cookies{Path: "/" + d.BasePath + "/"}
	api := &handlers.API{Auth: d.Auth, Audit: d.Audit, Cookies: cookies, Version: d.Version}
	signedIn := func(h http.HandlerFunc) http.Handler { return middleware.RequireSession(h) }

	rt := middleware.NewRouter()
	rt.HandleFunc("GET /{$}", spa.Page)
	rt.HandleFunc("GET /assets/{file}", spa.Asset)
	for _, name := range staticFiles {
		rt.HandleFunc("GET /"+name, spa.Static(name))
	}

	rt.HandleFunc("GET /api/v1/session", api.GetSession)
	rt.HandleFunc("POST /api/v1/session", api.CreateSession)
	rt.Handle("DELETE /api/v1/session", signedIn(api.DeleteSession))
	rt.Handle("GET /api/v1/overview", signedIn(api.Overview))
	rt.Handle("POST /api/v1/account/totp", signedIn(api.BeginTOTP))
	rt.Handle("POST /api/v1/account/totp/confirm", signedIn(api.ConfirmTOTP))
	rt.Handle("DELETE /api/v1/account/totp", signedIn(api.DisableTOTP))

	var h http.Handler = rt
	h = middleware.CSRF(cookies, h)
	h = middleware.LoadSession(d.Auth.Sessions, cookies, h)
	h = middleware.SecurityHeaders(h)
	h = rt.Guard(h)
	h = middleware.AccessLog(d.Logger, h)
	h = middleware.ClientIP(d.TrustedProxies, h)
	return middleware.BasePath(d.BasePath, h), nil
}
