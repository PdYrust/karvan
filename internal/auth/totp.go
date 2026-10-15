package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // G505: RFC 6238 TOTP uses HMAC-SHA1; every authenticator app expects it.
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"time"
)

const (
	totpPeriod     = 30
	totpDigits     = 6
	totpSkewSteps  = 1
	totpSecretSize = 20
	totpIssuer     = "Karvan"
)

var totpEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewTOTPSecret returns a random 160-bit TOTP secret.
func NewTOTPSecret() ([]byte, error) {
	s := make([]byte, totpSecretSize)
	if _, err := rand.Read(s); err != nil {
		return nil, fmt.Errorf("generate TOTP secret: %w", err)
	}
	return s, nil
}

// TOTPSecretText is the base32 form people type into an authenticator app.
func TOTPSecretText(secret []byte) string { return totpEncoding.EncodeToString(secret) }

// TOTPURI returns the otpauth:// URI shown as a QR code during setup.
func TOTPURI(secret []byte, account string) string {
	q := url.Values{}
	q.Set("secret", TOTPSecretText(secret))
	q.Set("issuer", totpIssuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprint(totpDigits))
	q.Set("period", fmt.Sprint(totpPeriod))
	label := url.PathEscape(totpIssuer + ":" + account)
	return "otpauth://totp/" + label + "?" + q.Encode()
}

func totpStep(t time.Time) int64 { return t.Unix() / totpPeriod }

// totpCode implements RFC 4226 dynamic truncation over the RFC 6238 step.
func totpCode(secret []byte, step int64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(step)) //nolint:gosec // G115: steps are positive Unix time / 30.
	mac := hmac.New(sha1.New, secret)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	v := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%0*d", totpDigits, v%1_000_000)
}

// VerifyTOTP checks code against the steps around now. A step at or before
// lastStep was already used and is refused, so a code works only once.
func VerifyTOTP(secret []byte, code string, now time.Time, lastStep int64) (int64, bool) {
	if len(code) != totpDigits {
		return 0, false
	}
	cur := totpStep(now)
	for s := cur - totpSkewSteps; s <= cur+totpSkewSteps; s++ {
		if s <= lastStep {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(totpCode(secret, s)), []byte(code)) == 1 {
			return s, true
		}
	}
	return 0, false
}
