package config

import (
	"flag"
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func parse(t *testing.T, args ...string) (Config, error) {
	t.Helper()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	f := BindFlags(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	return f.Load()
}

func writeFile(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDefaultsWhenDefaultFileIsMissing(t *testing.T) {
	old := DefaultPath
	DefaultPath = filepath.Join(t.TempDir(), "absent.json")
	t.Cleanup(func() { DefaultPath = old })

	cfg, err := parse(t)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DataDir != "/var/lib/karvan" || cfg.KeyFile != "/etc/karvan/secret.key" {
		t.Errorf("defaults = %+v", cfg)
	}
	if cfg.Level() != slog.LevelInfo {
		t.Errorf("level = %v, want info", cfg.Level())
	}
}

func TestExplicitMissingFileIsAnError(t *testing.T) {
	_, err := parse(t, "-config", filepath.Join(t.TempDir(), "absent.json"))
	if err == nil || !strings.Contains(err.Error(), "absent.json") {
		t.Fatalf("err = %v, want an error naming the file", err)
	}
}

func TestFileValuesOverrideDefaults(t *testing.T) {
	p := writeFile(t, `{"dataDir":"/srv/karvan","keyFile":"/etc/k/key","logLevel":"debug","trustedProxies":["10.0.0.0/8","127.0.0.1"]}`)
	cfg, err := parse(t, "-config", p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DataDir != "/srv/karvan" || cfg.KeyFile != "/etc/k/key" || cfg.Level() != slog.LevelDebug {
		t.Errorf("cfg = %+v", cfg)
	}
	want := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("127.0.0.1/32")}
	got := cfg.Proxies()
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("proxies = %v, want %v", got, want)
	}
}

func TestFlagsOverrideFile(t *testing.T) {
	p := writeFile(t, `{"dataDir":"/srv/karvan","logLevel":"debug"}`)
	cfg, err := parse(t, "-config", p, "-data-dir", "/opt/data", "-log-level", "warn")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DataDir != "/opt/data" || cfg.Level() != slog.LevelWarn {
		t.Errorf("cfg = %+v", cfg)
	}
}

func TestRelativeFlagPathsBecomeAbsolute(t *testing.T) {
	p := writeFile(t, `{}`)
	cfg, err := parse(t, "-config", p, "-data-dir", "data", "-key-file", "keys/k")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !filepath.IsAbs(cfg.DataDir) || !filepath.IsAbs(cfg.KeyFile) {
		t.Errorf("paths not absolute: %+v", cfg)
	}
}

func TestRejectsUnknownFields(t *testing.T) {
	p := writeFile(t, `{"dataDirr":"/x"}`)
	if _, err := parse(t, "-config", p); err == nil || !strings.Contains(err.Error(), "dataDirr") {
		t.Fatalf("err = %v, want unknown field error", err)
	}
}

func TestValidation(t *testing.T) {
	cases := map[string]string{
		"relative data dir":    `{"dataDir":"data"}`,
		"key inside data dir":  `{"dataDir":"/var/lib/karvan","keyFile":"/var/lib/karvan/secret.key"}`,
		"unknown log level":    `{"logLevel":"loud"}`,
		"bad trusted proxy":    `{"trustedProxies":["not-an-ip"]}`,
		"relative key file":    `{"keyFile":"secret.key"}`,
		"key equals data dir":  `{"dataDir":"/srv/k","keyFile":"/srv/k"}`,
		"key named ..x inside": `{"dataDir":"/srv/k","keyFile":"/srv/k/..key"}`,
		"trailing json values": `{} {}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parse(t, "-config", writeFile(t, body)); err == nil {
				t.Fatal("Load succeeded, want an error")
			}
		})
	}
}
