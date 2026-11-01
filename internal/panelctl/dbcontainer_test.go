package panelctl

import (
	"encoding/json"
	"testing"
)

// containers are a PasarGuard install with MySQL on the host network, as
// its script sets it up, next to a stray MariaDB that publishes 3307.
const containersJSON = `[
	{"Name": "/pasarguard", "Config": {"Image": "pasarguard/panel:latest",
		"Labels": {"com.docker.compose.project.working_dir": "/opt/pasarguard"}}},
	{"Name": "/pasarguard-mysql-1", "Config": {"Image": "mysql:lts",
		"Labels": {"com.docker.compose.project.working_dir": "/opt/pasarguard"},
		"ExposedPorts": {"3306/tcp": {}, "33060/tcp": {}}}, "NetworkSettings": {"Ports": {}}},
	{"Name": "/pasarguard-nats-1", "Config": {"Image": "nats:2.10-alpine",
		"Labels": {"com.docker.compose.project.working_dir": "/opt/pasarguard"}, "ExposedPorts": {"4222/tcp": {}}},
		"NetworkSettings": {"Ports": {"4222/tcp": [{"HostIp": "0.0.0.0", "HostPort": "4222"}]}}},
	{"Name": "/db-2", "Config": {"Image": "docker.io/library/mariadb:11.4@sha256:abc", "ExposedPorts": {"3306/tcp": {}}},
		"NetworkSettings": {"Ports": {"3306/tcp": [{"HostIp": "127.0.0.1", "HostPort": "3307"}]}}},
	{"Name": "/marzban-pg", "Config": {"Image": "timescale/timescaledb:2.30.2-pg17",
		"Labels": {"com.docker.compose.project.working_dir": "/opt/marzban/"}, "ExposedPorts": {"5432/tcp": {}}},
		"NetworkSettings": {"Ports": {"5432/tcp": [{"HostIp": "127.0.0.1", "HostPort": "15432"}], "5433/tcp": null}}}
]`

func TestPickDB(t *testing.T) {
	var list []inspected
	if err := json.Unmarshal([]byte(containersJSON), &list); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, dir, flavor, host string
		port                    int
		want                    DB
	}{
		{"compose service on the host network", "/opt/pasarguard", "mysql", "127.0.0.1", 3306, DB{"pasarguard-mysql-1", 3306}},
		{"compose service through a published port", "/opt/marzban", "postgres", "localhost", 15432, DB{"marzban-pg", 5432}},
		{"published port of another project", "/opt/pasarguard", "mariadb", "127.0.0.1", 3307, DB{"db-2", 3306}},
		{"another flavor", "/opt/pasarguard", "mariadb", "127.0.0.1", 3306, DB{}},
		{"an image that is no database", "/opt/pasarguard", "mysql", "127.0.0.1", 4222, DB{}},
		{"a published port on a remote server", "/opt/x", "mariadb", "10.0.0.5", 3307, DB{}},
		{"another port", "/opt/pasarguard", "mysql", "127.0.0.1", 3310, DB{}},
	}
	for _, c := range cases {
		got, ok := pickDB(list, c.dir, c.flavor, c.host, c.port)
		if got != c.want || ok != (c.want != DB{}) {
			t.Errorf("%s: got %+v %v, want %+v", c.name, got, ok, c.want)
		}
	}
}
