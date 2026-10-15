package auth

import (
	"strings"
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	h, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Fatalf("hash = %q, want argon2id PHC string", h)
	}
	if ok, err := VerifyPassword(h, "correct horse battery"); !ok || err != nil {
		t.Fatalf("Verify(correct) = %v, %v", ok, err)
	}
	if ok, _ := VerifyPassword(h, "correct horse batterY"); ok {
		t.Fatal("Verify(wrong) = true")
	}
}

func TestHashesAreSalted(t *testing.T) {
	a, _ := HashPassword("same password")
	b, _ := HashPassword("same password")
	if a == b {
		t.Fatal("two hashes of the same password are equal")
	}
}

func TestVerifyRejectsMalformedHashes(t *testing.T) {
	for _, h := range []string{
		"",
		"$2b$12$abcdefghijklmnopqrstuv",
		"$argon2id$v=18$m=65536,t=3,p=2$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=x,t=3,p=2$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=65536,t=3,p=2$!!$aGFzaA",
		"$argon2id$v=19$m=65536,t=3,p=2$c2FsdA$!!",
		"$argon2id$v=19$m=999999999,t=3,p=2$c2FsdA$aGFzaA",
	} {
		if _, err := VerifyPassword(h, "pw"); err == nil {
			t.Errorf("Verify(%q) gave no error", h)
		}
	}
}

func TestPasswordLengthRules(t *testing.T) {
	if err := CheckNewPassword("short"); err == nil {
		t.Error("short password accepted")
	}
	if err := CheckNewPassword(strings.Repeat("a", 1025)); err == nil {
		t.Error("oversized password accepted")
	}
	if err := CheckNewPassword("a long enough passphrase"); err != nil {
		t.Errorf("good password rejected: %v", err)
	}
	if _, err := HashPassword(strings.Repeat("a", 1025)); err == nil {
		t.Error("HashPassword accepted an oversized password")
	}
}

func TestGeneratedCredentialsAreValid(t *testing.T) {
	pw, err := GeneratePassword()
	if err != nil || CheckNewPassword(pw) != nil || len(pw) != generatedPasswordLen {
		t.Fatalf("GeneratePassword = %q, %v", pw, err)
	}
	u, err := GenerateUsername()
	if err != nil {
		t.Fatal(err)
	}
	if got, err := NormalizeUsername(u); err != nil || got != u {
		t.Fatalf("GenerateUsername = %q, normalize = %q, %v", u, got, err)
	}
}

func TestNormalizeUsername(t *testing.T) {
	if got, err := NormalizeUsername("  Owner.One "); err != nil || got != "owner.one" {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, bad := range []string{"ab", "has space", "ünïcode", strings.Repeat("a", 33), "slash/name"} {
		if _, err := NormalizeUsername(bad); err == nil {
			t.Errorf("NormalizeUsername(%q) accepted", bad)
		}
	}
}
