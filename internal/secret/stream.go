package secret

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Streams encrypt large files (Safe Mode backups) in chunks, so neither
// side holds the whole file in memory. Layout:
//
//	"KVS1" | 16-byte random prefix | chunk...
//
// Each chunk is chunkSize bytes of plaintext sealed with nonce = prefix |
// chunk counter, and AAD = aad | counter | final flag. The last chunk is
// the only one sealed as final (it may be empty), so cutting the file at a
// chunk boundary, dropping, swapping or repeating chunks all fail to open.
const (
	streamMagic  = "KVS1"
	prefixSize   = 16
	chunkSize    = 64 << 10
	sealedChunk  = chunkSize + 16
	finalFlag    = 1
	notFinalFlag = 0
)

// ErrStream is returned for any stream that fails to open.
var ErrStream = errors.New("secret: stream is damaged, cut short or for another key")

type streamWriter struct {
	k      *Key
	w      io.Writer
	prefix [prefixSize]byte
	aad    string
	buf    []byte
	n      uint64
	closed bool
}

// NewStreamWriter returns a writer that encrypts into w. Close must be
// called to seal the final chunk; without it the stream does not open.
func (k *Key) NewStreamWriter(w io.Writer, aad string) (io.WriteCloser, error) {
	sw := &streamWriter{k: k, w: w, aad: aad, buf: make([]byte, 0, chunkSize)}
	if _, err := rand.Read(sw.prefix[:]); err != nil {
		return nil, fmt.Errorf("generate stream prefix: %w", err)
	}
	if _, err := w.Write(append([]byte(streamMagic), sw.prefix[:]...)); err != nil {
		return nil, err
	}
	return sw, nil
}

func (s *streamWriter) Write(p []byte) (int, error) {
	if s.closed {
		return 0, errors.New("secret: write after close")
	}
	written := 0
	for len(p) > 0 {
		n := min(len(p), chunkSize-len(s.buf))
		s.buf = append(s.buf, p[:n]...)
		p, written = p[n:], written+n
		if len(s.buf) == chunkSize {
			if err := s.flush(notFinalFlag); err != nil {
				return written, err
			}
		}
	}
	return written, nil
}

func (s *streamWriter) flush(final byte) error {
	nonce, ad := chunkNonce(s.prefix, s.n), chunkAAD(s.aad, s.n, final)
	if _, err := s.w.Write(s.k.aead.Seal(nil, nonce, s.buf, ad)); err != nil {
		return err
	}
	s.buf, s.n = s.buf[:0], s.n+1
	return nil
}

// Close seals the last chunk. It does not close the underlying writer.
func (s *streamWriter) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return s.flush(finalFlag)
}

func chunkNonce(prefix [prefixSize]byte, n uint64) []byte {
	nonce := make([]byte, prefixSize+8)
	copy(nonce, prefix[:])
	binary.BigEndian.PutUint64(nonce[prefixSize:], n)
	return nonce
}

func chunkAAD(aad string, n uint64, final byte) []byte {
	b := make([]byte, 0, len(aad)+9)
	b = append(b, aad...)
	b = binary.BigEndian.AppendUint64(b, n)
	return append(b, final)
}

type streamReader struct {
	k      *Key
	r      *bufio.Reader
	prefix [prefixSize]byte
	aad    string
	plain  []byte
	n      uint64
	done   bool
}

// NewStreamReader returns a reader that decrypts a stream from r. Every
// read error that is not io.EOF wraps ErrStream.
func (k *Key) NewStreamReader(r io.Reader, aad string) (io.Reader, error) {
	sr := &streamReader{k: k, r: bufio.NewReaderSize(r, sealedChunk+1), aad: aad}
	head := make([]byte, len(streamMagic)+prefixSize)
	if _, err := io.ReadFull(sr.r, head); err != nil || string(head[:len(streamMagic)]) != streamMagic {
		return nil, ErrStream
	}
	copy(sr.prefix[:], head[len(streamMagic):])
	return sr, nil
}

func (s *streamReader) Read(p []byte) (int, error) {
	for len(s.plain) == 0 {
		if s.done {
			return 0, io.EOF
		}
		if err := s.next(); err != nil {
			return 0, err
		}
	}
	n := copy(p, s.plain)
	s.plain = s.plain[n:]
	return n, nil
}

func (s *streamReader) next() error {
	sealed := make([]byte, sealedChunk)
	n, err := io.ReadFull(s.r, sealed)
	switch {
	case err == nil:
	case errors.Is(err, io.ErrUnexpectedEOF):
	default:
		return ErrStream
	}
	sealed = sealed[:n]
	final := byte(notFinalFlag)
	if n < sealedChunk {
		final = finalFlag
	} else if _, err := s.r.Peek(1); errors.Is(err, io.EOF) {
		final = finalFlag
	}
	plain, err := s.k.aead.Open(nil, chunkNonce(s.prefix, s.n), sealed, chunkAAD(s.aad, s.n, final))
	if err != nil {
		return ErrStream
	}
	s.plain, s.n, s.done = plain, s.n+1, final == finalFlag
	return nil
}
