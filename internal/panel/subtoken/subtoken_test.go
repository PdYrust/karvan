package subtoken

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

type vectors struct {
	Secret  string `json:"secret"`
	Marzban []struct {
		Username string `json:"username"`
		TS       int64  `json:"ts"`
		Token    string `json:"token"`
	} `json:"marzban"`
	PasarGuard []struct {
		ID    int64  `json:"id"`
		TS    int64  `json:"ts"`
		Token string `json:"token"`
	} `json:"pasarguard"`
	LegacyHex []struct {
		Payload string `json:"payload"`
		Token   string `json:"token"`
	} `json:"legacyHex"`
}

func load(t *testing.T) vectors {
	t.Helper()
	b, err := os.ReadFile("testdata/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v vectors
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestMintMatchesThePanels(t *testing.T) {
	v := load(t)
	for _, m := range v.Marzban {
		if got := Marzban(m.Username, v.Secret, time.Unix(m.TS, 0)); got != m.Token {
			t.Errorf("Marzban(%s) = %s, want %s", m.Username, got, m.Token)
		}
		p, ok := Parse(m.Token)
		if !ok || p.Format != FormatLegacy || p.Username != m.Username || p.Issued.Unix() != m.TS || !LegacyValid(m.Token, v.Secret) {
			t.Errorf("Parse(%s) = %+v %v", m.Token, p, ok)
		}
	}
	for _, g := range v.PasarGuard {
		if got := PasarGuard(g.ID, v.Secret, time.Unix(g.TS, 0)); got != g.Token {
			t.Errorf("PasarGuard(%d) = %s, want %s", g.ID, got, g.Token)
		}
		p, ok := Parse(g.Token)
		if !ok || p.Format != FormatHMAC || p.UserID != g.ID || p.Issued.Unix() != g.TS {
			t.Errorf("Parse(%s) = %+v %v", g.Token, p, ok)
		}
	}
	for _, l := range v.LegacyHex {
		if !LegacyValid(l.Token, v.Secret) || LegacyValid(l.Token, "other") {
			t.Errorf("hex legacy %s", l.Payload)
		}
	}
}

func TestParseReadsLegacyIDTokensAndJWTs(t *testing.T) {
	v := load(t)
	p, ok := Parse(v.LegacyHex[0].Token)
	if !ok || p.UserID != 97318 || p.Format != FormatLegacy {
		t.Fatalf("legacy v3: %+v %v", p, ok)
	}
	jwt := jwtPrefix + "eyJzdWIiOiJhbGkiLCJhY2Nlc3MiOiJzdWJzY3JpcHRpb24iLCJpYXQiOjE3MDAwMDAwMDB9.sig"
	if p, ok := Parse(jwt); !ok || p.Username != "ali" || p.Issued.Unix() != 1700000000 {
		t.Fatalf("jwt: %+v %v", p, ok)
	}
	for _, bad := range []string{"short", "!!!!!!!!!!!!!!!!!!!!", jwtPrefix + "e30.x", "bm9jb21tYQ0123456789"} {
		if _, ok := Parse(bad); ok {
			t.Errorf("Parse(%q) accepted", bad)
		}
	}
}

func TestCeilLikePython(t *testing.T) {
	if ceil(time.Unix(10, 0)) != 10 || ceil(time.Unix(10, 1)) != 11 {
		t.Fatal("ceil")
	}
}
