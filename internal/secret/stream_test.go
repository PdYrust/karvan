package secret

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"path/filepath"
	"testing"
)

func testKey(t *testing.T) *Key {
	t.Helper()
	p := filepath.Join(t.TempDir(), "k")
	if err := CreateKeyFile(p); err != nil {
		t.Fatal(err)
	}
	k, err := LoadKeyFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func seal(t *testing.T, k *Key, plain []byte, aad string) []byte {
	t.Helper()
	var out bytes.Buffer
	w, err := k.NewStreamWriter(&out, aad)
	if err != nil {
		t.Fatal(err)
	}
	// Uneven writes cross chunk borders.
	for len(plain) > 0 {
		n := min(len(plain), 10_000)
		if _, err := w.Write(plain[:n]); err != nil {
			t.Fatal(err)
		}
		plain = plain[n:]
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func open(k *Key, sealed []byte, aad string) ([]byte, error) {
	r, err := k.NewStreamReader(bytes.NewReader(sealed), aad)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

func TestStreamRoundTrip(t *testing.T) {
	k := testKey(t)
	for _, size := range []int{0, 1, chunkSize - 1, chunkSize, chunkSize + 1, 3*chunkSize + 17} {
		plain := make([]byte, size)
		_, _ = rand.Read(plain)
		got, err := open(k, seal(t, k, plain, "backup/1"), "backup/1")
		if err != nil || !bytes.Equal(got, plain) {
			t.Fatalf("size %d: %v", size, err)
		}
	}
}

func TestStreamRefusesTampering(t *testing.T) {
	k := testKey(t)
	plain := make([]byte, 3*chunkSize)
	_, _ = rand.Read(plain)
	sealed := seal(t, k, plain, "backup/1")
	head := len(streamMagic) + prefixSize
	chunk := func(i int) []byte { return sealed[head+i*sealedChunk : head+(i+1)*sealedChunk] }

	swapped := append(append(append(append([]byte{}, sealed[:head]...), chunk(1)...), chunk(0)...), sealed[head+2*sealedChunk:]...)
	flipped := bytes.Clone(sealed)
	flipped[head+100] ^= 1
	cases := map[string][]byte{
		"cut at a chunk border": sealed[:head+3*sealedChunk],
		"cut inside a chunk":    sealed[:len(sealed)-5],
		"chunks swapped":        swapped,
		"one bit flipped":       flipped,
		"no header":             sealed[2:],
	}
	for name, s := range cases {
		if _, err := open(k, s, "backup/1"); !errors.Is(err, ErrStream) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if _, err := open(k, sealed, "backup/2"); !errors.Is(err, ErrStream) {
		t.Error("another AAD must not open")
	}
	if _, err := open(testKey(t), sealed, "backup/1"); !errors.Is(err, ErrStream) {
		t.Error("another key must not open")
	}
}
