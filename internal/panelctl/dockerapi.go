package panelctl

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// DefaultSocket is where the Docker daemon listens. Karvan's Docker image
// has no Docker CLI and reaches the daemon through this socket (D29).
const DefaultSocket = "/var/run/docker.sock"

// Labels Docker Compose puts on every container it creates.
const (
	projectLabel = "com.docker.compose.project"
	serviceLabel = "com.docker.compose.service"
)

// engine talks to the Docker Engine API over its unix socket.
type engine struct{ client *http.Client }

func newEngine(socket string) *engine {
	tr := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	return &engine{client: &http.Client{Transport: tr, Timeout: opTimeout}}
}

type container struct {
	ID     string            `json:"Id"`
	Image  string            `json:"Image"`
	Labels map[string]string `json:"Labels"`
}

// openSocket finds the panel's containers through the Docker socket: the
// compose project named after dir, as the official scripts name it, and
// its containers that run a panel image, stopped ones too.
func (o Opener) openSocket(ctx context.Context, dir string) (*Control, error) {
	file := composeFile(dir)
	if file == "" {
		return nil, ErrNoCompose
	}
	c := &Control{File: file, Project: projectName(dir), engine: newEngine(cmp.Or(o.Socket, DefaultSocket))}
	filters, err := json.Marshal(map[string][]string{"label": {projectLabel + "=" + c.Project}})
	if err != nil {
		return nil, fmt.Errorf("docker socket filter: %w", err)
	}
	var list []container
	if err := c.engine.do(ctx, http.MethodGet, "/containers/json?all=1&filters="+url.QueryEscape(string(filters)), &list); err != nil {
		return nil, err
	}
	list = slices.DeleteFunc(list, func(k container) bool { return !isPanelImage(k.Image) })
	if len(list) == 0 {
		return nil, ErrNoServices
	}
	slices.SortFunc(list, func(a, b container) int { return strings.Compare(a.Labels[serviceLabel], b.Labels[serviceLabel]) })
	for _, k := range list {
		c.Services = append(c.Services, k.Labels[serviceLabel])
		c.ids = append(c.ids, k.ID)
	}
	return c, nil
}

// each stops or starts the containers one by one.
func (e *engine) each(ctx context.Context, ids []string, verb string) error {
	for _, id := range ids {
		if err := e.do(ctx, http.MethodPost, "/containers/"+url.PathEscape(id)+"/"+verb, nil); err != nil {
			return err
		}
	}
	return nil
}

// do sends one request and decodes the answer into into, if set. A 304
// means the container already was stopped or started.
func (e *engine) do(ctx context.Context, method, path string, into any) error {
	req, err := http.NewRequestWithContext(ctx, method, "http://docker"+path, nil)
	if err != nil {
		return fmt.Errorf("docker socket: %w", err)
	}
	res, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("docker socket: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("docker socket: %w", err)
	}
	what, _, _ := strings.Cut(path, "?")
	switch {
	case res.StatusCode == http.StatusNotModified:
		return nil
	case res.StatusCode >= http.StatusMultipleChoices:
		var msg struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(body, &msg)
		return fmt.Errorf("docker socket %s %s: %s: %s", method, what, res.Status, msg.Message)
	case into != nil:
		if err := json.Unmarshal(body, into); err != nil {
			return fmt.Errorf("docker socket %s %s: %w", method, what, err)
		}
	}
	return nil
}

// projectName is the folder name the way Docker Compose turns it into a
// project name: lower case, only letters, digits, dash and underscore,
// starting with a letter or digit.
func projectName(dir string) string {
	name := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return -1
	}, strings.ToLower(filepath.Base(dir)))
	return strings.TrimLeft(name, "-_")
}

// workingDirLabel is the folder Docker Compose ran a container's project from.
const workingDirLabel = "com.docker.compose.project.working_dir"

// RunningDirs lists the folders of the compose projects that have a
// running container, through the Docker socket (DefaultSocket when socket
// is ""). It fails when Docker cannot be reached.
func RunningDirs(ctx context.Context, socket string) (map[string]bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var list []container
	if err := newEngine(cmp.Or(socket, DefaultSocket)).do(ctx, http.MethodGet, "/containers/json", &list); err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, k := range list {
		if d := k.Labels[workingDirLabel]; d != "" {
			out[filepath.Clean(d)] = true
		}
	}
	return out, nil
}
