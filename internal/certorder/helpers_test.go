package certorder

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"io"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/secret"
	"github.com/PdYrust/karvan/internal/settings"
	kstore "github.com/PdYrust/karvan/internal/store"
)

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// logBuffer collects every log line, so a test can look for keys in it.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *logBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}
func (l *logBuffer) String() string { l.mu.Lock(); defer l.mu.Unlock(); return l.buf.String() }

// orderEnv is a running bridge with a free challenge port, the fake CA
// and the service, on one store.
type orderEnv struct {
	t     *testing.T
	db    *sql.DB
	key   *secret.Key
	br    *bridge.Bridge
	ca    *fakeCA
	svc   *Service
	clock *clock
	logs  *logBuffer
	mu    sync.Mutex
	txt   map[string][]string
}

func newOrderEnv(t *testing.T) *orderEnv {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	db, err := kstore.Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := secret.CreateKeyFile(filepath.Join(dir, "key")); err != nil {
		t.Fatal(err)
	}
	key, err := secret.LoadKeyFile(filepath.Join(dir, "key"))
	if err != nil {
		t.Fatal(err)
	}
	e := &orderEnv{t: t, db: db, key: key, clock: &clock{t: time.Now().UTC().Truncate(time.Second)}, logs: &logBuffer{},
		txt: map[string][]string{}}
	log := slog.New(slog.NewTextHandler(e.logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	e.br = bridge.New(bridge.Deps{DB: db, Key: key, Now: e.clock.now, Log: log, ChallengePort: freePort(t)})
	e.ca = newFakeCA(t, e.clock.now)
	e.ca.validate = e.validate
	runCtx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { defer close(done); _ = e.br.Run(runCtx) }()
	<-e.br.Started()
	e.svc = e.newService(log)
	t.Cleanup(func() {
		e.svc.stop()
		e.svc.wg.Wait()
		stop()
		<-done
		_ = db.Close()
	})
	return e
}

// newService is a service on the env, as after a start.
func (e *orderEnv) newService(log *slog.Logger) *Service {
	s := New(Deps{DB: e.db, Key: e.key, Bridge: e.br, Settings: settings.New(e.db), Audit: audit.New(e.db), Log: log,
		Now: e.clock.now, TestDirectory: e.ca.url("/dir"), HTTPClient: e.ca.srv.Client(), LookupTXT: e.lookupTXT})
	s.retry, s.wait = time.Millisecond, 5*time.Second
	return s
}

func (e *orderEnv) lookupTXT(_ context.Context, name string) ([]string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	v, ok := e.txt[name]
	if !ok {
		return nil, &net.DNSError{Err: "no such host", Name: name, IsNotFound: true}
	}
	return slices.Clone(v), nil
}

func (e *orderEnv) setTXT(name string, values ...string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.txt[name] = values
}

// validate is the fake CA's check: HTTP-01 asks the bridge's challenge
// port for the token on the order's host; DNS-01 reads the TXT record.
func (e *orderEnv) validate(o *fakeOrder, keyAuth string) *problem {
	if o.chalType == HTTP01 {
		req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:"+strconv.Itoa(e.br.ChallengePort)+
			"/.well-known/acme-challenge/"+o.token, nil)
		req.Host = o.name
		resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
		if err != nil {
			return &problem{status: 403, typ: "connection", detail: "could not connect"}
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK || string(body) != keyAuth {
			return &problem{status: 403, typ: "incorrectResponse", detail: "wrong answer"}
		}
		return nil
	}
	sum := sha256.Sum256([]byte(keyAuth))
	vals, _ := e.lookupTXT(context.Background(), "_acme-challenge."+trimWild(o.name))
	if !slices.Contains(vals, base64.RawURLEncoding.EncodeToString(sum[:])) {
		return &problem{status: 403, typ: "unauthorized", detail: "no TXT record found"}
	}
	return nil
}

func trimWild(name string) string {
	if len(name) > 2 && name[:2] == "*." {
		return name[2:]
	}
	return name
}

// waitState waits for the latest order of domain id to reach one of states.
func (e *orderEnv) waitState(id int64, states ...string) bridge.ACMEOrder {
	e.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		st, err := e.br.ACME(context.Background(), id)
		if err != nil {
			e.t.Fatal(err)
		}
		if st.Order != nil && slices.Contains(states, st.Order.State) {
			return *st.Order
		}
		if time.Now().After(deadline) {
			e.t.Fatalf("order of domain %d: %+v, want %v", id, st.Order, states)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// addACME adds name with a Let's Encrypt certificate by method.
func (e *orderEnv) addACME(name, method string) bridge.Domain {
	e.t.Helper()
	d, err := e.svc.AddDomain(context.Background(), name, method, true)
	if err != nil {
		e.t.Fatalf("add %s by %s: %v", name, method, err)
	}
	return d
}

// dnsDone answers the DNS-01 order of domain d and checks it.
func (e *orderEnv) dnsDone(d bridge.Domain) bridge.ACMEOrder {
	e.t.Helper()
	o := e.waitState(d.ID, "waiting_dns")
	e.setTXT(o.DNS.Name, o.DNS.Value)
	if _, err := e.svc.Check(context.Background(), d.ID, false); err != nil {
		e.t.Fatal(err)
	}
	return e.waitState(d.ID, "done")
}

func (e *orderEnv) domain(id int64) bridge.Domain {
	e.t.Helper()
	d, err := e.br.Domain(context.Background(), id)
	if err != nil {
		e.t.Fatal(err)
	}
	return d
}

func (e *orderEnv) int(q string, args ...any) int64 {
	e.t.Helper()
	var n int64
	if err := e.db.QueryRow(q, args...).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

// handshake is the leaf the bridge serves on port for sni.
func handshake(t *testing.T, port int, sni string) string {
	t.Helper()
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", "127.0.0.1:"+strconv.Itoa(port),
		&tls.Config{ServerName: sni, InsecureSkipVerify: true, MinVersion: tls.VersionTLS12})
	if err != nil {
		return "error: " + err.Error()
	}
	defer conn.Close()
	sum := sha256.Sum256(conn.ConnectionState().PeerCertificates[0].Raw)
	return hex.EncodeToString(sum[:])
}

func dialable(port int) bool {
	c, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(port), time.Second)
	if err == nil {
		_ = c.Close()
	}
	return err == nil
}

// listenOn takes port, as another program would.
func listenOn(t *testing.T, port int) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

// waitCount waits for query q to count n, as background work writes after
// the state the test waited for.
func (e *orderEnv) waitCount(q string, n int64) {
	e.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for e.int(q) != n {
		if time.Now().After(deadline) {
			e.t.Fatalf("%s: %d, want %d", q, e.int(q), n)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
