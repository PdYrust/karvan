# ADR 0008: Migration progress goes out as server-sent events
## Status
Accepted, 2026-10-16
## Context
A migration of many users takes minutes. The web panel must show each stage live ("backing up", "applying", "undoing"...), survive a dropped connection or a reload of the page, and never show an event twice.
## Decision
- Every event is stored first in `job_events` with a sequence number per job, then sent to live subscribers. A slow subscriber may miss live events; it never blocks the job.
- `GET /api/v1/migrations/{id}/events` is a server-sent event stream: `id` is the sequence number, `event` the type (`stage`, `progress`, `log`, `done`), `data` the JSON event. Events hold codes, never prose or secrets; the web panel words them.
- A client resumes with `Last-Event-ID` (or `?after=` where it cannot set headers) and gets only what it missed. When the stream notices a gap it reads the missing events from the table before going on.
- The stream turns off the server's write timeout for itself and sends a comment every 20 seconds so proxies keep it open.
- A job reads as finished before its `done` event goes out, so a client may ask for the report as soon as it sees `done`.
## Alternatives considered
- **WebSocket.** Two-way is not needed; SSE reconnects by itself and passes proxies as plain HTTP.
- **Polling the job.** Misses short stages and costs a request per second per open page.
## Consequences
- `job_events` grows with every job; trimming old jobs comes with job history in the web panel (Phase 4).
