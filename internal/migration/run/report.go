package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/PdYrust/karvan/internal/migration/dest"
	"github.com/PdYrust/karvan/internal/migration/engine"
)

// ReportSchema names the JSON report format (D52).
const ReportSchema = "karvan.report/v1"

// Report is the full record of one run. It holds no secret: no password,
// UUID or token.
type Report struct {
	Schema    string    `json:"schema"`
	JobID     string    `json:"jobId"`
	State     string    `json:"state"`
	SafeMode  bool      `json:"safeMode"`
	Rollback  string    `json:"rollback,omitempty"` // what Safe Mode does on failure
	BackupID  string    `json:"backupId,omitempty"`
	StartedAt time.Time `json:"startedAt"`
	EndedAt   time.Time `json:"endedAt"`
	Source    struct {
		Kind   string `json:"kind"`
		Origin string `json:"origin"`
		Admin  string `json:"admin"`
	} `json:"source"`
	Destination struct {
		Kind string `json:"kind"`
		Name string `json:"name"`
		DB   string `json:"db"`
	} `json:"destination"`
	Admin struct {
		Username string `json:"username"`
		ID       int64  `json:"id,omitempty"`
		Merged   bool   `json:"merged"`
		RoleID   int64  `json:"roleId,omitempty"`
		Password string `json:"password,omitempty"` // kept, typed or generated
	} `json:"admin"` // the login's admin, as in Admins
	Admins      []AdminLine       `json:"admins"` // every admin that moved, the login's first (D34)
	Counts      Counts            `json:"counts"`
	Users       []UserLine        `json:"users"`
	Conflicts   []engine.Conflict `json:"conflicts,omitempty"`
	NotUndoable []string          `json:"notUndoable"`
	Reload      ReloadResult      `json:"reload"`
	Stages      []StageTime       `json:"stages"`
	Error       string            `json:"error,omitempty"`
	Left        []int64           `json:"left,omitempty"` // rows a rollback could not remove
	// LeftReasons says why a row of Left stayed, by destination id, when
	// Karvan knows: LeftInBatch.
	LeftReasons map[int64]string `json:"leftReasons,omitempty"`
}

// LeftInBatch is the reason for a row a later batch holds (D14): a
// rollback leaves it.
const LeftInBatch = "in_batch"

// Counts sum up the users.
type Counts struct {
	Read    int `json:"read"`
	Moved   int `json:"moved"`
	Skipped int `json:"skipped"`
	Adopted int `json:"adopted"`
	Undone  int `json:"undone"`
	Changed int `json:"configChanged"`
	Renamed int `json:"renamed"`
	// ProtocolsAdded counts users who got protocols they lacked (D67).
	ProtocolsAdded int `json:"protocolsAdded"`
	Verified       int `json:"verified"`
}

// AdminLine is where one source admin went (D34).
type AdminLine struct {
	Source   string `json:"source"`   // its name on the old panel
	Username string `json:"username"` // its name on the destination
	IsLogin  bool   `json:"isLogin"`
	ID       int64  `json:"id,omitempty"`
	Merged   bool   `json:"merged"`
	RoleID   int64  `json:"roleId,omitempty"`
	Password string `json:"password,omitempty"` // kept, typed or generated; none on a merge
	Users    int    `json:"users"`              // its users the run moves
}

// UserLine is one user's outcome.
type UserLine struct {
	Key            string   `json:"key"`
	Admin          string   `json:"admin"` // its source admin
	SourceUsername string   `json:"sourceUsername"`
	Username       string   `json:"username"`
	Moved          bool     `json:"moved"`
	DestID         int64    `json:"destId,omitempty"`
	Applied        []string `json:"applied,omitempty"`
	Fixed          []string `json:"fixed,omitempty"`
	ChangesConfig  bool     `json:"changesConfig"`
	ChangesLink    bool     `json:"changesLink"`
	NewLink        string   `json:"newLink,omitempty"`
}

// StageTime is how long a stage took and how it ended.
type StageTime struct {
	Stage   string    `json:"stage"`
	Started time.Time `json:"started"`
	Ended   time.Time `json:"ended"`
	OK      bool      `json:"ok"`
}

// Things a rollback cannot take back (D40).
const (
	UndoNotifications = "notifications_sent"
	UndoConnections   = "client_connections"
	UndoAfterReload   = "after_reload"
	UndoCoreLoaded    = "core_may_have_loaded"
	UndoInboundRows   = "inbound_tag_rows"
	// UndoLostPanelWrites: putting the backup back also took back what
	// other admins and the panel wrote during the job.
	UndoLostPanelWrites = "restore_lost_panel_writes" // Marzban inbound rows and their default hosts
)

func newReport(j *Job, start time.Time) *Report {
	r := &Report{Schema: ReportSchema, JobID: j.ID, SafeMode: j.safeMode, Rollback: j.rollbackOrUndo(), StartedAt: start,
		NotUndoable: []string{UndoNotifications, UndoConnections, UndoCoreLoaded}}
	r.Source.Kind, r.Source.Origin = string(j.kind), j.origin
	if j.snapshot != nil {
		r.Source.Admin = j.snapshot.Admin.Username
	}
	r.Destination.Kind = string(j.facts.Kind)
	if j.facts.Kind == "marzban" {
		r.NotUndoable = append(r.NotUndoable, UndoInboundRows)
	}
	r.Conflicts = j.result.Conflicts
	r.Admin.Username, r.Admin.RoleID, r.Admin.Password = j.result.Login().Username, j.result.Login().RoleID, j.passwordFrom
	r.Admins = reportAdmins(j)
	for _, it := range j.result.Items {
		line := UserLine{Key: it.Key, Admin: it.Admin, SourceUsername: it.Source.Username, Username: it.User.Username, Applied: it.Applied,
			ChangesConfig: it.ChangesConfig, ChangesLink: it.ChangesLink}
		for _, c := range it.Resolved {
			line.Fixed = append(line.Fixed, c.Type+":"+c.Fix)
		}
		r.Users = append(r.Users, line)
		r.Counts.Read++
		switch {
		case it.Adopt != nil:
			r.Counts.Adopted++
		case !it.Include:
			r.Counts.Skipped++
		}
		if it.ChangesConfig && it.Include {
			r.Counts.Changed++
		}
		if it.User.Username != it.Source.Username && it.Include {
			r.Counts.Renamed++
		}
		if it.Include && slices.ContainsFunc(it.Applied, func(a string) bool { return strings.HasPrefix(a, "added:") }) {
			r.Counts.ProtocolsAdded++
		}
	}
	return r
}

// wroteAdmin records where source admin src went: its destination id,
// and whether that admin was there already.
func (r *Report) wroteAdmin(src string, id int64, merged bool) {
	for i := range r.Admins {
		if a := &r.Admins[i]; a.Source == src {
			a.ID, a.Merged = id, merged
			if a.IsLogin {
				r.Admin.ID, r.Admin.Merged = id, merged
			}
		}
	}
}

// adminID is the destination id of a user's admin. A report from before
// several admins has one admin, its Admin.
func (r *Report) adminID(src string) int64 {
	for _, a := range r.Admins {
		if a.Source == src {
			return a.ID
		}
	}
	return r.Admin.ID
}

// addLinks mints a new link for every moved user that has none, with
// its own admin's link settings.
func (r *Report) addLinks(ctx context.Context, d *dest.Panel, now time.Time) error {
	byAdmin := map[int64]dest.Links{}
	for i := range r.Users {
		u := &r.Users[i]
		if !u.Moved || u.NewLink != "" {
			continue
		}
		id := r.adminID(u.Admin)
		l, ok := byAdmin[id]
		if !ok {
			var err error
			if l, err = d.Links(ctx, id); err != nil {
				return err
			}
			byAdmin[id] = l
		}
		u.NewLink = l.Link(u.DestID, u.Username, now)
	}
	return nil
}

// reportAdmins lists the admins the run writes; IDs come as they are.
func reportAdmins(j *Job) []AdminLine {
	out := []AdminLine{}
	for _, a := range j.result.Admins {
		if !a.Moves() {
			continue
		}
		line := AdminLine{Source: a.Source.Username, Username: a.Username, IsLogin: a.IsLogin, RoleID: a.RoleID, Users: a.Users}
		switch {
		case a.IsLogin:
			line.Password = j.passwordFrom
		case a.Merge == nil:
			line.Password = PasswordGenerated
		}
		out = append(out, line)
	}
	return out
}

func (r *Report) stage(name string, start time.Time, ok bool, now time.Time) {
	r.Stages = append(r.Stages, StageTime{Stage: name, Started: start, Ended: now, OK: ok})
}

func (r *Report) moved(key string, id int64) {
	i := slices.IndexFunc(r.Users, func(u UserLine) bool { return u.Key == key })
	if i >= 0 {
		r.Users[i].Moved, r.Users[i].DestID = true, id
		r.Counts.Moved++
	}
}

// Report returns the report of a job that has run: from memory, or as
// stored when the job is from before a restart. It is nil before the
// first run and ErrBusy while the job runs.
func (s *Service) Report(jobID string) (*Report, error) {
	j, err := s.job(jobID)
	if errors.Is(err, ErrNoJob) {
		return s.storedReport(jobID)
	}
	if err != nil {
		return nil, err
	}
	if !j.lockIdle() {
		return nil, ErrBusy
	}
	defer j.mu.Unlock()
	return j.report, nil
}

// WithoutLinks is the report as Karvan keeps it: without the new links, which
// work as passwords. They are minted again when asked for (WithLinks).
func (r *Report) WithoutLinks() *Report {
	c := *r
	c.Users = slices.Clone(r.Users)
	for i := range c.Users {
		c.Users[i].NewLink = ""
	}
	return &c
}

// WithLinks returns the report with a fresh new link for every moved user
// that has none (a report from before a restart).
func (s *Service) WithLinks(ctx context.Context, jobID string) (*Report, error) {
	rep, err := s.Report(jobID)
	if err != nil || rep == nil {
		return rep, err
	}
	if !slices.ContainsFunc(rep.Users, func(u UserLine) bool { return u.Moved && u.NewLink == "" }) {
		return rep, nil
	}
	var panelID int64
	if err := s.db.QueryRowContext(ctx, "SELECT panel_id FROM jobs WHERE id = ?", jobID).Scan(&panelID); err != nil {
		return nil, err
	}
	env, err := s.envPath(ctx, panelID)
	if err != nil {
		return nil, err
	}
	d, err := dest.Open(ctx, env)
	if err != nil {
		return nil, err
	}
	out := *rep
	out.Users = slices.Clone(rep.Users)
	if err := out.addLinks(ctx, d, s.now()); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Service) storedReport(jobID string) (*Report, error) {
	var raw string
	if err := s.db.QueryRow("SELECT report FROM jobs WHERE id = ?", jobID).Scan(&raw); err != nil {
		return nil, ErrNoJob
	}
	if raw == "" {
		return nil, nil
	}
	var r Report
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return nil, fmt.Errorf("stored report of %s: %w", jobID, err)
	}
	return &r, nil
}
