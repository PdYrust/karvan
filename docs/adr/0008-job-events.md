# ADR 0008: Migration progress goes out as server-sent events
## Status
Accepted, 2026-10-16. Changed 2026-10-21: the compare runs in the background and reports on the same stream, and a run whose re-check finds changes ends in checked. Changed 2026-10-24: a rollback on request runs in the background on the same stream, with undo progress and a done event.
## Context
A migration of many users takes minutes. The web panel must show each stage live ("backing up", "applying", "undoing"...), survive a dropped connection or a reload of the page, and never show an event twice. Reading a large old panel before the plan (the compare) can take minutes too, longer than one request may last, and the operator needs a counter while it runs.
## Decision
- Every event is stored first in `job_events` with a sequence number per job, then sent to live subscribers. A slow subscriber may miss live events; it never blocks the job.
- `GET /api/v1/migrations/{id}/events` is a server-sent event stream: `id` is the sequence number, `event` the type (`stage`, `progress`, `log`, `done`), `data` the JSON event. Events hold codes, never prose or secrets; the web panel words them.
- A client resumes with `Last-Event-ID` (or `?after=` where it cannot set headers) and gets only what it missed. When the stream notices a gap it reads the missing events from the table before going on.
- The stream turns off the server's write timeout for itself and sends a comment every 20 seconds so proxies keep it open.
- A job reads as finished before its `done` event goes out, so a client may ask for the report as soon as it sees `done`.
- `POST .../compare` answers 202 at once and reads both panels in the background, on the job's stream:
  - the `read` stage, with a `progress` event for each page of users: users read of the total, and the pass (a list is read until two passes agree);
  - then the `build` stage, while Karvan reads the destination and builds the plan.
  - It ends with the `build` stage done, or with a `log` event that carries the error's code (and the certificate's fingerprint, subject and expiry when the old panel's certificate is not trusted), followed by the failed stage. A compare sends no `done`: that event ends a run.
- While a compare reads, readers keep the job's previous build; a plan change, a second compare and a run answer busy. Cancel and shutdown stop the read.
- A run whose re-check finds the outcome changed writes nothing, logs `plan_stale` and ends with `done` in state `checked`. It keeps the old panel's login, so the operator reviews the changes and runs the new plan version.
- `POST .../rollback` answers 202 once it holds the destination's lock (409 `not_rollbackable` for a job that is not failed or interrupted, 409 `destination_locked` when another job writes there) and undoes in the background, on the job's stream:
  - the `undo` stage, with a `progress` event before the first chunk and after each one: the job's user rows handled (removed, or found gone already) of the total. A run's own Safe Mode undo sends the same events;
  - the `restore` stage when Safe Mode puts the whole backup back;
  - then `done` with `rolled_back`, or `failed` when rows are left. Before `done` the job's report is stored again with the undo counted, so the job list and the overview count what the rollback took back.
## Alternatives considered
- **WebSocket.** Two-way is not needed; SSE reconnects by itself and passes proxies as plain HTTP.
- **Polling the job.** Misses short stages and costs a request per second per open page.
- **A compare that answers when it is done, with a longer timeout.** The page would wait with no counter, and a proxy in front of Karvan may cut the request.
## Consequences
- `job_events` grows with every job. Each compare adds two progress events for every 500 users of the old panel. The events of a job that ended more than 90 days ago are trimmed daily down to its last one, and deleting a draft deletes its events.
- A client that replays a whole stream sees every compare before the run. Each compare starts with its `read` stage, so a client keeps the last one only.
- A job that was rolled back on request has two `done` events: the run's, then the rollback's. A client that replays the stream takes the last one.
