package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/migration/run"
)

// keepAlive is how often an idle event stream sends a comment, so proxies
// and browsers keep it open.
const keepAlive = 20 * time.Second

// Events streams a job's events as server-sent events (D50). A client
// that reconnects sends Last-Event-ID and gets only what it missed;
// events are numbered, so nothing is shown twice.
func (a *API) Events(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := a.Runs.JobState(r.Context(), id); err != nil {
		writeRunError(w, r, err)
		return
	}
	after := lastEventID(r)
	past, live, stop, err := a.Runs.Events(r.Context(), id, after)
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	s := &stream{w: w, rc: http.NewResponseController(w), last: after, stop: stop, live: live}
	defer func() { s.stop() }()
	_ = s.rc.SetWriteDeadline(time.Time{}) // the server's write timeout would cut the stream
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	if s.sendAll(past) && s.rc.Flush() == nil {
		a.follow(r.Context(), s, id)
	}
}

func lastEventID(r *http.Request) int64 {
	after, _ := strconv.ParseInt(r.Header.Get("Last-Event-ID"), 10, 64)
	if q := r.URL.Query().Get("after"); q != "" && after == 0 {
		after, _ = strconv.ParseInt(q, 10, 64)
	}
	return after
}

// stream writes events in order, each once.
type stream struct {
	w    http.ResponseWriter
	rc   *http.ResponseController
	last int64
	stop func()
	live <-chan run.Event
}

func (s *stream) send(e run.Event) bool {
	if e.Seq <= s.last {
		return true
	}
	data, _ := json.Marshal(e)
	if _, err := fmt.Fprintf(s.w, "id: %d\nevent: %s\ndata: %s\n\n", e.Seq, e.Type, data); err != nil {
		return false
	}
	s.last = e.Seq
	return s.rc.Flush() == nil
}

func (s *stream) sendAll(es []run.Event) bool {
	for _, e := range es {
		if !s.send(e) {
			return false
		}
	}
	return true
}

// follow sends live events until the client goes away. A stream that fell
// behind reads the gap from the table first.
func (a *API) follow(ctx context.Context, s *stream, id string) {
	tick := time.NewTicker(keepAlive)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-s.live:
			if !ok || !a.catchUp(ctx, s, id, e) || !s.send(e) {
				return
			}
		case <-tick.C:
			if _, err := fmt.Fprint(s.w, ": keep-alive\n\n"); err != nil || s.rc.Flush() != nil {
				return
			}
		}
	}
}

func (a *API) catchUp(ctx context.Context, s *stream, id string, e run.Event) bool {
	if e.Seq <= s.last+1 {
		return true
	}
	s.stop()
	past, live, stop, err := a.Runs.Events(ctx, id, s.last)
	if err != nil {
		return false
	}
	s.live, s.stop = live, stop
	return s.sendAll(past)
}
