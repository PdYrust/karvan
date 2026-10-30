package subtoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"time"
)

// AdminClaims are the fields of an admin access token. Marzban reads sub,
// access ("admin" or "sudo") and iat; PasarGuard also reads aid, and from
// 5.5 binding for a database admin. Both check the token against the
// admin's password_reset_at through iat.
type AdminClaims struct {
	Sub     string `json:"sub"`
	Access  string `json:"access"`
	IAT     int64  `json:"iat"`
	AdminID int64  `json:"aid,omitempty"`
	Binding string `json:"binding,omitempty"`
	Exp     int64  `json:"exp"`
}

// AdminBinding ties a PasarGuard 5.5 admin token to the admin's password
// hash, like _admin_token_binding in app/utils/jwt.py: the hex HMAC-SHA256
// of the hash under the jwt secret. A token without it is refused there;
// older versions ignore it.
func AdminBinding(hashedPassword, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("admin-token-binding:" + hashedPassword))
	return hex.EncodeToString(mac.Sum(nil))
}

// AdminToken mints an HS256 admin token like create_admin_token in both
// panels, valid from t for ttl. Karvan uses it only for the reload calls,
// on the same server, and never stores it.
func AdminToken(c AdminClaims, secret string, t time.Time, ttl time.Duration) string {
	c.IAT = t.Unix()
	c.Exp = t.Add(ttl).Unix()
	body, _ := json.Marshal(c) // a struct of strings and ints always encodes
	data := jwtPrefix + base64.RawURLEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return data + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
