// Package subtoken reads and mints subscription tokens the way Marzban
// 0.8.4 and PasarGuard 5.4.1 do (app/utils/jwt.py in both). Reading a
// payload never trusts it: only the panel that holds the secret, or a
// verification against it, can say a token is real (D56).
package subtoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// Formats a token can have.
const (
	FormatJWT    = "jwt"    // eyJhbGci... with access=subscription
	FormatHMAC   = "hmac"   // PasarGuard: b64(v3,id,ts) "." b64(HMAC)
	FormatLegacy = "legacy" // b64(payload) + 10 characters of sha256 (b64 or hex)
)

const jwtPrefix = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9."

// Payload is what a token claims, unverified.
type Payload struct {
	Format   string
	Username string // set for Marzban-style payloads
	UserID   int64  // set for v2/v3 payloads
	Issued   time.Time
}

// Parse reads the claim in token. ok is false when it has no shape either
// panel issues.
func Parse(token string) (Payload, bool) {
	if len(token) < 15 {
		return Payload{}, false
	}
	if strings.HasPrefix(token, jwtPrefix) {
		return parseJWT(token)
	}
	if i := strings.LastIndexByte(token, '.'); i > 0 {
		data, ok := decode(token[:i])
		if !ok {
			return Payload{}, false
		}
		p, ok := parseData(data)
		p.Format = FormatHMAC
		return p, ok && p.UserID > 0
	}
	data, ok := decode(token[:len(token)-10])
	if !ok {
		return Payload{}, false
	}
	p, ok := parseData(data)
	p.Format = FormatLegacy
	return p, ok
}

func decode(s string) (string, bool) {
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "="))
	return string(b), err == nil
}

// parseData reads "v2|v3,id,ts" or "username,ts" (_parse_subscription_data).
func parseData(s string) (Payload, bool) {
	parts := strings.Split(s, ",")
	switch {
	case len(parts) == 3 && (parts[0] == "v2" || parts[0] == "v3"):
		id, err1 := strconv.ParseInt(parts[1], 10, 64)
		ts, err2 := strconv.ParseInt(parts[2], 10, 64)
		return Payload{UserID: id, Issued: time.Unix(ts, 0).UTC()}, err1 == nil && err2 == nil
	case len(parts) == 2:
		ts, err := strconv.ParseInt(parts[1], 10, 64)
		return Payload{Username: parts[0], Issued: time.Unix(ts, 0).UTC()}, err == nil && parts[0] != ""
	}
	return Payload{}, false
}

func parseJWT(token string) (Payload, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Payload{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Payload{}, false
	}
	var claims struct {
		Sub    string `json:"sub"`
		Access string `json:"access"`
		IAT    int64  `json:"iat"`
	}
	if json.Unmarshal(raw, &claims) != nil || claims.Access != "subscription" || claims.Sub == "" {
		return Payload{}, false
	}
	return Payload{Format: FormatJWT, Username: claims.Sub, Issued: time.Unix(claims.IAT, 0).UTC()}, true
}

// Marzban mints a Marzban token for username at t (create_subscription_token).
func Marzban(username, secret string, t time.Time) string {
	data := base64.RawURLEncoding.EncodeToString([]byte(username + "," + strconv.FormatInt(ceil(t), 10)))
	sum := sha256.Sum256([]byte(data + secret))
	return data + base64.URLEncoding.EncodeToString(sum[:])[:10]
}

// PasarGuard mints a PasarGuard v3 token for userID at t.
func PasarGuard(userID int64, secret string, t time.Time) string {
	data := base64.RawURLEncoding.EncodeToString([]byte("v3," + strconv.FormatInt(userID, 10) + "," + strconv.FormatInt(ceil(t), 10)))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return data + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// ceil is Python's ceil(time.time()).
func ceil(t time.Time) int64 {
	s := t.Unix()
	if t.Nanosecond() > 0 {
		s++
	}
	return s
}

// LegacyValid checks a legacy token against secret with both signature
// encodings PasarGuard accepts (base64 and hex); Marzban only issues the
// first.
func LegacyValid(token, secret string) bool {
	if len(token) < 15 {
		return false
	}
	data, sig := token[:len(token)-10], token[len(token)-10:]
	sum := sha256.Sum256([]byte(data + secret))
	b64 := base64.URLEncoding.EncodeToString(sum[:])[:10]
	hx := hex.EncodeToString(sum[:])[:10]
	return hmac.Equal([]byte(sig), []byte(b64)) || hmac.Equal([]byte(sig), []byte(hx))
}
