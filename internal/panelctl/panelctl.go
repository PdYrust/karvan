// Package panelctl stops and starts a panel installed with its official
// script, the way that script does: `docker compose -f <dir>/<compose file>
// -p <folder name> stop|start <app services>`. Only the panel's own
// containers stop; a database container in the same file keeps running,
// so a backup can be put back into it.
package panelctl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Modes say how Karvan may run Docker.
const (
	ModeAuto   = "auto"   // Docker itself, else sudo -n
	ModeDocker = "docker" // Karvan's user may use Docker itself
	ModeSudo   = "sudo"   // through sudo -n, for a rule that allows only these commands
	ModeOff    = "off"    // never stop or start a panel
)

// Modes lists the valid modes.
var Modes = []string{ModeAuto, ModeDocker, ModeSudo, ModeOff}

// Runner builds a command; tests replace it.
type Runner func(ctx context.Context, name string, args ...string) *exec.Cmd

// Errors.
var (
	ErrOff        = errors.New("panel control is turned off in Karvan's config")
	ErrNoCompose  = errors.New("no compose file next to the panel's ENV file")
	ErrNoServices = errors.New("the compose file has no panel service")
)

// composeNames are the files Docker Compose reads, in its own order.
var composeNames = []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"}

// panelImages are the images of panel services (PasarGuard's backend,
// node worker and scheduler all use its one image).
var panelImages = []string{"gozargah/marzban", "pasarguard/panel"}

// opTimeout bounds one stop or start; Docker stops a container after 10 s.
const opTimeout = 2 * time.Minute

// Control stops and starts one panel.
type Control struct {
	File     string
	Project  string
	Services []string
	compose  []string // "docker", "compose" or "docker-compose"
	run      Runner
}

// Opener finds the control for a panel folder.
type Opener struct {
	Mode string
	Run  Runner // nil: exec.CommandContext, with sudo -n in ModeSudo
}

// Open reads the compose file in dir (the folder of the panel's ENV file),
// finds its panel services and checks Karvan may control them now. In
// ModeAuto it tries Docker itself first, then sudo -n.
func (o Opener) Open(ctx context.Context, dir string) (*Control, error) {
	switch o.Mode {
	case ModeOff:
		return nil, ErrOff
	case ModeAuto:
		c, err := Opener{Mode: ModeDocker, Run: o.Run}.Open(ctx, dir)
		if err == nil || errors.Is(err, ErrNoCompose) || errors.Is(err, ErrNoServices) {
			return c, err
		}
		if c, serr := (Opener{Mode: ModeSudo, Run: o.Run}).Open(ctx, dir); serr == nil {
			return c, nil
		}
		return nil, err
	}
	c, err := o.open(ctx, dir)
	if err != nil {
		return nil, err
	}
	if err := c.check(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

func (o Opener) open(ctx context.Context, dir string) (*Control, error) {
	c := &Control{Project: filepath.Base(dir), run: o.runner()}
	for _, n := range composeNames {
		if f := filepath.Join(dir, n); fileExists(f) {
			c.File = f
			break
		}
	}
	if c.File == "" {
		return nil, ErrNoCompose
	}
	c.compose = []string{"docker", "compose"}
	if err := c.cmd(ctx, "version").Run(); err != nil {
		c.compose = []string{"docker-compose"}
	}
	out, err := c.output(ctx, "config", "--format", "json")
	if err != nil {
		return nil, err
	}
	if c.Services, err = panelServices(out); err != nil {
		return nil, err
	}
	return c, nil
}

func (o Opener) runner() Runner {
	if o.Run != nil {
		return o.Run
	}
	if o.Mode == ModeSudo {
		return func(ctx context.Context, name string, args ...string) *exec.Cmd {
			// The sudo rule names the absolute path.
			if p, err := exec.LookPath(name); err == nil {
				name = p
			}
			return exec.CommandContext(ctx, "sudo", append([]string{"-n", name}, args...)...) //nolint:gosec // G204: fixed docker commands
		}
	}
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, name, args...) //nolint:gosec // G204: fixed docker commands
	}
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.Mode().IsRegular()
}

// panelServices lists the services that run a panel image, sorted.
func panelServices(config []byte) ([]string, error) {
	var c struct {
		Services map[string]struct {
			Image string `json:"image"`
		} `json:"services"`
	}
	if err := json.Unmarshal(config, &c); err != nil {
		return nil, fmt.Errorf("read the compose config: %w", err)
	}
	var out []string
	for name, s := range c.Services {
		if slices.ContainsFunc(panelImages, func(img string) bool {
			return s.Image == img || strings.HasPrefix(s.Image, img+":") || strings.HasPrefix(s.Image, img+"@")
		}) {
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		return nil, ErrNoServices
	}
	slices.Sort(out)
	return out, nil
}

// Stop stops the panel's services and leaves the rest running.
func (c *Control) Stop(ctx context.Context) error { return c.op(ctx, "stop") }

// Start starts the panel's services again.
func (c *Control) Start(ctx context.Context) error { return c.op(ctx, "start") }

func (c *Control) op(ctx context.Context, verb string) error {
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	_, err := c.output(ctx, append([]string{verb}, c.Services...)...)
	return err
}

// check reports whether Karvan may control the panel now, by asking the
// Docker daemon for the panel's containers (config alone needs no daemon).
func (c *Control) check(ctx context.Context) error {
	_, err := c.output(ctx, append([]string{"ps", "-q"}, c.Services...)...)
	return err
}

func (c *Control) cmd(ctx context.Context, args ...string) *exec.Cmd {
	full := append(append(slices.Clone(c.compose[1:]), "-f", c.File, "-p", c.Project), args...)
	if args[0] == "version" {
		full = append(slices.Clone(c.compose[1:]), args...)
	}
	return c.run(ctx, c.compose[0], full...)
}

// output runs a compose command and returns stdout; a failure carries the
// first line of stderr, which says why (no permission, no such service).
func (c *Control) output(ctx context.Context, args ...string) ([]byte, error) {
	cmd := c.cmd(ctx, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		line, _, _ := strings.Cut(strings.TrimSpace(stderr.String()), "\n")
		return nil, fmt.Errorf("docker compose %s: %w: %s", args[0], err, line)
	}
	return stdout.Bytes(), nil
}
