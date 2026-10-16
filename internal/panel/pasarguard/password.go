package pasarguard

import (
	"crypto/rand"
	"math/big"
	"strings"
	"unicode"
)

// Password policy codes (app/models/validators.py:116).
const (
	PasswordShort      = "password_short"     // under 12 characters
	PasswordLong       = "password_long"      // over 72 bytes (bcrypt)
	PasswordWeak       = "password_weak"      // needs 2 digits, 2 upper, 2 lower, 1 special
	PasswordHasName    = "password_has_name"  // contains the username
	PasswordHasQuote   = "password_has_quote" // contains "
	minPasswordLength  = 12
	maxPasswordBytes   = 72
	generatedPasswords = 20
)

// CheckPassword returns the first policy code password breaks, or "".
func CheckPassword(password, username string) string {
	var digits, upper, lower, special int
	for _, r := range password {
		switch {
		case unicode.IsDigit(r):
			digits++
		case unicode.IsUpper(r):
			upper++
		case unicode.IsLower(r):
			lower++
		default:
			special++
		}
	}
	switch {
	case len([]rune(password)) < minPasswordLength:
		return PasswordShort
	case len(password) > maxPasswordBytes:
		return PasswordLong
	case digits < 2 || upper < 2 || lower < 2 || special < 1:
		return PasswordWeak
	case username != "" && strings.Contains(strings.ToLower(password), strings.ToLower(username)):
		return PasswordHasName
	case strings.ContainsRune(password, '"'):
		return PasswordHasQuote
	}
	return ""
}

const (
	lowers   = "abcdefghijkmnpqrstuvwxyz"
	uppers   = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	digitSet = "23456789"
	specials = "!#$%&*+-=?@_"
)

// GeneratePassword returns a password that meets the policy for username
// (owner decision 1). Look-alike characters are left out.
func GeneratePassword(username string) (string, error) {
	for {
		b := make([]byte, 0, generatedPasswords)
		for _, set := range []string{lowers, lowers, uppers, uppers, digitSet, digitSet, specials} {
			c, err := pick(set)
			if err != nil {
				return "", err
			}
			b = append(b, c)
		}
		all := lowers + uppers + digitSet + specials
		for len(b) < generatedPasswords {
			c, err := pick(all)
			if err != nil {
				return "", err
			}
			b = append(b, c)
		}
		if err := shuffle(b); err != nil {
			return "", err
		}
		if p := string(b); CheckPassword(p, username) == "" {
			return p, nil
		}
	}
}

func pick(set string) (byte, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(set))))
	if err != nil {
		return 0, err
	}
	return set[n.Int64()], nil
}

func shuffle(b []byte) error {
	for i := len(b) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		b[i], b[j.Int64()] = b[j.Int64()], b[i]
	}
	return nil
}
