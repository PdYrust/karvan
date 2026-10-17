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
	} `json:"admin"`
	Counts      Counts            `json:"counts"`
	Users       []UserLine        `json:"users"`
	Conflicts   []engine.Conflict `json:"conflicts,omitempty"`
	NotUndoable []string          `json:"notUndoable"`
	Reload      ReloadResult      `json:"reload"`
	Stages      []StageTime       `json:"stages"`
	Error       string            `json:"error,omitempty"`
	Left        []int64           `json:"left,omitempty"` // rows a rollback could not remove
}

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

// UserLine is one user's outcome.
type UserLine struct {
	Key            string   `json:"key"`
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
	r.Admin.Username, r.Admin.RoleID, r.Admin.Password = j.result.Admin.Username, j.result.Admin.RoleID, j.passwordFrom
	for _, it := range j.result.Items {
		line := UserLine{Key: it.Key, SourceUsername: it.Source.Username, Username: it.User.Username, Applied: it.Applied,
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
	l, err := d.Links(ctx, rep.Admin.ID)
	if err != nil {
		return nil, err
	}
	out := *rep
	out.Users = slices.Clone(rep.Users)
	for i := range out.Users {
		if u := &out.Users[i]; u.Moved && u.NewLink == "" {
			u.NewLink = l.Link(u.DestID, u.Username, s.now())
		}
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
