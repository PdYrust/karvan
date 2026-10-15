// Package secret encrypts the values Karvan stores (tokens, TOTP secrets,
// SSH keys, passwords) with a key that lives in its own file, never in the
// database.
package secret

import (
	"bytes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/chacha20poly1305"
)

const keyVersion byte = 1

// Key seals and opens values with XChaCha20-Poly1305.
type Key struct {
	aead cipher.AEAD
}

// CreateKeyFile writes a new random key to path. It never replaces an
// existing file, because that would make every stored secret unreadable.
func CreateKeyFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create key directory: %w", err)
	}
	raw := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Errorf("generate key: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // G304: path comes from the operator's config.
	if err != nil {
		return fmt.Errorf("create key file: %w", err)
	}
	if _, err := fmt.Fprintln(f, hex.EncodeToString(raw)); err != nil {
		_ = f.Close()
		return fmt.Errorf("write key file: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("sync key file: %w", err)
	}
	return f.Close()
}

// LoadKeyFile reads the key at path. It refuses a file other users can read.
func LoadKeyFile(path string) (*Key, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("open key file: %w", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("key file %s can be read by other users; run: chmod 600 %s", path, path)
	}
	data, err := os.ReadFile(path) //nolint:gosec // G304: path comes from the operator's config.
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}
	raw, err := hex.DecodeString(string(bytes.TrimSpace(data)))
	if err != nil || len(raw) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("key file %s is not a valid key", path)
	}
	aead, err := chacha20poly1305.NewX(raw)
	if err != nil {
		return nil, fmt.Errorf("init cipher: %w", err)
	}
	return &Key{aead: aead}, nil
}

// AAD names the place a value is stored, so a ciphertext copied into another
// row or column fails to open.
func AAD(table, column string, rowID int64) string {
	return fmt.Sprintf("%s/%s/%d", table, column, rowID)
}

// Seal encrypts plaintext. Output layout: version | nonce | ciphertext+tag.
func (k *Key) Seal(plaintext []byte, aad string) ([]byte, error) {
	nonce := make([]byte, k.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	out := make([]byte, 0, 1+len(nonce)+len(plaintext)+k.aead.Overhead())
	out = append(append(out, keyVersion), nonce...)
	return k.aead.Seal(out, nonce, plaintext, []byte(aad)), nil
}

// ErrDecrypt is returned for any ciphertext that fails to open.
var ErrDecrypt = errors.New("secret: cannot decrypt value")

// Open decrypts a value produced by Seal with the same aad.
func (k *Key) Open(ciphertext []byte, aad string) ([]byte, error) {
	n := k.aead.NonceSize()
	if len(ciphertext) < 1+n+k.aead.Overhead() || ciphertext[0] != keyVersion {
		return nil, ErrDecrypt
	}
	pt, err := k.aead.Open(nil, ciphertext[1:1+n], ciphertext[1+n:], []byte(aad))
	if err != nil {
		return nil, ErrDecrypt
	}
	return pt, nil
}
