package address

import (
	"bufio"
	"crypto/tls"
	"errors"
	"net"
	"strings"
	"sync"
	"time"
)

// sniffTimeout is how long a new connection gets to send its first byte.
const sniffTimeout = 10 * time.Second

// tlsRecordHandshake is the first byte of every TLS connection.
const tlsRecordHandshake = 0x16

// connListener hands each connection of a port to plain HTTP or to TLS by
// its first byte: TLS goes to TLS, and plain HTTP is served too, so a
// plain request can be sent on to HTTPS on the same port, or answered
// where that is allowed. A connection that does not send that byte in
// time is dropped without holding up the others.
type connListener struct {
	net.Listener
	l     *listener
	conns chan net.Conn
	errs  chan error
	done  chan struct{}
	start sync.Once
	stop  sync.Once
}

func newConnListener(ln net.Listener, l *listener) *connListener {
	return &connListener{Listener: ln, l: l, conns: make(chan net.Conn), errs: make(chan error), done: make(chan struct{})}
}

func (c *connListener) Accept() (net.Conn, error) {
	c.start.Do(func() { go c.loop() })
	select {
	case conn := <-c.conns:
		return conn, nil
	case err := <-c.errs:
		return nil, err
	case <-c.done:
		return nil, net.ErrClosed
	}
}

func (c *connListener) Close() error {
	c.stop.Do(func() { close(c.done) })
	return c.Listener.Close()
}

// loop accepts until the listener closes. An error goes to Accept, which
// lets http.Server retry the temporary ones.
func (c *connListener) loop() {
	for {
		raw, err := c.Listener.Accept()
		if err != nil {
			select {
			case c.errs <- err:
			case <-c.done:
				return
			}
			if errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}
		go c.hand(raw)
	}
}

func (c *connListener) hand(raw net.Conn) {
	conn := c.wrap(raw)
	if conn == nil {
		_ = raw.Close()
		return
	}
	select {
	case c.conns <- conn:
	case <-c.done:
		_ = raw.Close()
	}
}

// wrap is raw as plain HTTP or TLS, nil when it says nothing in time.
func (c *connListener) wrap(raw net.Conn) net.Conn {
	_ = raw.SetReadDeadline(time.Now().Add(sniffTimeout))
	br := bufio.NewReaderSize(raw, 1)
	first, err := br.Peek(1)
	if err != nil {
		return nil
	}
	_ = raw.SetReadDeadline(time.Time{})
	conn := &peekedConn{Conn: raw, r: br}
	if first[0] == tlsRecordHandshake {
		return tls.Server(conn, c.l.tlsConfig)
	}
	return conn
}

// peekedConn reads the byte that was peeked first.
type peekedConn struct {
	net.Conn
	r *bufio.Reader
}

func (p *peekedConn) Read(b []byte) (int, error) { return p.r.Read(b) }

// errNotOpen refuses a handshake for a name the panel does not open for.
var errNotOpen = errors.New("the panel does not answer this name")

// certificate is the certificate for a handshake: the one the server
// name picks (ADR 0010), none for a name the panel does not open for.
func (s *Server) certificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if s.Certificate == nil {
		return nil, errors.New("no certificate for the panel")
	}
	name := strings.ToLower(strings.TrimSuffix(hello.ServerName, "."))
	if !s.opens(hello.Context(), hello.Conn.RemoteAddr().String(), name) {
		return nil, errNotOpen
	}
	return s.Certificate(hello.Context(), hello.ServerName)
}
