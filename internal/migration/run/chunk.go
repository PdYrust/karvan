package run

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/pasarguard"
	"github.com/PdYrust/karvan/internal/safemode"
)

// backup takes the Safe Mode backup (D42-D44) and records it.
func (s *Service) backup(ctx context.Context, j *Job, p *prepared) error {
	srv := p.db.Server()
	if err := safemode.Probe(ctx, srv, s.Tools); err != nil {
		return err
	}
	r, err := paneldb.OpenReader(ctx, p.target)
	if err != nil {
		return err
	}
	var size int64
	err = r.View(ctx, func(q paneldb.Querier) error {
		var err error
		size, err = safemode.Size(ctx, q, p.target)
		return err
	})
	_ = r.Close()
	if err != nil {
		return err
	}
	free, err := safemode.Free(s.SafeModeDir())
	if err != nil {
		return err
	}
	if err := safemode.CheckSpace(size, free); err != nil {
		return err
	}
	b, err := safemode.Taker{Dir: s.SafeModeDir(), Key: s.Key, Run: s.Tools, Now: s.now}.Take(ctx, p.target, srv)
	if err != nil {
		return err
	}
	j.backupID = b.ID
	_, err = s.db.ExecContext(ctx, "INSERT INTO safe_mode_backups (id, job_id, dest, path, size, method, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		b.ID, j.ID, p.dest, b.Path, b.Size, b.Method, b.CreatedAt.UnixMilli())
	return err
}

// write creates the admins and the users, chunk by chunk, with the ledger
// protocol of D32.
func (s *Service) write(ctx context.Context, j *Job, p *prepared, hash string, rep *Report) error {
	return s.stage(j, rep, StageExecute, func() error {
		if err := s.adopt(ctx, j, p); err != nil {
			return err
		}
		if err := s.writeAdmins(ctx, j, p, hash, rep); err != nil {
			return err
		}
		items := movable(j.result)
		size := s.ChunkSize(p.db.Server().Dialect)
		started := s.now()
		for n, start := 1, 0; start < len(items); n, start = n+1, start+size {
			if err := ctx.Err(); err != nil {
				return err
			}
			chunk := items[start:min(start+size, len(items))]
			if err := s.writeChunk(ctx, j, p, n, chunk, rep); err != nil {
				return err
			}
			done := min(start+size, len(items))
			s.emit(j, StageExecute, EventProgress, map[string]any{"done": done, "total": len(items), "etaSeconds": eta(started, s.now(), done, len(items))})
		}
		return nil
	})
}

func movable(r *engine.Result) []*engine.Item {
	var out []*engine.Item
	for _, it := range r.Items {
		if it.Include {
			out = append(out, it)
		}
	}
	return out
}

func (s *Service) entry(j *Job, p *prepared, it *engine.Item, state string, chunk int) ledger.Entry {
	return ledger.Entry{Kind: ledger.KindUser, JobID: j.ID, State: state, Chunk: chunk, SourceOrigin: j.origin,
		SourceKey: it.Key, SourceCreated: it.Source.CreatedAt, SourceUsername: it.Source.Username,
		CredPrint: ledger.Fingerprint(it.Source.Credentials), Dest: p.dest, SourceAdmin: it.Admin,
		Row: panel.Row{Username: it.User.Username, CreatedAt: paneldb.Stored(p.db.Server().Dialect, it.User.CreatedAt)}}
}

// adopt records C28 users without writing anything, and retires the
// ledger rows of users the operator chose to move again.
func (s *Service) adopt(ctx context.Context, j *Job, p *prepared) error {
	var entries []ledger.Entry
	var superseded []int64
	for _, it := range j.result.Items {
		if it.Adopt != nil {
			e := s.entry(j, p, it, ledger.Adopted, 0)
			e.Row = panel.Row{ID: it.Adopt.ID, Username: it.Adopt.Username, CreatedAt: it.Adopt.CreatedAt}
			entries = append(entries, e)
		}
		if _, ok := j.moved[it.Key]; ok && it.Include {
			superseded = append(superseded, j.moved[it.Key].DestID)
		}
	}
	if len(superseded) > 0 {
		if err := s.supersede(ctx, j, p, superseded); err != nil {
			return err
		}
	}
	if len(entries) == 0 {
		return nil
	}
	_, err := s.ledger.Add(ctx, entries)
	return err
}

func (s *Service) supersede(ctx context.Context, j *Job, p *prepared, destIDs []int64) error {
	live, err := s.ledger.Live(ctx, j.origin, p.dest)
	if err != nil {
		return err
	}
	var ids []int64
	for _, e := range live {
		if e.Kind == ledger.KindUser && slices.Contains(destIDs, e.Row.ID) {
			ids = append(ids, e.ID)
		}
	}
	return s.ledger.Set(ctx, ledger.Superseded, ids)
}

// newAdmin is an admin the job creates, with its password hash.
type newAdmin struct {
	out   *engine.AdminOutcome
	hash  string
	entry ledger.Entry
}

// writeAdmins records the merges and creates every other admin that
// moves in chunk 0, one ledger row each (D34). The login's admin gets
// hash; every other one a generated password, kept for passwords.csv.
func (s *Service) writeAdmins(ctx context.Context, j *Job, p *prepared, hash string, rep *Report) error {
	p.admins = map[string]int64{}
	var merged []ledger.Entry
	var created []newAdmin
	now := s.now().UTC()
	for i := range j.result.Admins {
		out := &j.result.Admins[i]
		if !out.Moves() {
			continue
		}
		e := ledger.Entry{Kind: ledger.KindAdmin, JobID: j.ID, SourceOrigin: j.origin, SourceKey: out.Source.Username,
			SourceCreated: zeroTime, SourceUsername: out.Source.Username, Dest: p.dest}
		if out.Merge != nil {
			p.admins[out.Source.Username] = out.Merge.ID
			rep.wroteAdmin(out.Source.Username, out.Merge.ID, true)
			e.State, e.Row = ledger.Merged, panel.Row{ID: out.Merge.ID, Username: out.Merge.Username}
			merged = append(merged, e)
			continue
		}
		h, err := s.adminHash(j, out, hash)
		if err != nil {
			return err
		}
		e.State, e.Username, e.CreatedAt = ledger.Pending, out.Username, paneldb.Stored(p.db.Server().Dialect, now)
		created = append(created, newAdmin{out: out, hash: h, entry: e})
	}
	if len(merged) > 0 {
		if _, err := s.ledger.Add(ctx, merged); err != nil {
			return err
		}
	}
	return s.createAdmins(ctx, j, p, created, now, rep)
}

// adminHash is the password hash of an admin the job creates: hash for
// the login's, a generated password's for any other (D34).
func (s *Service) adminHash(j *Job, out *engine.AdminOutcome, hash string) (string, error) {
	if out.IsLogin {
		return hash, nil
	}
	pw, err := pasarguard.GeneratePassword(out.Username)
	if err != nil {
		return "", err
	}
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcryptCost)
	if err != nil {
		return "", err
	}
	j.passwords[out.Username] = pw
	return string(h), nil
}

// createAdmins writes the new admins and their ledger rows in one chunk.
func (s *Service) createAdmins(ctx context.Context, j *Job, p *prepared, created []newAdmin, now time.Time, rep *Report) error {
	if len(created) == 0 {
		return nil
	}
	var ids []int64
	entries := make([]ledger.Entry, len(created))
	err := p.db.Chunk(ctx, func(x paneldb.Execer) error {
		if err := p.writer.Prepare(ctx, x); err != nil {
			return err
		}
		for i, c := range created {
			admin := c.out.Source
			admin.Username = c.out.Username
			id, err := p.writer.InsertAdmin(ctx, x, admin, c.hash, c.out.RoleID, now)
			if err != nil {
				return err
			}
			entries[i] = c.entry
			entries[i].Row.ID = id
		}
		var err error
		if ids, err = s.ledger.Add(ctx, entries); err != nil {
			return err
		}
		p.wrote = true
		return s.fault("admin", 0)
	})
	if err != nil {
		return s.afterFailedChunk(ctx, j, p, ids, err)
	}
	for i, c := range created {
		p.admins[c.out.Source.Username] = entries[i].Row.ID
		rep.wroteAdmin(c.out.Source.Username, entries[i].Row.ID, false)
	}
	return s.ledger.Set(ctx, ledger.Done, ids)
}

// writeChunk is one destination transaction (D32): insert (ids come back),
// ledger rows pending with those ids (Karvan commit), destination commit,
// ledger rows done.
func (s *Service) writeChunk(ctx context.Context, j *Job, p *prepared, n int, items []*engine.Item, rep *Report) error {
	now := s.now().UTC()
	var ids []int64
	entries := make([]ledger.Entry, len(items))
	err := p.db.Chunk(ctx, func(x paneldb.Execer) error {
		for i, it := range items {
			id, err := p.writer.InsertUser(ctx, x, it, p.admins[it.Admin], usageLog(it.User, j.plan.ResetSchedule, now), now)
			if err != nil {
				return err
			}
			entries[i] = s.entry(j, p, it, ledger.Pending, n)
			entries[i].Row.ID = id
		}
		var err error
		if ids, err = s.ledger.Add(ctx, entries); err != nil {
			return err
		}
		p.wrote = true
		return s.fault("chunk", n)
	})
	if err != nil {
		return s.afterFailedChunk(ctx, j, p, ids, err)
	}
	if err := s.fault("ledger", n); err != nil {
		return s.afterFailedChunk(ctx, j, p, ids, errors.Join(paneldb.ErrCommitUnknown, err))
	}
	for i, it := range items {
		rep.moved(it.Key, entries[i].Row.ID)
	}
	return s.ledger.Set(ctx, ledger.Done, ids)
}

// afterFailedChunk settles a chunk whose outcome is not clean (D32, D33):
// rolled back rows become compensated, an unknown commit is read back.
func (s *Service) afterFailedChunk(ctx context.Context, j *Job, p *prepared, ids []int64, cause error) error {
	if len(ids) == 0 {
		return cause
	}
	if !errors.Is(cause, paneldb.ErrCommitUnknown) {
		if err := s.ledger.Set(ctx, ledger.Compensated, ids); err != nil {
			return errors.Join(cause, err)
		}
		return cause
	}
	if err := s.settle(ctx, j, p); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

// zeroTime is the source created_at of an admin: neither API returns it,
// so admins are known by origin and username alone.
var zeroTime = time.Unix(0, 0).UTC()

// fingerprint is a short hash of sorted credential keys, kept in the
// ledger so a later read can tell the same user from a stranger.
func eta(start, now time.Time, done, total int) int {
	if done == 0 {
		return 0
	}
	per := now.Sub(start) / time.Duration(done)
	return int((per * time.Duration(total-done)).Seconds())
}

func (s *Service) fault(step string, n int) error {
	if s.Fault == nil {
		return nil
	}
	if err := s.Fault(step, n); err != nil {
		return fmt.Errorf("fault %s %d: %w", step, n, err)
	}
	return nil
}
