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
	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/dbstats"
	"github.com/PdYrust/karvan/internal/hoststats"
	"github.com/PdYrust/karvan/internal/httpapi/address"
	"github.com/PdYrust/karvan/internal/httpapi/handlers"
	"github.com/PdYrust/karvan/internal/httpapi/middleware"
	"github.com/PdYrust/karvan/internal/migration/run"
	"github.com/PdYrust/karvan/internal/settings"
)

// staticFiles are top-level files from web/public served without a session.
var staticFiles = []string{"favicon.svg", "boot.js"}

// appPages are the first path segments of the web panel's own pages.
var appPages = []string{"analytics", "migrations", "reports", "templates", "links", "panels", "backups", "settings"}

// Deps is everything the handler needs.
type Deps struct {
	BasePath       string
	TrustedProxies []netip.Prefix
	Auth           *auth.Service
	Audit          *audit.Log
	Web            fs.FS
	Version        string
	Logger         *slog.Logger

	// Phase 3: migrations, Safe Mode and the subscription bridge.
	Runs      *run.Service
	Bridge    *bridge.Bridge
	Settings  *settings.Store
	ScanRoots []string
	// Closing is closed when the server shuts down, so long-lived event
	// streams end and let the shutdown finish.
	Closing <-chan struct{}

	// Analytics: this server's numbers and the panels' databases.
	Host       *hoststats.Sampler
	HostStore  *hoststats.Store
	HostReader hoststats.Reader
	DBStats    *dbstats.Sampler

	// Address serves the panel; with it the address can be changed.
	Address *address.Server
}

// New builds the root handler.
func New(d Deps) (http.Handler, error) {
	spa, err := handlers.NewSPA(d.Web, d.BasePath)
	if err != nil {
		return nil, err
	}
	cookies := middleware.Cookies{Path: "/" + d.BasePath + "/"}
	api := &handlers.API{Auth: d.Auth, Audit: d.Audit, Cookies: cookies, Version: d.Version,
		Runs: d.Runs, Bridge: d.Bridge, Settings: d.Settings, ScanRoots: d.ScanRoots, Closing: d.Closing,
		Host: d.Host, HostStore: d.HostStore, HostReader: d.HostReader, DBStats: d.DBStats, Address: d.Address}
	signedIn := func(h http.HandlerFunc) http.Handler { return middleware.RequireSession(h) }

	rt := middleware.NewRouter()
	rt.HandleFunc("GET /{$}", spa.Page)
	rt.HandleFunc("GET /assets/{file}", spa.Asset)
	for _, name := range staticFiles {
		rt.HandleFunc("GET /"+name, spa.Static(name))
	}
	for _, p := range appPages {
		rt.HandleFunc("GET /"+p, spa.App)
		rt.HandleFunc("GET /"+p+"/{rest...}", spa.App)
	}

	rt.HandleFunc("GET /api/v1/session", api.GetSession)
	rt.HandleFunc("POST /api/v1/session", api.CreateSession)
	rt.Handle("DELETE /api/v1/session", signedIn(api.DeleteSession))
	rt.Handle("GET /api/v1/overview", signedIn(api.Overview))
	accountRoutes(rt, api, signedIn, d.Address != nil)
	if d.Runs != nil {
		migrationRoutes(rt, api, signedIn)
		rt.Handle("GET /api/v1/analytics/server", signedIn(api.ServerAnalytics))
		rt.Handle("GET /api/v1/analytics/panels", signedIn(api.PanelsAnalytics))
		rt.Handle("GET /api/v1/analytics/panels/{id}", signedIn(api.PanelAnalytics))
	}

	var h http.Handler = rt
	h = middleware.CSRF(cookies, h)
	h = middleware.LoadSession(d.Auth.Sessions, cookies, h)
	h = middleware.SecurityHeaders(h)
	h = rt.Guard(h)
	h = middleware.AccessLog(d.Logger, h)
	h = middleware.ClientIP(d.TrustedProxies, h)
	return middleware.BasePath(d.BasePath, h), nil
}

// accountRoutes are the operator's login, sessions, audit log and, when the
// server can change it, the panel's address.
func accountRoutes(rt *middleware.Router, api *handlers.API, signedIn func(http.HandlerFunc) http.Handler, address bool) {
	routes := map[string]http.HandlerFunc{
		"POST /api/v1/account/totp":            api.BeginTOTP,
		"POST /api/v1/account/totp/confirm":    api.ConfirmTOTP,
		"DELETE /api/v1/account/totp":          api.DisableTOTP,
		"GET /api/v1/account/sessions":         api.AccountSessions,
		"DELETE /api/v1/account/sessions":      api.EndOtherSessions,
		"DELETE /api/v1/account/sessions/{id}": api.EndSession,
		"POST /api/v1/account/credentials":     api.RenewLogin,
		"GET /api/v1/audit":                    api.AuditLog,
	}
	if address {
		routes["GET /api/v1/settings/address"] = api.GetAddress
		routes["POST /api/v1/settings/address"] = api.StartAddress
		routes["POST /api/v1/settings/address/confirm"] = api.ConfirmAddress
	}
	for pattern, h := range routes {
		rt.Handle(pattern, signedIn(h))
	}
}

func migrationRoutes(rt *middleware.Router, api *handlers.API, signedIn func(http.HandlerFunc) http.Handler) {
	const m = "/api/v1/migrations/{id}"
	for pattern, h := range map[string]http.HandlerFunc{
		"GET /api/v1/panels":                          api.Panels,
		"POST /api/v1/panels/scan":                    api.ScanPanels,
		"GET /api/v1/migrations":                      api.ListMigrations,
		"POST /api/v1/migrations":                     api.CreateMigration,
		"GET " + m:                                    api.GetMigration,
		"POST " + m + "/compare":                      api.Compare,
		"GET " + m + "/changes":                       api.Changes,
		"GET " + m + "/check-report":                  api.CheckReport,
		"GET " + m + "/check-report.csv":              api.CheckReportCSV,
		"GET " + m + "/summary":                       api.Summary,
		"GET " + m + "/groups":                        api.Groups,
		"GET " + m + "/users":                         api.Users,
		"GET " + m + "/plan":                          api.GetPlan,
		"PUT " + m + "/plan":                          api.PutPlan,
		"GET " + m + "/preview":                       api.Preview,
		"GET " + m + "/reload-options":                api.ReloadOptions,
		"GET " + m + "/roles":                         api.Roles,
		"POST " + m + "/run":                          api.RunMigration,
		"POST " + m + "/cancel":                       api.CancelMigration,
		"POST " + m + "/rollback":                     api.RollbackMigration,
		"GET " + m + "/events":                        api.Events,
		"GET " + m + "/report":                        api.Report,
		"GET " + m + "/report.csv":                    api.ReportCSV,
		"GET " + m + "/links.csv":                     api.LinksCSV,
		"POST " + m + "/passwords.csv":                api.PasswordsCSV,
		"GET /api/v1/operations":                      api.Operations,
		"GET /api/v1/templates":                       api.Templates,
		"POST /api/v1/templates":                      api.SaveTemplate,
		"DELETE /api/v1/templates/{id}":               api.DeleteTemplate,
		"GET /api/v1/settings/safe-mode":              api.GetSafeMode,
		"PUT /api/v1/settings/safe-mode":              api.PutSafeMode,
		"GET /api/v1/safe-mode/backups":               api.Backups,
		"DELETE /api/v1/safe-mode/backups/{id}":       api.DeleteBackup,
		"POST /api/v1/safe-mode/backups/{id}/restore": api.RestoreBackup,
		"GET /api/v1/legacy-sites":                    api.Sites,
		"POST /api/v1/legacy-sites":                   api.SaveSite,
		"PATCH /api/v1/legacy-sites/{id}":             api.SaveSite,
		"DELETE /api/v1/legacy-sites/{id}":            api.DeleteSite,
		"POST /api/v1/legacy-sites/cert":              api.InspectCert,
		"POST /api/v1/legacy-sites/{id}/tokens":       api.UploadTokens,
		"GET /api/v1/legacy-sites/{id}/tokens":        api.UploadStatus,
	} {
		rt.Handle(pattern, signedIn(h))
	}
}
