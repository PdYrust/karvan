package secret

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
)

// RandomString returns n characters drawn uniformly from alphabet.
func RandomString(alphabet string, n int) (string, error) {
	b := make([]byte, n)
	size := big.NewInt(int64(len(alphabet)))
	for i := range b {
		idx, err := rand.Int(rand.Reader, size)
		if err != nil {
			return "", fmt.Errorf("generate random string: %w", err)
		}
		b[i] = alphabet[idx.Int64()]
	}
	return string(b), nil
}

// NewToken returns 256 random bits, URL-safe base64 encoded.
func NewToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
