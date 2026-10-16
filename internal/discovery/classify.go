package discovery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzban"
	"github.com/PdYrust/karvan/internal/panel/marzbandb"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
)

// classify decides the panel from the database shape (PasarGuard has
// admin_roles, Marzban has proxies) and checks the revision and schema.
func classify(ctx context.Context, q paneldb.Querier, srv paneldb.Server, p *Panel) error {
	roles, err := paneldb.Columns(ctx, q, srv.Dialect, "admin_roles")
	if err != nil {
		return err
	}
	proxies, err := paneldb.Columns(ctx, q, srv.Dialect, "proxies")
	if err != nil {
		return err
	}
	var unknown []paneldb.ForeignKey
	switch {
	case len(roles) > 0:
		p.Kind = panel.PasarGuard
		s, err := pasarguarddb.Inspect(ctx, q, srv)
		p.Revision, unknown = s.Revision, s.UnknownChildren
		noteSchema(p, err)
	case len(proxies) > 0:
		p.Kind = panel.Marzban
		s, err := marzbandb.Inspect(ctx, q, srv)
		p.Revision, unknown = s.Revision, s.UnknownChildren
		noteSchema(p, err)
		checkXray(p)
	default:
		p.note(NotAPanel, "no admin_roles or proxies table")
		return nil
	}
	for _, fk := range unknown {
		p.note(UnknownChildren, fmt.Sprintf("%s.%s -> %s", fk.Table, fk.Column, fk.RefTable))
	}
	return nil
}

func noteSchema(p *Panel, err error) {
	var ve *panel.VersionError
	var se *panel.SchemaError
	switch {
	case err == nil:
	case errors.As(err, &ve):
		p.note(UnknownRevision, ve.Revision)
	case errors.As(err, &se):
		p.note(SchemaMismatch, se.Detail)
	default:
		p.note(DBUnreachable, err.Error())
	}
}

// checkXray makes sure Marzban's Xray config, which defines its inbounds,
// can be read. Marzban's default is ./xray_config.json in its working
// directory, which cannot be found from outside the container.
func checkXray(p *Panel) {
	path := p.Settings.XrayConfig
	if path == "" {
		path = "./xray_config.json"
	}
	if !filepath.IsAbs(path) {
		p.note(PathRelative, path)
		return
	}
	if _, err := os.Stat(path); err != nil {
		p.note(XrayUnreadable, err.Error())
		return
	}
	if _, err := marzban.Inbounds(path, p.Settings.ExcludeInbounds); err != nil {
		p.note(XrayUnreadable, err.Error())
	}
}
