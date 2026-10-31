package run

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/PdYrust/karvan/internal/migration/dest"
	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
)

// VerifyError lists users that did not read back as planned.
type VerifyError struct{ Users []string }

func (e *VerifyError) Error() string {
	return fmt.Sprintf("%d users did not read back as written", len(e.Users))
}

// verify reads back every user the job wrote and compares name, created
// time, owner, credentials and targets with the plan.
func (s *Service) verify(ctx context.Context, j *Job, p *prepared, rep *Report) error {
	return s.stage(j, rep, StageVerify, func() error {
		rows, err := s.ledger.Job(ctx, j.ID, ledger.Done)
		if err != nil {
			return err
		}
		byKey := map[string]*engine.Item{}
		for _, it := range j.result.Items {
			byKey[it.Key] = it
		}
		var ids []int64
		for _, e := range rows {
			if e.Kind == ledger.KindUser {
				ids = append(ids, e.Row.ID)
			}
		}
		r, err := paneldb.OpenReader(ctx, p.target)
		if err != nil {
			return err
		}
		defer r.Close()
		var bad []string
		err = r.View(ctx, func(q paneldb.Querier) error {
			back, err := p.writer.ReadBack(ctx, q, ids)
			if err != nil {
				return err
			}
			for _, e := range rows {
				if e.Kind != ledger.KindUser {
					continue
				}
				rb, found := back[e.Row.ID]
				if !matches(byKey[e.SourceKey], rb, found, j.facts.Kind, p.admins[e.SourceAdmin], p.db.Server().Dialect) {
					bad = append(bad, e.SourceKey)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		rep.Counts.Verified = len(ids) - len(bad)
		if len(bad) > 0 {
			return &VerifyError{Users: bad}
		}
		return nil
	})
}

// matches compares a read-back row with the planned user.
func matches(it *engine.Item, b dest.ReadBack, found bool, kind panel.Kind, adminID int64, d paneldb.Dialect) bool {
	if it == nil || !found {
		return false
	}
	keys := it.User.Credentials.Keys()
	slices.Sort(keys)
	var targets []string
	for _, t := range it.Targets {
		proto, _, isInbound := strings.Cut(t, "/")
		if kind == panel.Marzban && isInbound && !hasProtocol(it.User.Credentials, proto) {
			continue // Marzban keeps inbounds per protocol the user has
		}
		targets = append(targets, t)
	}
	slices.Sort(targets)
	return b.Username == it.User.Username && b.CreatedAt.Equal(paneldb.Stored(d, it.User.CreatedAt)) &&
		b.AdminID == adminID && slices.Equal(keys, b.Keys) && slices.Equal(targets, b.Targets)
}

func hasProtocol(c panel.Credentials, proto string) bool {
	for _, k := range c.Keys() {
		if strings.HasPrefix(string(k), proto+":") {
			return true
		}
	}
	return false
}
