package subtoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// The running panels accept these tokens (test/panels); this test pins
// the encoding so a change shows up without Docker.
func TestAdminToken(t *testing.T) {
	now := time.Unix(1785654065, 0)
	tok := AdminToken(AdminClaims{Sub: "owner", Access: "admin", AdminID: 7}, "s3cret", now, 10*time.Minute)
	parts := strings.Split(tok, ".")
	if len(parts) != 3 || parts[0]+"." != jwtPrefix {
		t.Fatalf("token shape: %q", tok)
	}
	mac := hmac.New(sha256.New, []byte("s3cret"))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	if parts[2] != base64.RawURLEncoding.EncodeToString(mac.Sum(nil)) {
		t.Fatal("signature does not match HS256")
	}
	raw, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"sub": "owner", "access": "admin", "iat": 1785654065.0, "aid": 7.0, "exp": 1785654665.0}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}
	if _, ok := got["binding"]; ok {
		t.Error("a token with no binding carries one")
	}
	if _, ok := Parse(tok); ok {
		t.Error("an admin token must not read as a subscription token")
	}
	m := AdminToken(AdminClaims{Sub: "root", Access: "sudo"}, "s3cret", now, time.Minute)
	raw, _ = base64.RawURLEncoding.DecodeString(strings.Split(m, ".")[1])
	if strings.Contains(string(raw), "aid") {
		t.Errorf("Marzban token carries aid: %s", raw)
	}
}

// The binding is PasarGuard 5.5's: hex HMAC-SHA256 of
// "admin-token-binding:" and the password hash, under the jwt secret.
func TestAdminBinding(t *testing.T) {
	// hmac.new(b"s3cret", b"admin-token-binding:$2b$12$x", sha256).hexdigest()
	const want = "04fd508267544aaa353629fead2bdaa3b4c21f0668fecf271494beb3b93568f1"
	if got := AdminBinding("$2b$12$x", "s3cret"); got != want {
		t.Fatalf("binding = %s, want %s", got, want)
	}
	tok := AdminToken(AdminClaims{Sub: "owner", Access: "admin", AdminID: 7, Binding: want}, "s3cret", time.Unix(0, 0), time.Minute)
	raw, _ := base64.RawURLEncoding.DecodeString(strings.Split(tok, ".")[1])
	if !strings.Contains(string(raw), `"binding":"`+want+`"`) {
		t.Fatalf("claims %s", raw)
	}
}
