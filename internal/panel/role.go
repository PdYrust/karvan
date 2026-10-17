package panel

import "encoding/json"

// Role is a PasarGuard admin role as Karvan uses it (app/models/admin_role.py
// in 5.4.1). Marzban has none. An admin has exactly one role.
type Role struct {
	ID      int64
	Name    string
	IsOwner bool // bypasses every check below
	// Groups are the group ids the role lets an admin give users; nil
	// means all groups.
	Groups          []string
	Limits          RoleLimits
	Features        RoleFeatures
	RequireTemplate bool // users only from templates; no manual edits
}

// RoleLimits are a role's limits on what an admin gives a user, in the
// panel's own JSON form. Nil is no limit. Durations are seconds.
type RoleLimits struct {
	MaxUsers         *int64 `json:"max_users"`
	DataLimitMin     *int64 `json:"data_limit_min"`
	DataLimitMax     *int64 `json:"data_limit_max"`
	ExpireMin        *int64 `json:"expire_min"`
	ExpireMax        *int64 `json:"expire_max"`
	MinHWID          *int64 `json:"min_hwid_per_user"`
	MaxHWID          *int64 `json:"max_hwid_per_user"`
	OnHoldTimeoutMin *int64 `json:"on_hold_timeout_min,omitempty"`
	OnHoldTimeoutMax *int64 `json:"on_hold_timeout_max,omitempty"`
}

// With returns l with every limit o sets put over it, the way the panel
// merges an admin's permission_overrides into its role's limits.
func (l RoleLimits) With(o *RoleLimits) RoleLimits {
	if o == nil {
		return l
	}
	for _, f := range []struct{ dst, src **int64 }{
		{&l.MaxUsers, &o.MaxUsers}, {&l.DataLimitMin, &o.DataLimitMin}, {&l.DataLimitMax, &o.DataLimitMax},
		{&l.ExpireMin, &o.ExpireMin}, {&l.ExpireMax, &o.ExpireMax}, {&l.MinHWID, &o.MinHWID}, {&l.MaxHWID, &o.MaxHWID},
		{&l.OnHoldTimeoutMin, &o.OnHoldTimeoutMin}, {&l.OnHoldTimeoutMax, &o.OnHoldTimeoutMax},
	} {
		if *f.src != nil {
			*f.dst = *f.src
		}
	}
	return l
}

// RoleFeatures say which user settings a role allows. Both default to
// allowed, as in the panel.
type RoleFeatures struct {
	ResetStrategy bool `json:"can_use_reset_strategy"`
	NextPlan      bool `json:"can_use_next_plan"`
}

// UnmarshalJSON keeps the panel's defaults for missing keys.
func (f *RoleFeatures) UnmarshalJSON(b []byte) error {
	type plain RoleFeatures
	p := plain{ResetStrategy: true, NextPlan: true}
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	*f = RoleFeatures(p)
	return nil
}
