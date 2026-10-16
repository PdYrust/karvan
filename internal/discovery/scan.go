package discovery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
)

// Finding codes. Each one names something missing or wrong; the UI turns
// it into words.
const (
	EnvUnreadable    = "env_unreadable"
	DBURLMissing     = "db_url_missing"
	DBRelativePath   = "db_relative_path"
	DBUnsupported    = "db_unsupported"
	DBUnreachable    = "db_unreachable"
	DBNotWritable    = "db_not_writable"
	DBShared         = "db_shared"
	NotAPanel        = "not_a_panel"
	UnknownRevision  = "unknown_revision"
	SchemaMismatch   = "schema_mismatch"
	UnknownChildren  = "unknown_children"
	XrayUnreadable   = "xray_config_unreadable"
	CertUnreadable   = "cert_unreadable"
	PathRelative     = "path_relative"
	maxEnvFileSize   = 1 << 20
	maxComposeLength = 1 << 20
)

// Finding is one thing discovery could not do or check.
type Finding struct {
	Code   string `json:"code"`
	Detail string `json:"detail,omitempty"`
}

// Panel is one panel found on this server. It holds no secret: the DB
// password is re-read from the ENV file whenever Karvan connects.
type Panel struct {
	Name     string     `json:"name"`
	Dir      string     `json:"dir"`
	EnvPath  string     `json:"envPath"`
	Kind     panel.Kind `json:"kind,omitempty"`
	Hint     panel.Kind `json:"hint,omitempty"` // from the compose image name
	Revision string     `json:"revision,omitempty"`
	DB       string     `json:"db,omitempty"` // Target.String(), no password
	Flavor   string     `json:"flavor,omitempty"`
	Identity string     `json:"-"`
	Settings Settings   `json:"settings"`
	Findings []Finding  `json:"findings"`
}

// Settings are the ENV values Karvan uses, as the panel would read them.
type Settings struct {
	DatabaseURL     string   `json:"-"`
	XrayConfig      string   `json:"xrayConfig,omitempty"`
	ExcludeInbounds []string `json:"excludeInbounds,omitempty"`
	SubPath         string   `json:"subPath"`
	SubURLPrefix    string   `json:"subUrlPrefix,omitempty"`
	Host            string   `json:"host"`
	Port            int      `json:"port"`
	UDS             string   `json:"uds,omitempty"`
	CertFile        string   `json:"certFile,omitempty"`
	KeyFile         string   `json:"keyFile,omitempty"`
}

// Ready reports whether Karvan can use the panel as a destination.
func (p Panel) Ready() bool { return p.Kind != "" && len(p.Findings) == 0 }

func (p *Panel) note(code, detail string) { p.Findings = append(p.Findings, Finding{code, detail}) }

// DefaultRoots are where both official installers put panels
// (/opt/<APP_NAME>/.env; APP_NAME is configurable, so every folder counts).
var DefaultRoots = []string{"/opt"}

// Scan inspects every <root>/*/.env, plus extra ENV files, and returns
// the panels sorted by folder. Panels that share a database are flagged.
func Scan(ctx context.Context, roots, extra []string) []Panel {
	var envs []string
	for _, r := range roots {
		matches, _ := filepath.Glob(filepath.Join(r, "*", ".env"))
		envs = append(envs, matches...)
	}
	envs = append(envs, extra...)
	slices.Sort(envs)
	envs = slices.Compact(envs)
	var out []Panel
	seen := map[string]string{}
	for _, e := range envs {
		p := Inspect(ctx, e)
		if p.Kind == "" && hasFinding(p, NotAPanel) {
			continue
		}
		if p.Identity != "" {
			if first, ok := seen[p.Identity]; ok {
				p.note(DBShared, first)
			} else {
				seen[p.Identity] = p.Name
			}
		}
		out = append(out, p)
	}
	return out
}

func hasFinding(p Panel, code string) bool {
	return slices.ContainsFunc(p.Findings, func(f Finding) bool { return f.Code == code })
}

// Inspect reads one ENV file and probes the database it names.
func Inspect(ctx context.Context, envPath string) Panel {
	dir := filepath.Dir(envPath)
	p := Panel{Name: filepath.Base(dir), Dir: dir, EnvPath: envPath, Findings: []Finding{}}
	env, err := readEnv(envPath)
	if err != nil {
		p.note(EnvUnreadable, err.Error())
		return p
	}
	p.Hint = composeHint(dir)
	p.Settings = settingsFrom(env)
	if p.Settings.DatabaseURL == "" {
		if p.Hint == "" {
			p.note(NotAPanel, "")
			return p
		}
		p.note(DBURLMissing, "")
		return p
	}
	probe(ctx, &p)
	checkFiles(&p)
	return p
}

func readEnv(path string) (map[string]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxEnvFileSize {
		return nil, fmt.Errorf("%s is larger than %d bytes", path, maxEnvFileSize)
	}
	b, err := os.ReadFile(path) //nolint:gosec // a panel ENV file, read only
	if err != nil {
		return nil, err
	}
	return ParseDotenv(b), nil
}

// composeHint guesses the panel from the image in the folder's compose
// file. The database has the last word.
func composeHint(dir string) panel.Kind {
	for _, name := range []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"} {
		f := filepath.Join(dir, name)
		info, err := os.Stat(f)
		if err != nil || info.Size() > maxComposeLength {
			continue
		}
		b, err := os.ReadFile(f) //nolint:gosec // a panel compose file, read only
		if err != nil {
			continue
		}
		switch s := string(b); {
		case strings.Contains(s, "pasarguard/panel"):
			return panel.PasarGuard
		case strings.Contains(s, "gozargah/marzban"):
			return panel.Marzban
		}
	}
	return ""
}

func settingsFrom(env map[string]string) Settings {
	s := Settings{
		DatabaseURL:     env["SQLALCHEMY_DATABASE_URL"],
		XrayConfig:      env["XRAY_JSON"],
		ExcludeInbounds: strings.Fields(env["XRAY_EXCLUDE_INBOUND_TAGS"]),
		SubPath:         firstOf(env, "XRAY_SUBSCRIPTION_PATH", "SUBSCRIPTION_PATH"),
		SubURLPrefix:    env["XRAY_SUBSCRIPTION_URL_PREFIX"],
		Host:            firstOf(env, "UVICORN_HOST"),
		UDS:             env["UVICORN_UDS"],
		CertFile:        env["UVICORN_SSL_CERTFILE"],
		KeyFile:         env["UVICORN_SSL_KEYFILE"],
		Port:            8000,
	}
	if s.SubPath == "" {
		s.SubPath = "sub"
	}
	s.SubPath = strings.Trim(s.SubPath, "/")
	if s.Host == "" {
		s.Host = "0.0.0.0"
	}
	if n, err := strconv.Atoi(env["UVICORN_PORT"]); err == nil && n > 0 && n < 65536 {
		s.Port = n
	}
	return s
}

func firstOf(env map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := env[k]; v != "" {
			return v
		}
	}
	return ""
}

func probe(ctx context.Context, p *Panel) {
	t, err := paneldb.ParseURL(p.Settings.DatabaseURL)
	switch {
	case errors.Is(err, paneldb.ErrRelativePath):
		p.note(DBRelativePath, "")
		return
	case err != nil:
		p.note(DBUnsupported, err.Error())
		return
	}
	p.DB, p.Identity = t.String(), identity(t)
	if t.Dialect == paneldb.SQLite {
		if err := writable(t.Path); err != nil {
			p.note(DBNotWritable, err.Error())
		}
	}
	r, err := paneldb.OpenReader(ctx, t)
	if err != nil {
		p.note(DBUnreachable, err.Error())
		return
	}
	defer r.Close()
	p.Flavor = r.Server().Flavor
	if err := r.View(ctx, func(q paneldb.Querier) error { return classify(ctx, q, r.Server(), p) }); err != nil {
		p.note(DBUnreachable, err.Error())
	}
}

// identity names the physical database, so two ENV files pointing at one
// database are caught.
func identity(t paneldb.Target) string {
	if t.Dialect == paneldb.SQLite {
		if info, err := os.Stat(t.Path); err == nil {
			if st, ok := info.Sys().(*syscall.Stat_t); ok {
				return fmt.Sprintf("sqlite:%d:%d", st.Dev, st.Ino)
			}
		}
		return "sqlite:" + filepath.Clean(t.Path)
	}
	host := t.Host
	if host == "localhost" || host == "::1" {
		host = "127.0.0.1"
	}
	if t.Socket != "" {
		host = t.Socket
	}
	return fmt.Sprintf("%s:%s:%d/%s", t.Dialect, host, t.Port, t.Database)
}

// writable checks the file and its folder (SQLite writes -wal and -shm
// next to it) for the user Karvan runs as.
func writable(path string) error {
	const wOK = 2
	if err := syscall.Access(path, wOK); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := syscall.Access(filepath.Dir(path), wOK); err != nil {
		return fmt.Errorf("%s: %w", filepath.Dir(path), err)
	}
	return nil
}

func checkFiles(p *Panel) {
	for _, f := range []string{p.Settings.CertFile, p.Settings.KeyFile} {
		if f == "" {
			continue
		}
		if !filepath.IsAbs(f) {
			p.note(PathRelative, f)
		} else if fh, err := os.Open(f); err != nil { //nolint:gosec // a path from the panel ENV, read only
			p.note(CertUnreadable, err.Error())
		} else {
			_ = fh.Close()
		}
	}
}
