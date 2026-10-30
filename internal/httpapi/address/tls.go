package address

import (
	"bufio"
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"time"
)

// sniffTimeout is how long a new connection gets to send its first byte
// while one port serves an HTTP and an HTTPS address.
const sniffTimeout = 10 * time.Second

// tlsRecordHandshake is the first byte of every TLS connection.
const tlsRecordHandshake = 0x16

// connListener hands each connection of a port to plain HTTP or to TLS by
// the schemes its addresses use. On a port with an HTTPS address the first
// byte decides: TLS goes to TLS, and plain HTTP is served too, so a change
// of scheme can wait for its confirmation there (D7) and a plain request
// for an HTTPS address can be sent on to it. A connection that does not
// send that byte in time is dropped without holding up the others.
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
	if !c.l.secure() {
		return raw
	}
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

// secure reports whether any of the port's addresses uses HTTPS.
func (l *listener) secure() bool {
	for _, s := range *l.sites.Load() {
		if s.TLS.On() {
			return true
		}
	}
	return false
}

// certificate is the certificate of the port's HTTPS addresses: the first
// one the client supports, the current address first.
func (l *listener) certificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	var last *tls.Certificate
	lastErr := errors.New("no HTTPS address on this port")
	if l.certs == nil {
		return nil, lastErr
	}
	for _, s := range *l.sites.Load() {
		if !s.TLS.On() {
			continue
		}
		c, err := l.certs(hello.Context(), s.TLS)
		if err != nil {
			lastErr = err
			continue
		}
		if hello.SupportsCertificate(c) == nil {
			return c, nil
		}
		last = c
	}
	if last != nil {
		return last, nil
	}
	return nil, lastErr
}
