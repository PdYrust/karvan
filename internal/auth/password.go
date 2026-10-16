// Package auth handles operator accounts: password hashing, sessions, CSRF
// tokens, login lockout and TOTP two-step sign-in.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/argon2"

	"github.com/PdYrust/karvan/internal/secret"
)

const (
	argonMemoryKiB = 64 * 1024
	argonTime      = 3
	argonThreads   = 2
	argonKeyLen    = 32
	argonSaltLen   = 16
	// Upper bound accepted from stored hashes, so a tampered row cannot make
	// one login allocate gigabytes.
	argonMaxMemoryKiB = 1024 * 1024

	minPasswordLen       = 12
	maxPasswordLen       = 1024
	generatedPasswordLen = 20
	generatedUserLen     = 8

	passwordAlphabet = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	usernameAlphabet = "abcdefghijkmnpqrstuvwxyz23456789"
)

var (
	errMalformedHash = errors.New("malformed password hash")
	usernamePattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,31}$`)
	b64              = base64.RawStdEncoding
)

// CheckNewPassword applies the rules for a password an operator chooses.
func CheckNewPassword(pw string) error {
	if len(pw) < minPasswordLen {
		return fmt.Errorf("password must be at least %d characters", minPasswordLen)
	}
	if len(pw) > maxPasswordLen {
		return fmt.Errorf("password must be at most %d characters", maxPasswordLen)
	}
	return nil
}

// NormalizeUsername trims and lowercases a username and checks its shape.
func NormalizeUsername(u string) (string, error) {
	u = strings.ToLower(strings.TrimSpace(u))
	if !usernamePattern.MatchString(u) {
		return "", errors.New("username must be 3 to 32 characters: a-z, 0-9, '.', '_' or '-'")
	}
	return u, nil
}

// HashPassword returns an argon2id hash in PHC string format.
func HashPassword(pw string) (string, error) {
	if len(pw) > maxPasswordLen {
		return "", fmt.Errorf("password must be at most %d characters", maxPasswordLen)
	}
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	key := argon2.IDKey([]byte(pw), salt, argonTime, argonMemoryKiB, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemoryKiB, argonTime, argonThreads, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// VerifyPassword reports whether pw matches the encoded hash.
func VerifyPassword(encoded, pw string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != fmt.Sprintf("v=%d", argon2.Version) {
		return false, errMalformedHash
	}
	var mem, iter uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &iter, &threads); err != nil {
		return false, errMalformedHash
	}
	if mem == 0 || mem > argonMaxMemoryKiB || iter == 0 || threads == 0 {
		return false, errMalformedHash
	}
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false, errMalformedHash
	}
	want, err := b64.DecodeString(parts[5])
	if err != nil || len(want) == 0 || len(pw) > maxPasswordLen {
		return false, errMalformedHash
	}
	got := argon2.IDKey([]byte(pw), salt, iter, mem, threads, uint32(len(want))) //nolint:gosec // G115: len(want) is a decoded hash, far below 2^32.
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// GeneratePassword returns a random password (about 117 bits) without
// look-alike characters, so it can be copied from a terminal by hand.
func GeneratePassword() (string, error) {
	return secret.RandomString(passwordAlphabet, generatedPasswordLen)
}

// GenerateUsername returns a random, valid username.
func GenerateUsername() (string, error) {
	return secret.RandomString(usernameAlphabet, generatedUserLen)
}
