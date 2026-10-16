package marzban

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/PdYrust/karvan/internal/panel"
)

// maxXrayConfig caps the config file read; real ones are a few KB.
const maxXrayConfig = 8 << 20

// Inbounds reads the user-facing inbounds of the Xray config at path the
// way Marzban does (app/xray/config.py:143): protocols vmess, vless, trojan
// and shadowsocks, minus the tags in exclude (XRAY_EXCLUDE_INBOUND_TAGS).
// Targets come back sorted by protocol, then tag.
func Inbounds(path string, exclude []string) ([]panel.Target, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("read xray config: %w", err)
	}
	if info.Size() > maxXrayConfig {
		return nil, fmt.Errorf("read xray config: %s is larger than %d bytes", path, maxXrayConfig)
	}
	raw, err := os.ReadFile(path) //nolint:gosec // the path comes from the panel's own ENV, read only
	if err != nil {
		return nil, fmt.Errorf("read xray config: %w", err)
	}
	return ParseInbounds(raw, exclude)
}

// ParseInbounds is Inbounds on bytes already read.
func ParseInbounds(raw []byte, exclude []string) ([]panel.Target, error) {
	var cfg struct {
		Inbounds []struct {
			Tag      string `json:"tag"`
			Protocol string `json:"protocol"`
		} `json:"inbounds"`
	}
	if err := json.Unmarshal(StripComments(raw), &cfg); err != nil {
		return nil, fmt.Errorf("parse xray config: %w", err)
	}
	var out []panel.Target
	for _, in := range cfg.Inbounds {
		if !slices.Contains(Protocols, in.Protocol) || slices.Contains(exclude, in.Tag) {
			continue
		}
		out = append(out, panel.Target{ID: in.Protocol + "/" + in.Tag, Name: in.Tag, Protocol: in.Protocol})
	}
	slices.SortFunc(out, func(a, b panel.Target) int { return strings.Compare(a.ID, b.ID) })
	return out, nil
}

// StripComments removes "//" and "#" line comments outside strings, as
// commentjson allows them in Marzban's config.
func StripComments(src []byte) []byte {
	out := make([]byte, 0, len(src))
	inString, escaped := false, false
	for i := 0; i < len(src); i++ {
		c := src[i]
		if inString {
			out = append(out, c)
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		if c == '#' || c == '/' && i+1 < len(src) && src[i+1] == '/' {
			for i < len(src) && src[i] != '\n' {
				i++
			}
			if i < len(src) {
				out = append(out, '\n')
			}
			continue
		}
		if c == '"' {
			inString = true
		}
		out = append(out, c)
	}
	return out
}
