package run

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/PdYrust/karvan/internal/migration/dest"
	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/safemode"
)

// Summary is step 1 of the wizard and the counts of step 2.
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
	// UserTargets where its users go: the role's groups, or every inbound.
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
	// Admins has one entry per source admin, the login's first;
	// RoleID, UserTargets and NewAdmin above are the login's.
	Admins []AdminInfo `json:"admins"`
}

// AdminInfo is one source admin as the summary and the roles show it.
type AdminInfo struct {
	Username    string              `json:"username"` // its name on the old panel
	IsLogin     bool                `json:"isLogin"`
	Users       int                 `json:"users"` // its users the run moves
	Moves       bool                `json:"moves"` // false: no user of its moves, so it is not written
	RoleID      int64               `json:"roleId"`
	RoleFixed   bool                `json:"roleFixed"` // a merge keeps the existing admin's role
	NewAdmin    engine.AdminSummary `json:"newAdmin"`
	UserTargets []string            `json:"userTargets"`
}

func adminInfos(r *engine.Result) []AdminInfo {
	out := make([]AdminInfo, 0, len(r.Admins))
	for _, a := range r.Admins {
		out = append(out, AdminInfo{Username: a.Source.Username, IsLogin: a.IsLogin, Users: a.Users, Moves: a.Moves(), RoleID: a.RoleID,
			RoleFixed: a.Merge != nil, NewAdmin: a.Summary(), UserTargets: append([]string{}, a.Targets...)})
	}
	return out
}

// Expiries counts users by when they expire on the old panel.
type Expiries struct {
	Expired int `json:"expired"`
	Week    int `json:"week"`  // within 7 days
	Month   int `json:"month"` // within 30 days, after the first 7
	Later   int `json:"later"`
	OnHold  int `json:"onHold"` // on hold without an end: the time starts at the first connection
	Never   int `json:"never"`
}

func (e *Expiries) add(expire *time.Time, status panel.Status, now time.Time) {
	switch {
	case expire == nil && status == panel.StatusOnHold:
		e.OnHold++
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

// RolesView is the destination's admin roles and the one each moved
// admin gets now; RoleID and RoleFixed are the login's admin.
// RoleFixed is true on a merge: the role is the existing admin's own and
// cannot change. A Marzban destination has none.
type RolesView struct {
	RoleID    int64       `json:"roleId"`
	RoleFixed bool        `json:"roleFixed"`
	Admins    []AdminInfo `json:"admins"`
	Roles     []RoleInfo  `json:"roles"`
}

// Roles lists the destination's admin roles for the final admin step.
func (s *Service) Roles(jobID string) (RolesView, error) {
	j, err := s.job(jobID)
	if err != nil {
		return RolesView{}, err
	}
	if !j.lockIdle() {
		return RolesView{}, ErrBusy
	}
	defer j.mu.Unlock()
	if j.result == nil {
		return RolesView{}, ErrNotChecked
	}
	out := RolesView{RoleID: j.result.Login().RoleID, RoleFixed: j.result.Login().Merge != nil, Admins: adminInfos(j.result), Roles: []RoleInfo{}}
	for _, r := range j.facts.Roles {
		ri := RoleInfo{ID: r.ID, Name: r.Name, IsOwner: r.IsOwner, AllGroups: r.Groups == nil || r.IsOwner, Groups: []TargetInfo{},
			Limits: r.Limits, Features: r.Features, RequireTemplate: r.RequireTemplate}
		for _, t := range j.facts.Targets {
			if ri.AllGroups || slices.Contains(r.Groups, t.ID) {
				ri.Groups = append(ri.Groups, targetInfo(j.facts, t))
			}
		}
		out.Roles = append(out.Roles, ri)
	}
	return out, nil
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
		Protocols: map[string]int{}, NewAdmin: j.result.Login().Summary(), CanKeepPassword: j.ownPassword(), Admins: adminInfos(j.result)}
	if j.snapshot != nil {
		sum.Admin = j.snapshot.Admin.Username
	}
	for _, t := range j.facts.Targets {
		sum.Targets = append(sum.Targets, targetInfo(j.facts, t))
	}
	sum.RoleID, sum.UserTargets = j.result.Login().RoleID, j.result.Login().Targets
	now := s.now()
	for _, it := range j.result.Items {
		sum.Users++
		sum.UsedTraffic += it.Source.UsedTraffic
		sum.Statuses[string(it.Source.Status)]++
		sum.Expiries.add(it.Source.Expire, it.Source.Status, now)
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
// them.
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

// ErrNotFound means no such backup or panel.
var ErrNotFound = errors.New("not found")

// BackupInfo is a Safe Mode backup on disk, with the job that took it:
// its panel, its state and the old panel's kind, address and admin.
type BackupInfo struct {
	ID          string    `json:"id"`
	JobID       string    `json:"jobId"`
	Dest        string    `json:"dest"`
	Size        int64     `json:"size"`
	Method      string    `json:"method"`
	Created     time.Time `json:"createdAt"`
	PanelID     int64     `json:"panelId"`
	JobState    string    `json:"jobState"`
	SourceKind  string    `json:"sourceKind"`
	SourceOrig  string    `json:"sourceOrigin"`
	SourceAdmin string    `json:"sourceAdmin"`
}

// Backups lists Safe Mode backups, newest first.
func (s *Service) Backups(ctx context.Context) ([]BackupInfo, error) {
	// A backup whose job row is gone still lists, with the job's fields empty.
	rows, err := s.db.QueryContext(ctx, "SELECT b.id, b.job_id, b.dest, b.size, b.method, b.created_at, COALESCE(j.panel_id, 0), "+
		"COALESCE(j.state, ''), COALESCE(j.source_kind, ''), COALESCE(j.source_origin, ''), COALESCE(j.source_admin, '') "+
		"FROM safe_mode_backups b LEFT JOIN jobs j ON j.id = b.job_id ORDER BY b.created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("read backups: %w", err)
	}
	defer rows.Close()
	var out []BackupInfo
	for rows.Next() {
		var b BackupInfo
		var at int64
		if err := rows.Scan(&b.ID, &b.JobID, &b.Dest, &b.Size, &b.Method, &at, &b.PanelID, &b.JobState, &b.SourceKind, &b.SourceOrig, &b.SourceAdmin); err != nil {
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
