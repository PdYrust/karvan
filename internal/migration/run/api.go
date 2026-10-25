package run

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/PdYrust/karvan/internal/migration/dest"
	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/safemode"
)

// Summary is step 1 of the owner's flow and the counts of step 2.
type Summary struct {
	State       string         `json:"state"`
	PlanVersion int            `json:"planVersion"`
	SourceKind  panel.Kind     `json:"sourceKind"`
	Origin      string         `json:"origin"`
	Admin       string         `json:"admin"`
	Users       int            `json:"users"`
	UsedTraffic int64          `json:"usedTraffic"`
	Statuses    map[string]int `json:"statuses"`
	DestKind    panel.Kind     `json:"destKind"`
	Targets     []TargetInfo   `json:"targets"`
	Ready       int            `json:"ready"`    // included, with no open conflict
	Problems    int            `json:"problems"` // with an open conflict
	Skipped     int            `json:"skipped"`
	Moved       int            `json:"movedBefore"` // already in the ledger
	Blockers    int            `json:"blockers"`
	Stable      bool           `json:"stable"`
	// RoleID is the moved admin's role on a PasarGuard destination, and
	// UserTargets where its users go unless an operation says otherwise.
	RoleID      int64    `json:"roleId,omitempty"`
	UserTargets []string `json:"userTargets"`
	// PanelID is the destination; Expiries and Protocols sum up the old
	// panel's users; NewAdmin is the admin the plan makes on the
	// destination (its name, and whether it merges into one there).
	PanelID   int64               `json:"panelId"`
	Expiries  Expiries            `json:"expiries"`
	Protocols map[string]int      `json:"protocols"`
	NewAdmin  engine.AdminSummary `json:"newAdmin"`
	// CanKeepPassword is true when a run without a typed password keeps
	// the reseller's own one; otherwise Karvan generates one.
	CanKeepPassword bool `json:"canKeepPassword"`
}

// Expiries counts users by when they expire on the old panel.
type Expiries struct {
	Expired int `json:"expired"`
	Week    int `json:"week"`  // within 7 days
	Month   int `json:"month"` // within 30 days, after the first 7
	Later   int `json:"later"`
	Never   int `json:"never"`
}

func (e *Expiries) add(expire *time.Time, now time.Time) {
	switch {
	case expire == nil:
		e.Never++
	case !expire.After(now):
		e.Expired++
	case expire.Sub(now) <= 7*24*time.Hour:
		e.Week++
	case expire.Sub(now) <= 30*24*time.Hour:
		e.Month++
	default:
		e.Later++
	}
}

// protocols names the protocols a user has credentials for.
func protocols(c panel.Credentials) []string {
	var out []string
	for p, ok := range map[string]bool{"vmess": c.VMess != nil, "vless": c.VLESS != nil, "trojan": c.Trojan != nil,
		"shadowsocks": c.Shadowsocks != nil, "wireguard": c.WireGuard != nil, "hysteria": c.Hysteria != nil} {
		if ok {
			out = append(out, p)
		}
	}
	return out
}

// TargetInfo is a destination group or inbound users can be put in.
type TargetInfo struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Protocols []string `json:"protocols"`
	Hosts     int      `json:"hosts"` // enabled hosts: one link each per user
	Disabled  bool     `json:"disabled,omitempty"`
}

func targetInfo(d engine.Dest, t panel.Target) TargetInfo {
	return TargetInfo{ID: t.ID, Name: t.Name, Protocols: d.TargetProtocols[t.ID], Hosts: d.TargetHosts[t.ID], Disabled: t.Disabled}
}

// RoleInfo is a destination admin role as the final admin step shows it.
type RoleInfo struct {
	ID              int64              `json:"id"`
	Name            string             `json:"name"`
	IsOwner         bool               `json:"isOwner"`
	AllGroups       bool               `json:"allGroups"` // the role allows every group
	Groups          []TargetInfo       `json:"groups"`
	Limits          panel.RoleLimits   `json:"limits"`
	Features        panel.RoleFeatures `json:"features"`
	RequireTemplate bool               `json:"requireTemplate"`
}

// Roles lists the destination's admin roles and the one the moved admin
// gets now (D65). A Marzban destination has none.
func (s *Service) Roles(jobID string) (int64, []RoleInfo, error) {
	j, err := s.job(jobID)
	if err != nil {
		return 0, nil, err
	}
	if !j.lockIdle() {
		return 0, nil, ErrBusy
	}
	defer j.mu.Unlock()
	if j.result == nil {
		return 0, nil, ErrNotChecked
	}
	out := []RoleInfo{}
	for _, r := range j.facts.Roles {
		ri := RoleInfo{ID: r.ID, Name: r.Name, IsOwner: r.IsOwner, AllGroups: r.Groups == nil || r.IsOwner, Groups: []TargetInfo{},
			Limits: r.Limits, Features: r.Features, RequireTemplate: r.RequireTemplate}
		for _, t := range j.facts.Targets {
			if ri.AllGroups || slices.Contains(r.Groups, t.ID) {
				ri.Groups = append(ri.Groups, targetInfo(j.facts, t))
			}
		}
		out = append(out, ri)
	}
	return j.result.Admin.RoleID, out, nil
}

// Summary sums up a checked job.
func (s *Service) Summary(jobID string) (Summary, error) {
	j, err := s.job(jobID)
	if err != nil {
		return Summary{}, err
	}
	if !j.lockIdle() {
		return Summary{State: StateRunning}, nil
	}
	defer j.mu.Unlock()
	if j.result == nil {
		return Summary{}, ErrNotChecked
	}
	sum := Summary{State: j.state, PlanVersion: j.version, SourceKind: j.kind, Origin: j.origin, DestKind: j.facts.Kind,
		Statuses: map[string]int{}, Blockers: j.result.Blockers, Stable: j.result.Stable, PanelID: j.PanelID,
		Protocols: map[string]int{}, NewAdmin: j.result.Admin.Summary(), CanKeepPassword: j.ownPassword()}
	if j.snapshot != nil {
		sum.Admin = j.snapshot.Admin.Username
	}
	for _, t := range j.facts.Targets {
		sum.Targets = append(sum.Targets, targetInfo(j.facts, t))
	}
	sum.RoleID, sum.UserTargets = j.result.Admin.RoleID, j.result.Targets
	now := s.now()
	for _, it := range j.result.Items {
		sum.Users++
		sum.UsedTraffic += it.Source.UsedTraffic
		sum.Statuses[string(it.Source.Status)]++
		sum.Expiries.add(it.Source.Expire, now)
		for _, p := range protocols(it.Source.Credentials) {
			sum.Protocols[p]++
		}
		_, moved := j.moved[it.Key]
		switch {
		case moved:
			sum.Moved++
		case !it.Include:
			sum.Skipped++
		case len(it.Conflicts) > 0:
			sum.Problems++
		default:
			sum.Ready++
		}
	}
	return sum, nil
}

// TakePasswords returns the generated admin passwords once, then forgets
// them (D22).
func (s *Service) TakePasswords(jobID string) (map[string]string, error) {
	j, err := s.job(jobID)
	if err != nil {
		return nil, err
	}
	if !j.lockIdle() {
		return nil, ErrBusy
	}
	defer j.mu.Unlock()
	out := j.passwords
	j.passwords = map[string]string{}
	return out, nil
}

// StoredTemplate is a saved template.
type StoredTemplate struct {
	ID       int64         `json:"id"`
	Template plan.Template `json:"template"`
	Updated  time.Time     `json:"updatedAt"`
}

// ErrTemplateName means a template name is empty, too long or taken.
var ErrTemplateName = errors.New("template name is empty, too long or taken")

// Templates lists saved templates by name.
func (s *Service) Templates(ctx context.Context) ([]StoredTemplate, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, body, updated_at FROM templates ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("read templates: %w", err)
	}
	defer rows.Close()
	var out []StoredTemplate
	for rows.Next() {
		var t StoredTemplate
		var body string
		var at int64
		if err := rows.Scan(&t.ID, &body, &at); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(body), &t.Template); err != nil {
			return nil, fmt.Errorf("template %d: %w", t.ID, err)
		}
		t.Updated = time.UnixMilli(at).UTC()
		out = append(out, t)
	}
	return out, rows.Err()
}

// SaveTemplate stores the reusable part of a job's plan under name.
func (s *Service) SaveTemplate(ctx context.Context, jobID, name string) (StoredTemplate, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 64 {
		return StoredTemplate{}, ErrTemplateName
	}
	_, p, _, err := s.View(jobID)
	if err != nil {
		return StoredTemplate{}, err
	}
	t := plan.TemplateOf(name, p)
	body, err := json.Marshal(t)
	if err != nil {
		return StoredTemplate{}, err
	}
	now := s.now().UnixMilli()
	res, err := s.db.ExecContext(ctx, "INSERT INTO templates (name, body, created_at, updated_at) VALUES (?, ?, ?, ?)", name, string(body), now, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return StoredTemplate{}, ErrTemplateName
		}
		return StoredTemplate{}, fmt.Errorf("save template: %w", err)
	}
	id, err := res.LastInsertId()
	return StoredTemplate{ID: id, Template: t, Updated: time.UnixMilli(now).UTC()}, err
}

// ErrNotFound means no such template or backup.
var ErrNotFound = errors.New("not found")

// DeleteTemplate removes a template.
func (s *Service) DeleteTemplate(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM templates WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// BackupInfo is a Safe Mode backup on disk.
type BackupInfo struct {
	ID      string    `json:"id"`
	JobID   string    `json:"jobId"`
	Dest    string    `json:"dest"`
	Size    int64     `json:"size"`
	Method  string    `json:"method"`
	Created time.Time `json:"createdAt"`
}

// Backups lists Safe Mode backups, newest first.
func (s *Service) Backups(ctx context.Context) ([]BackupInfo, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, job_id, dest, size, method, created_at FROM safe_mode_backups ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("read backups: %w", err)
	}
	defer rows.Close()
	var out []BackupInfo
	for rows.Next() {
		var b BackupInfo
		var at int64
		if err := rows.Scan(&b.ID, &b.JobID, &b.Dest, &b.Size, &b.Method, &at); err != nil {
			return nil, err
		}
		b.Created = time.UnixMilli(at).UTC()
		out = append(out, b)
	}
	return out, rows.Err()
}

// DeleteBackup removes a backup file and its record. A running job's
// backup stays.
func (s *Service) DeleteBackup(ctx context.Context, id string) error {
	if s.runningBackup(id) {
		return ErrBusy
	}
	b, err := s.backupRecord(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := os.Remove(b.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete backup file: %w", err)
	}
	_, err = s.db.ExecContext(ctx, "DELETE FROM safe_mode_backups WHERE id = ?", id)
	return err
}

// runningBackup reports whether the job that took backup id runs now. It
// asks the jobs table: the job's own fields belong to its run goroutine.
func (s *Service) runningBackup(id string) bool {
	var state string
	err := s.db.QueryRow("SELECT j.state FROM safe_mode_backups b JOIN jobs j ON j.id = b.job_id WHERE b.id = ?", id).Scan(&state)
	return err == nil && state == StateRunning
}

// ErrPanelChanged means the panel a backup was taken from now points at
// another database, so the backup must not be put there.
var ErrPanelChanged = errors.New("the panel's database is not the one this backup was taken from")

// RestoreBackup puts a Safe Mode backup back into the panel it came from
// and retires the job's ledger rows. With stopped false, Karvan stops the
// panel, restores and starts it again (the web panel); with stopped true
// the caller has stopped it (the command line). The destination's lock is
// held throughout, so no job writes meanwhile.
func (s *Service) RestoreBackup(ctx context.Context, id string, stopped bool) error {
	b, jobID, env, d, err := s.backupTarget(ctx, id)
	if err != nil {
		return err
	}
	var ctl PanelControl
	if !stopped {
		if ctl, err = s.control(ctx, env); err != nil {
			return err
		}
	}
	w, err := paneldb.OpenWriter(ctx, d.Target)
	if err != nil {
		return err
	}
	defer w.Close()
	unlock, err := w.Lock(ctx, d.Target)
	if err != nil {
		return err
	}
	defer unlock()
	put := func() error { return safemode.Restorer{Key: s.Key, Run: s.Tools}.Restore(ctx, d.Target, w.Server(), b) }
	if ctl != nil {
		err = restoreStopped(ctx, ctl, put)
	} else {
		err = put()
	}
	if err != nil {
		return err
	}
	if err := s.markRestored(ctx, jobID); err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "UPDATE jobs SET state = ?, updated_at = ? WHERE id = ?", StateRolledBack, s.now().UnixMilli(), jobID)
	if j, jerr := s.job(jobID); jerr == nil && j.lockIdle() {
		j.state = StateRolledBack
		j.mu.Unlock()
	}
	return err
}

// backupTarget finds a backup, its job and the panel it goes back into,
// and refuses one whose panel now points at another database.
func (s *Service) backupTarget(ctx context.Context, id string) (safemode.Backup, string, string, *dest.Panel, error) {
	b, err := s.backupRecord(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return b, "", "", nil, ErrNotFound
	}
	if err != nil {
		return b, "", "", nil, err
	}
	var jobID, destName, env string
	err = s.db.QueryRowContext(ctx, "SELECT b.job_id, b.dest, p.env_path FROM safe_mode_backups b JOIN jobs j ON j.id = b.job_id "+
		"JOIN panels p ON p.id = j.panel_id WHERE b.id = ?", id).Scan(&jobID, &destName, &env)
	if err != nil {
		return b, "", "", nil, fmt.Errorf("find the backup's panel: %w", err)
	}
	if s.runningBackup(id) {
		return b, "", "", nil, ErrBusy
	}
	d, err := dest.Open(ctx, env)
	if err != nil {
		return b, "", "", nil, err
	}
	if d.Target.String() != destName {
		return b, "", "", nil, ErrPanelChanged
	}
	return b, jobID, env, d, nil
}

// controlTTL is how long a panel control probe is trusted. A probe runs
// Docker several times and can take seconds per panel.
const controlTTL = time.Minute

type controlProbe struct {
	err error
	at  time.Time
}

// PanelControlStatus says whether Karvan can stop and start a panel, and
// why not. A probe younger than controlTTL is reused unless fresh is set.
func (s *Service) PanelControlStatus(ctx context.Context, envPath string, fresh bool) error {
	s.mu.Lock()
	p, ok := s.probes[envPath]
	s.mu.Unlock()
	if ok && !fresh && s.now().Sub(p.at) < controlTTL {
		return p.err
	}
	_, err := s.control(ctx, envPath)
	// A probe its request cut short (the page was left) says nothing about
	// the panel; kept, it would show a wrong reason for a minute.
	if ctx.Err() != nil {
		return err
	}
	s.mu.Lock()
	s.probes[envPath] = controlProbe{err: err, at: s.now()}
	s.mu.Unlock()
	return err
}

// PanelRow is a registered destination panel.
type PanelRow struct {
	ID      int64
	EnvPath string
	Name    string
	Kind    string
}

// Panels lists the registered panels.
func (s *Service) Panels(ctx context.Context) ([]PanelRow, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, env_path, name, kind FROM panels ORDER BY name, id")
	if err != nil {
		return nil, fmt.Errorf("read panels: %w", err)
	}
	defer rows.Close()
	var out []PanelRow
	for rows.Next() {
		var p PanelRow
		if err := rows.Scan(&p.ID, &p.EnvPath, &p.Name, &p.Kind); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Items returns the job's users as the last build left them.
func (s *Service) Items(jobID string) ([]*engine.Item, error) {
	r, _, _, err := s.View(jobID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrNotChecked
	}
	return r.Items, nil
}

// JobState reads a job's state, also for jobs from before a restart.
func (s *Service) JobState(ctx context.Context, jobID string) (string, error) {
	if j, err := s.job(jobID); err == nil {
		return j.State(), nil
	}
	var state string
	err := s.db.QueryRowContext(ctx, "SELECT state FROM jobs WHERE id = ?", jobID).Scan(&state)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNoJob
	}
	return state, err
}
