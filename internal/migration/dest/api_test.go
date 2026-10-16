package dest

import (
	"testing"

	"github.com/PdYrust/karvan/internal/discovery"
	"github.com/PdYrust/karvan/internal/panel"
)

// Where each panel's main.py binds, for the ENV values that matter.
func TestBaseURL(t *testing.T) {
	for _, c := range []struct {
		kind panel.Kind
		s    discovery.Settings
		want string
	}{
		{panel.Marzban, discovery.Settings{Host: "0.0.0.0", Port: 8000}, "http://127.0.0.1:8000"},
		{panel.Marzban, discovery.Settings{Host: "10.0.0.5", Port: 8000}, "http://127.0.0.1:8000"},
		{panel.PasarGuard, discovery.Settings{Host: "10.0.0.5", Port: 8000}, "http://10.0.0.5:8000"},
		{panel.PasarGuard, discovery.Settings{Host: "203.0.113.9", Port: 8000}, "http://127.0.0.1:8000"},
		{panel.Marzban, discovery.Settings{Host: "0.0.0.0", Port: 443, CertFile: "c", KeyFile: "k"}, "https://127.0.0.1:443"},
		{panel.Marzban, discovery.Settings{Host: "203.0.113.9", Port: 443, CertFile: "c", KeyFile: "k"}, "https://203.0.113.9:443"},
		{panel.PasarGuard, discovery.Settings{Host: "::", Port: 443, CertFile: "c"}, "http://127.0.0.1:443"},
		{panel.Marzban, discovery.Settings{UDS: "/run/m.sock", Port: 8000, CertFile: "c", KeyFile: "k"}, "https://localhost:8000"},
	} {
		if got := baseURL(c.kind, c.s); got != c.want {
			t.Errorf("%s %+v: %s, want %s", c.kind, c.s, got, c.want)
		}
	}
}
