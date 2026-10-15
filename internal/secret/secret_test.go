package secret

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newKey(t *testing.T) *Key {
	t.Helper()
	p := filepath.Join(t.TempDir(), "keys", "secret.key")
	if err := CreateKeyFile(p); err != nil {
		t.Fatalf("CreateKeyFile: %v", err)
	}
	k, err := LoadKeyFile(p)
	if err != nil {
		t.Fatalf("LoadKeyFile: %v", err)
	}
	return k
}

func TestCreateKeyFileUsesPrivatePermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "keys")
	p := filepath.Join(dir, "secret.key")
	if err := CreateKeyFile(p); err != nil {
		t.Fatalf("CreateKeyFile: %v", err)
	}
	for path, want := range map[string]os.FileMode{p: 0o600, dir: 0o700} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != want {
			t.Errorf("%s mode = %o, want %o", path, got, want)
		}
	}
}

func TestCreateKeyFileNeverOverwrites(t *testing.T) {
	p := filepath.Join(t.TempDir(), "secret.key")
	if err := CreateKeyFile(p); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(p)
	if err := CreateKeyFile(p); err == nil {
		t.Fatal("second CreateKeyFile succeeded, want an error")
	}
	after, _ := os.ReadFile(p)
	if !bytes.Equal(before, after) {
		t.Fatal("key file changed")
	}
}

func TestLoadKeyFileRejectsLoosePermissions(t *testing.T) {
	p := filepath.Join(t.TempDir(), "secret.key")
	if err := CreateKeyFile(p); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadKeyFile(p); err == nil || !strings.Contains(err.Error(), "chmod 600") {
		t.Fatalf("err = %v, want a permissions error", err)
	}
}

func TestLoadKeyFileReportsMissingFile(t *testing.T) {
	if _, err := LoadKeyFile(filepath.Join(t.TempDir(), "absent.key")); err == nil {
		t.Fatal("LoadKeyFile succeeded on a missing file")
	}
}

func TestCreateKeyFileFailsWhenDirectoryCannotBeMade(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CreateKeyFile(filepath.Join(blocker, "secret.key")); err == nil {
		t.Fatal("CreateKeyFile succeeded under a regular file")
	}
}

func TestLoadKeyFileRejectsMalformedKey(t *testing.T) {
	for name, body := range map[string]string{"short": "abcd\n", "not hex": strings.Repeat("zz", 32) + "\n"} {
		t.Run(name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "secret.key")
			if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadKeyFile(p); err == nil {
				t.Fatal("LoadKeyFile succeeded, want an error")
			}
		})
	}
}

func TestSealOpenRoundTrip(t *testing.T) {
	k := newKey(t)
	aad := AAD("operators", "totp_secret", 7)
	ct, err := k.Seal([]byte("JBSWY3DPEHPK3PXP"), aad)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ct, []byte("JBSWY3DPEHPK3PXP")) {
		t.Fatal("ciphertext contains the plaintext")
	}
	pt, err := k.Open(ct, aad)
	if err != nil || string(pt) != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("Open = %q, %v", pt, err)
	}
}

func TestSealUsesFreshNonces(t *testing.T) {
	k := newKey(t)
	a, _ := k.Seal([]byte("same"), "x")
	b, _ := k.Seal([]byte("same"), "x")
	if bytes.Equal(a, b) {
		t.Fatal("two seals of the same plaintext are identical")
	}
}

func TestOpenRejectsSwappedRow(t *testing.T) {
	k := newKey(t)
	ct, _ := k.Seal([]byte("token"), AAD("panels", "token", 1))
	if _, err := k.Open(ct, AAD("panels", "token", 2)); err == nil {
		t.Fatal("Open with another row's AAD succeeded")
	}
}

func TestOpenRejectsTamperingAndOtherKeys(t *testing.T) {
	k := newKey(t)
	ct, _ := k.Seal([]byte("token"), "aad")

	flipped := bytes.Clone(ct)
	flipped[len(flipped)-1] ^= 1
	if _, err := k.Open(flipped, "aad"); err == nil {
		t.Error("Open of a modified ciphertext succeeded")
	}
	if _, err := newKey(t).Open(ct, "aad"); err == nil {
		t.Error("Open with a different key succeeded")
	}
	if _, err := k.Open(ct[:5], "aad"); err == nil {
		t.Error("Open of a truncated ciphertext succeeded")
	}
	wrongVersion := bytes.Clone(ct)
	wrongVersion[0] = 9
	if _, err := k.Open(wrongVersion, "aad"); err == nil {
		t.Error("Open with an unknown key version succeeded")
	}
}

func TestValueNeverPrintsItself(t *testing.T) {
	v := NewValue("hunter2")
	var logs bytes.Buffer
	slog.New(slog.NewJSONHandler(&logs, nil)).Info("x", "password", v)
	js, _ := json.Marshal(struct{ P Value }{v})

	outputs := []string{
		fmt.Sprint(v), fmt.Sprintf("%v %s %+v %#v %q", v, v, v, v, v),
		logs.String(), string(js),
	}
	for _, out := range outputs {
		if strings.Contains(out, "hunter2") {
			t.Errorf("output leaks the secret: %s", out)
		}
	}
	if v.Reveal() != "hunter2" {
		t.Errorf("Reveal = %q", v.Reveal())
	}
	if v.IsZero() || !NewValue("").IsZero() {
		t.Error("IsZero is wrong")
	}
}
