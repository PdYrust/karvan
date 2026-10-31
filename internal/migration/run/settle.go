package run

import (
	"context"
	"time"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
)

// settle resolves pending ledger rows for this source and destination by
// reading the destination: a row that still has the recorded id,
// username and created_at committed; anything else is unknown, which
// behaves like "moved, then deleted" so nobody is created twice. It only
// updates the ledger.
func (s *Service) settle(ctx context.Context, j *Job, p *prepared) error {
	live, err := s.ledger.Live(ctx, j.origin, p.dest)
	if err != nil {
		return err
	}
	var pending []ledger.Entry
	for _, e := range live {
		if e.State == ledger.Pending {
			pending = append(pending, e)
		}
	}
	if len(pending) == 0 {
		return nil
	}
	r, err := paneldb.OpenReader(ctx, p.target)
	if err != nil {
		return err
	}
	defer r.Close()
	var done, unknown []int64
	err = r.View(ctx, func(q paneldb.Querier) error {
		for _, e := range pending {
			ok, err := present(ctx, q, e)
			if err != nil {
				return err
			}
			if ok {
				done = append(done, e.ID)
			} else {
				unknown = append(unknown, e.ID)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := s.ledger.Set(ctx, ledger.Done, done); err != nil {
		return err
	}
	return s.ledger.Set(ctx, ledger.Unknown, unknown)
}

// present reads one row back; both panels name their tables admins and
// users with id, username and created_at.
func present(ctx context.Context, q paneldb.Querier, e ledger.Entry) (bool, error) {
	table := "users"
	if e.Kind == ledger.KindAdmin {
		table = "admins"
	}
	rows, err := q.QueryContext(ctx, "SELECT username, created_at FROM "+table+" WHERE id = ?", e.Row.ID)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return false, rows.Err()
	}
	var name string
	var created paneldb.NullTime
	if err := rows.Scan(&name, &created); err != nil {
		return false, err
	}
	return name == e.Username && sameMoment(created.Time, e.CreatedAt), rows.Err()
}

// sameMoment compares at microseconds, the finest any backend keeps; the
// ledger stores times already cut to the column's precision.
func sameMoment(a, b time.Time) bool {
	return a.UTC().Truncate(time.Microsecond).Equal(b.UTC().Truncate(time.Microsecond))
}
