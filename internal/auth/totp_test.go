package auth

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

// RFC 6238 appendix B, SHA-1 rows, last six digits.
var rfcSecret = []byte("12345678901234567890")

func TestTOTPMatchesRFC6238Vectors(t *testing.T) {
	for unix, want := range map[int64]string{
		59:          "287082",
		1111111109:  "081804",
		1111111111:  "050471",
		1234567890:  "005924",
		2000000000:  "279037",
		20000000000: "353130",
	} {
		if got := totpCode(rfcSecret, totpStep(time.Unix(unix, 0))); got != want {
			t.Errorf("T=%d: code %s, want %s", unix, got, want)
		}
	}
}

func TestVerifyTOTPAllowsOneStepOfClockSkew(t *testing.T) {
	now := time.Unix(1234567890, 0)
	prev := totpCode(rfcSecret, totpStep(now)-1)
	next := totpCode(rfcSecret, totpStep(now)+1)
	for _, code := range []string{prev, next} {
		if _, ok := VerifyTOTP(rfcSecret, code, now, 0); !ok {
			t.Errorf("code %s from a neighbouring step refused", code)
		}
	}
	far := totpCode(rfcSecret, totpStep(now)+2)
	if _, ok := VerifyTOTP(rfcSecret, far, now, 0); ok {
		t.Error("code two steps away accepted")
	}
}

func TestVerifyTOTPRefusesReplay(t *testing.T) {
	now := time.Unix(1234567890, 0)
	code := totpCode(rfcSecret, totpStep(now))
	step, ok := VerifyTOTP(rfcSecret, code, now, 0)
	if !ok {
		t.Fatal("fresh code refused")
	}
	if _, ok := VerifyTOTP(rfcSecret, code, now, step); ok {
		t.Fatal("same code accepted twice")
	}
}

func TestVerifyTOTPRejectsBadInput(t *testing.T) {
	now := time.Unix(1234567890, 0)
	for _, code := range []string{"", "12345", "1234567", "abcdef"} {
		if _, ok := VerifyTOTP(rfcSecret, code, now, 0); ok {
			t.Errorf("code %q accepted", code)
		}
	}
}

func TestTOTPURI(t *testing.T) {
	s, err := NewTOTPSecret()
	if err != nil || len(s) != totpSecretSize {
		t.Fatalf("secret %d bytes, %v", len(s), err)
	}
	u, err := url.Parse(TOTPURI(s, "owner"))
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme != "otpauth" || u.Host != "totp" || !strings.HasSuffix(u.Path, ":owner") {
		t.Errorf("uri = %s", u)
	}
	if u.Query().Get("secret") != TOTPSecretText(s) || u.Query().Get("digits") != "6" {
		t.Errorf("query = %v", u.Query())
	}
}
