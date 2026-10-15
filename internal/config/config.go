// Package config loads Karvan's bootstrap configuration from defaults, an
// optional JSON file and command-line flags, in that order of precedence.
//
// Values Karvan changes at runtime (port, bind address, hidden path) are not
// here; they live in the settings store so each value has a single source.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
)

// DefaultPath is where the config file is looked for when -config is not given.
var DefaultPath = "/etc/karvan/config.json"

// Config is the bootstrap configuration.
type Config struct {
	DataDir        string   `json:"dataDir"`
	KeyFile        string   `json:"keyFile"`
	LogLevel       string   `json:"logLevel"`
	TrustedProxies []string `json:"trustedProxies"`

	level   slog.Level
	proxies []netip.Prefix
}

// Defaults returns the configuration used when nothing overrides it.
func Defaults() Config {
	return Config{
		DataDir:  "/var/lib/karvan",
		KeyFile:  "/etc/karvan/secret.key",
		LogLevel: "info",
	}
}

// Level returns the parsed log level. Valid only after Load.
func (c Config) Level() slog.Level { return c.level }

// Proxies returns the parsed trusted proxy networks. Valid only after Load.
func (c Config) Proxies() []netip.Prefix { return c.proxies }

// Flags holds the command-line overrides bound to a flag set.
type Flags struct {
	path, dataDir, keyFile, logLevel string
}

// BindFlags registers the config flags on fs.
func BindFlags(fs *flag.FlagSet) *Flags {
	f := &Flags{}
	fs.StringVar(&f.path, "config", "", "config file (default "+DefaultPath+")")
	fs.StringVar(&f.dataDir, "data-dir", "", "data directory (overrides the config file)")
	fs.StringVar(&f.keyFile, "key-file", "", "encryption key file (overrides the config file)")
	fs.StringVar(&f.logLevel, "log-level", "", "debug, info, warn or error (overrides the config file)")
	return f
}

// Load builds the configuration after the flag set has been parsed.
func (f *Flags) Load() (Config, error) {
	cfg := Defaults()
	path, explicit := f.path, f.path != ""
	if !explicit {
		path = DefaultPath
	}
	if err := readFile(path, &cfg); err != nil {
		if explicit || !errors.Is(err, fs.ErrNotExist) {
			return Config{}, err
		}
	}
	if err := f.override(&cfg); err != nil {
		return Config{}, err
	}
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

func readFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path) //nolint:gosec // G304: the path is chosen by the operator on purpose.
	if err != nil {
		return fmt.Errorf("read config %s: %w", path, err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(cfg); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("parse config %s: unexpected data after the JSON object", path)
	}
	return nil
}

func (f *Flags) override(cfg *Config) error {
	for _, o := range []struct {
		flag string
		dst  *string
	}{{f.dataDir, &cfg.DataDir}, {f.keyFile, &cfg.KeyFile}} {
		if o.flag == "" {
			continue
		}
		abs, err := filepath.Abs(o.flag)
		if err != nil {
			return fmt.Errorf("resolve %s: %w", o.flag, err)
		}
		*o.dst = abs
	}
	if f.logLevel != "" {
		cfg.LogLevel = f.logLevel
	}
	return nil
}

func (c *Config) validate() error {
	if !filepath.IsAbs(c.DataDir) {
		return fmt.Errorf("dataDir %q must be an absolute path", c.DataDir)
	}
	if !filepath.IsAbs(c.KeyFile) {
		return fmt.Errorf("keyFile %q must be an absolute path", c.KeyFile)
	}
	c.DataDir, c.KeyFile = filepath.Clean(c.DataDir), filepath.Clean(c.KeyFile)
	if rel, err := filepath.Rel(c.DataDir, c.KeyFile); err == nil && !strings.HasPrefix(rel, "..") {
		return fmt.Errorf("keyFile %q must be outside dataDir %q, so a copy of the data cannot decrypt itself", c.KeyFile, c.DataDir)
	}
	if err := c.level.UnmarshalText([]byte(c.LogLevel)); err != nil {
		return fmt.Errorf("logLevel %q: use debug, info, warn or error", c.LogLevel)
	}
	c.proxies = c.proxies[:0]
	for _, s := range c.TrustedProxies {
		p, err := parsePrefix(s)
		if err != nil {
			return fmt.Errorf("trustedProxies: %q is not an IP address or CIDR", s)
		}
		c.proxies = append(c.proxies, p)
	}
	return nil
}

func parsePrefix(s string) (netip.Prefix, error) {
	if strings.Contains(s, "/") {
		p, err := netip.ParsePrefix(s)
		return p.Masked(), err
	}
	a, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Prefix{}, err
	}
	return netip.PrefixFrom(a, a.BitLen()), nil
}
