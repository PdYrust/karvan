package marzbanapi

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzban"
	"github.com/PdYrust/karvan/internal/panel/source"
)

// adminJSON is GET /api/admin (app/models/admin.py:22).
type adminJSON struct {
	Username       string  `json:"username"`
	IsSudo         *bool   `json:"is_sudo"`
	TelegramID     *int64  `json:"telegram_id"`
	DiscordWebhook *string `json:"discord_webhook"`
	UsersUsage     *int64  `json:"users_usage"`
}

// userJSON is UserResponse (app/models/user.py:282).
type userJSON struct {
	Username            string                     `json:"username"`
	Status              string                     `json:"status"`
	UsedTraffic         *int64                     `json:"used_traffic"`
	LifetimeUsedTraffic int64                      `json:"lifetime_used_traffic"`
	CreatedAt           string                     `json:"created_at"`
	Proxies             map[string]json.RawMessage `json:"proxies"`
	Inbounds            map[string][]string        `json:"inbounds"`
	Expire              *int64                     `json:"expire"`
	DataLimit           *int64                     `json:"data_limit"`
	ResetStrategy       string                     `json:"data_limit_reset_strategy"`
	Note                *string                    `json:"note"`
	SubUpdatedAt        *string                    `json:"sub_updated_at"`
	SubLastUserAgent    *string                    `json:"sub_last_user_agent"`
	OnlineAt            *string                    `json:"online_at"`
	OnHoldExpire        *int64                     `json:"on_hold_expire_duration"`
	OnHoldTimeout       *string                    `json:"on_hold_timeout"`
	AutoDeleteInDays    *int                       `json:"auto_delete_in_days"`
	NextPlan            *nextPlanJSON              `json:"next_plan"`
	SubscriptionURL     string                     `json:"subscription_url"`
	Admin               *struct {
		Username string `json:"username"`
	} `json:"admin"`
}

type nextPlanJSON struct {
	DataLimit           *int64 `json:"data_limit"`
	Expire              *int64 `json:"expire"`
	AddRemainingTraffic bool   `json:"add_remaining_traffic"`
	FireOnEither        bool   `json:"fire_on_either"`
}

type usersJSON struct {
	Users []userJSON `json:"users"`
	Total *int       `json:"total"`
}

var (
	statuses   = []string{"active", "disabled", "limited", "expired", "on_hold"}
	strategies = []string{"no_reset", "day", "week", "month", "year"}
)

func (a adminJSON) toAdmin() (panel.Admin, error) {
	if a.Username == "" || a.IsSudo == nil {
		return panel.Admin{}, source.BadResponse("admin: missing username or is_sudo")
	}
	out := panel.Admin{Username: a.Username, IsOwner: *a.IsSudo, TelegramID: a.TelegramID}
	if a.DiscordWebhook != nil {
		out.DiscordWebhook = *a.DiscordWebhook
	}
	if a.UsersUsage != nil {
		out.UsedTraffic = *a.UsersUsage
	}
	return out, nil
}

// toUser validates one user. Any doubt stops the read with the username,
// because a silently wrong UUID would break that user's config.
func (j userJSON) toUser() (panel.User, error) {
	u := panel.User{Username: j.Username, Status: panel.Status(j.Status), ResetStrategy: panel.ResetStrategy(j.ResetStrategy),
		LifetimeUsed: j.LifetimeUsedTraffic, SubscriptionURL: j.SubscriptionURL, SourceInbounds: j.Inbounds}
	fail := func(f string, a ...any) (panel.User, error) {
		return panel.User{}, source.BadResponse("user %q: %s", j.Username, fmt.Sprintf(f, a...))
	}
	switch {
	case j.Username == "":
		return fail("no username")
	case !slices.Contains(statuses, j.Status):
		return fail("status %q", j.Status)
	case !slices.Contains(strategies, j.ResetStrategy):
		return fail("reset strategy %q", j.ResetStrategy)
	case j.UsedTraffic == nil || *j.UsedTraffic < 0:
		return fail("used traffic")
	case len(j.Proxies) == 0:
		return fail("no proxies")
	}
	u.UsedTraffic = *j.UsedTraffic
	if err := j.times(&u); err != nil {
		return fail("%v", err)
	}
	for proto, raw := range j.Proxies {
		if !slices.Contains(marzban.Protocols, proto) {
			return fail("protocol %q", proto)
		}
		if err := marzban.SetProxy(&u.Credentials, proto, raw); err != nil {
			return fail("%v", err)
		}
	}
	j.limits(&u)
	if j.Admin != nil {
		u.OwnerUsername = j.Admin.Username
	}
	return u, nil
}

func (j userJSON) times(u *panel.User) error {
	created, err := source.ParseTime(j.CreatedAt)
	if err != nil {
		return err
	}
	u.CreatedAt = created
	if u.OnlineAt, err = source.ParseTimePtr(j.OnlineAt); err != nil {
		return err
	}
	if u.SubUpdatedAt, err = source.ParseTimePtr(j.SubUpdatedAt); err != nil {
		return err
	}
	if u.OnHoldTimeout, err = source.ParseTimePtr(j.OnHoldTimeout); err != nil {
		return err
	}
	if j.Expire != nil && *j.Expire > 0 {
		t := time.Unix(*j.Expire, 0).UTC()
		u.Expire = &t
	}
	return nil
}

func (j userJSON) limits(u *panel.User) {
	if j.DataLimit != nil {
		u.DataLimit = *j.DataLimit
	}
	if j.Note != nil {
		u.Note = *j.Note
	}
	if j.SubLastUserAgent != nil {
		u.SubUserAgent = *j.SubLastUserAgent
	}
	if j.OnHoldExpire != nil {
		d := time.Duration(*j.OnHoldExpire) * time.Second
		u.OnHoldExpire = &d
	}
	u.AutoDeleteIn = j.AutoDeleteInDays
	if p := j.NextPlan; p != nil {
		np := &panel.NextPlan{AddRemainingTraffic: p.AddRemainingTraffic, FireOnEither: p.FireOnEither}
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
