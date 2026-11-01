package panelctl

import (
	"cmp"
	"context"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// DB is the container a panel's database server runs in, so its own dump
// tools, which match the server's version, can run there.
type DB struct {
	Name string // for docker exec
	Port int    // the server's port inside the container
}

// inspected is the part of a container's details the pick reads.
type inspected struct {
	Name   string `json:"Name"`
	Config struct {
		Image        string              `json:"Image"`
		Labels       map[string]string   `json:"Labels"`
		ExposedPorts map[string]struct{} `json:"ExposedPorts"`
	} `json:"Config"`
	NetworkSettings struct {
		Ports map[string][]binding `json:"Ports"`
	} `json:"NetworkSettings"`
}

// binding is one host port a container port is published on.
type binding struct {
	HostPort string `json:"HostPort"`
}

// dbImages are the image names of each database flavor's server.
var dbImages = map[string][]string{
	"mysql":    {"mysql", "percona"},
	"mariadb":  {"mariadb"},
	"postgres": {"postgres", "timescale", "postgis"},
}

// FindDB finds the running container of the database server at host:port
// of the given flavor ("mysql", "mariadb" or "postgres") through the Docker
// socket (DefaultSocket when socket is ""): first a service of the panel's
// own compose project, run from dir, whose image fits the flavor and that
// publishes or listens on port; else, for a server on this host's loopback,
// any such container that publishes port.
func FindDB(ctx context.Context, socket, dir, flavor, host string, port int) (DB, bool, error) {
	if _, ok := dbImages[flavor]; !ok || port == 0 {
		return DB{}, false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	e := newEngine(cmp.Or(socket, DefaultSocket))
	var list []container
	if err := e.do(ctx, http.MethodGet, "/containers/json", &list); err != nil {
		return DB{}, false, err
	}
	all := make([]inspected, 0, len(list))
	for _, k := range list {
		var c inspected
		if err := e.do(ctx, http.MethodGet, "/containers/"+url.PathEscape(k.ID)+"/json", &c); err != nil {
			return DB{}, false, err
		}
		all = append(all, c)
	}
	db, ok := pickDB(all, dir, flavor, host, port)
	return db, ok, nil
}

// pickDB is FindDB's choice among the running containers.
func pickDB(list []inspected, dir, flavor, host string, port int) (DB, bool) {
	list = slices.Clone(list)
	slices.SortFunc(list, func(a, b inspected) int { return strings.Compare(a.Name, b.Name) })
	list = slices.DeleteFunc(list, func(k inspected) bool { return !isDBImage(k.Config.Image, flavor) })
	for _, k := range list {
		wd := k.Config.Labels[workingDirLabel]
		if wd == "" || dir == "" || filepath.Clean(wd) != filepath.Clean(dir) {
			continue
		}
		if inner, ok := k.published(port); ok {
			return DB{Name: k.name(), Port: inner}, true
		}
		if _, ok := k.Config.ExposedPorts[strconv.Itoa(port)+"/tcp"]; ok {
			return DB{Name: k.name(), Port: port}, true
		}
	}
	if !isLoopback(host) {
		return DB{}, false
	}
	for _, k := range list {
		if inner, ok := k.published(port); ok {
			return DB{Name: k.name(), Port: inner}, true
		}
	}
	return DB{}, false
}

// published is the container port that host port is published from.
func (k inspected) published(port int) (int, bool) {
	want := strconv.Itoa(port)
	for spec, binds := range k.NetworkSettings.Ports {
		inner, proto, _ := strings.Cut(spec, "/")
		n, err := strconv.Atoi(inner)
		if err != nil || proto != "tcp" {
			continue
		}
		if slices.ContainsFunc(binds, func(b binding) bool { return b.HostPort == want }) {
			return n, true
		}
	}
	return 0, false
}

func (k inspected) name() string { return strings.TrimPrefix(k.Name, "/") }

// isDBImage reports whether image runs a server of flavor, at any
// registry, tag or digest.
func isDBImage(image, flavor string) bool {
	image, _, _ = strings.Cut(image, "@")
	base := image[strings.LastIndex(image, "/")+1:]
	base, _, _ = strings.Cut(base, ":")
	return slices.ContainsFunc(dbImages[flavor], func(name string) bool { return strings.Contains(base, name) })
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
