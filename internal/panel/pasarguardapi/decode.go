package pasarguardapi

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/pasarguard"
	"github.com/PdYrust/karvan/internal/panel/source"
)

// adminJSON is AdminDetails (app/models/admin.py:124). Shape C has role;
// shape B (before 66c38b8a687a) has is_sudo instead.
type adminJSON struct {
	ID             *int64  `json:"id"`
	Username       string  `json:"username"`
	TelegramID     *int64  `json:"telegram_id"`
	DiscordWebhook *string `json:"discord_webhook"`
	UsedTraffic    *int64  `json:"used_traffic"`
	Note           *string `json:"note"`
	IsSudo         *bool   `json:"is_sudo"`
	Role           *struct {
		Name    string `json:"name"`
		IsOwner bool   `json:"is_owner"`
	} `json:"role"`
}

// userJSON is UserResponse (app/models/user.py:118-142).
type userJSON struct {
	ID               *int64          `json:"id"`
	Username         string          `json:"username"`
	Status           string          `json:"status"`
	UsedTraffic      *int64          `json:"used_traffic"`
	LifetimeUsed     int64           `json:"lifetime_used_traffic"`
	CreatedAt        string          `json:"created_at"`
	OnlineAt         *string         `json:"online_at"`
	SubscriptionURL  string          `json:"subscription_url"`
	ProxySettings    json.RawMessage `json:"proxy_settings"`
	Expire           json.RawMessage `json:"expire"`
	DataLimit        *int64          `json:"data_limit"`
	ResetStrategy    *string         `json:"data_limit_reset_strategy"`
	Note             *string         `json:"note"`
	OnHoldExpire     *int64          `json:"on_hold_expire_duration"`
	OnHoldTimeout    json.RawMessage `json:"on_hold_timeout"`
	GroupIDs         []int64         `json:"group_ids"`
	AutoDeleteInDays *int            `json:"auto_delete_in_days"`
	HWIDLimit        *int            `json:"hwid_limit"`
	NextPlan         *struct {
		DataLimit           *int64 `json:"data_limit"`
		Expire              *int64 `json:"expire"`
		AddRemainingTraffic bool   `json:"add_remaining_traffic"`
	} `json:"next_plan"`
	Admin *struct {
		Username string `json:"username"`
	} `json:"admin"`
}

type usersJSON struct {
	Users []userJSON `json:"users"`
	Total *int       `json:"total"`
}

var (
	statuses   = []string{"active", "disabled", "limited", "expired", "on_hold"}
	strategies = []string{"no_reset", "day", "week", "month", "year"}
)

// shape reports which API shape the admin response has.
func (a adminJSON) shape() string {
	switch {
	case a.Role != nil:
		return "rbac"
	case a.IsSudo != nil:
		return "is_sudo"
	}
	return ""
}

func (a adminJSON) toAdmin() (panel.Admin, error) {
	if a.Username == "" || a.shape() == "" {
		return panel.Admin{}, source.BadResponse("admin: no username, or neither role nor is_sudo")
	}
	out := panel.Admin{Username: a.Username, TelegramID: a.TelegramID}
	if a.ID != nil {
		out.SourceID = *a.ID
	}
	if a.Role != nil {
		out.IsOwner = a.Role.IsOwner
	} else {
		out.IsOwner = *a.IsSudo
	}
	if a.DiscordWebhook != nil {
		out.DiscordWebhook = *a.DiscordWebhook
	}
	if a.UsedTraffic != nil {
		out.UsedTraffic = *a.UsedTraffic
	}
	if a.Note != nil {
		out.Note = *a.Note
	}
	return out, nil
}

func (j userJSON) toUser() (panel.User, error) {
	fail := func(f string, a ...any) (panel.User, error) {
		return panel.User{}, source.BadResponse("user %q: %s", j.Username, fmt.Sprintf(f, a...))
	}
	strategy := "no_reset"
	if j.ResetStrategy != nil {
		strategy = *j.ResetStrategy
	}
	switch {
	case j.ID == nil || *j.ID <= 0:
		return fail("no id")
	case j.Username == "":
		return fail("no username")
	case !slices.Contains(statuses, j.Status):
		return fail("status %q", j.Status)
	case !slices.Contains(strategies, strategy):
		return fail("reset strategy %q", strategy)
	case j.UsedTraffic == nil || *j.UsedTraffic < 0:
		return fail("used traffic")
	}
	c, err := pasarguard.DecodeProxySettings(j.ProxySettings)
	if err != nil {
		return fail("%v", err)
	}
	u := panel.User{SourceID: *j.ID, Username: j.Username, Status: panel.Status(j.Status), Credentials: c,
		ResetStrategy: panel.ResetStrategy(strategy), UsedTraffic: *j.UsedTraffic, LifetimeUsed: j.LifetimeUsed,
		SubscriptionURL: j.SubscriptionURL, SourceGroups: j.GroupIDs, AutoDeleteIn: j.AutoDeleteInDays, HWIDLimit: j.HWIDLimit}
	if err := j.times(&u); err != nil {
		return fail("%v", err)
	}
	j.limits(&u)
	return u, nil
}

func (j userJSON) times(u *panel.User) error {
	var err error
	if u.CreatedAt, err = source.ParseTime(j.CreatedAt); err != nil {
		return err
	}
	if u.OnlineAt, err = source.ParseTimePtr(j.OnlineAt); err != nil {
		return err
	}
	if u.Expire, err = flexTime(j.Expire); err != nil {
		return err
	}
	u.OnHoldTimeout, err = flexTime(j.OnHoldTimeout)
	return err
}

// flexTime reads a field PasarGuard types as datetime | int | None. The
// response carries a datetime; an int is unix seconds.
func flexTime(raw json.RawMessage) (*time.Time, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var n int64
	if json.Unmarshal(raw, &n) == nil {
		if n <= 0 {
			return nil, nil
		}
		t := time.Unix(n, 0).UTC()
		return &t, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, source.BadResponse("datetime %s", raw)
	}
	return source.ParseTimePtr(&s)
}

func (j userJSON) limits(u *panel.User) {
	if j.DataLimit != nil {
		u.DataLimit = *j.DataLimit
	}
	if j.Note != nil {
		u.Note = *j.Note
	}
	if j.OnHoldExpire != nil {
		d := time.Duration(*j.OnHoldExpire) * time.Second
		u.OnHoldExpire = &d
	}
	if j.Admin != nil {
		u.OwnerUsername = j.Admin.Username
	}
	if p := j.NextPlan; p != nil {
		np := &panel.NextPlan{AddRemainingTraffic: p.AddRemainingTraffic}
		if p.DataLimit != nil {
			np.DataLimit = *p.DataLimit
		}
		if p.Expire != nil {
			d := time.Duration(*p.Expire) * time.Second
			np.Expire = &d
		}
		u.NextPlan = np
	}
}
