// Package pasarguardapi reads a PasarGuard panel through its API with a
// reseller's login or API key (Idea 1 source). It never writes to the panel.
package pasarguardapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/source"
	"github.com/PdYrust/karvan/internal/secret"
)

// PageSize is how many users one list call asks for (no server cap).
const PageSize = 500

// Login is how the reseller signs in: a password, or an API key
// (pg_key_..., PasarGuard 5.x from c9b48df42f10).
type Login struct {
	Username string
	Password secret.Value
	APIKey   secret.Value
}

// Source is one signed-in reseller on one PasarGuard panel.
type Source struct {
	c *source.Client

	mu    sync.Mutex
	login Login
	token secret.Value
	shape string // "rbac" or "is_sudo", from the admin response
}

// New prepares a source; nothing is sent until the first call.
func New(c *source.Client, l Login) *Source { return &Source{c: c, login: l} }

// Close drops the credentials and token.
func (s *Source) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.login, s.token = Login{}, secret.Value{}
}

// Info reports the panel kind and what the admin response revealed.
func (s *Source) Info(ctx context.Context) (panel.Info, error) {
	if _, err := s.CurrentAdmin(ctx); err != nil {
		return panel.Info{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rbac := s.shape == "rbac"
	return panel.Info{Kind: panel.PasarGuard, Capabilities: panel.Capabilities{
		Groups: true, RBAC: rbac, APIKey: rbac, WireGuard: true, Hysteria: true, HWIDLimit: true,
	}}, nil
}

func (s *Source) signIn(ctx context.Context) error {
	if s.login.Password.IsZero() {
		return errors.New("source has no password")
	}
	resp, err := s.c.PostForm(ctx, "/api/admin/token", url.Values{
		"username": {s.login.Username}, "password": {s.login.Password.Reveal()}, "grant_type": {"password"},
	}, nil)
	if err != nil {
		return err
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := source.DecodeJSON(resp.Body, &tok); err != nil {
		return err
	}
	if tok.AccessToken == "" {
		return source.BadResponse("login: no access_token")
	}
	s.token = secret.NewValue(tok.AccessToken)
	return nil
}

func (s *Source) get(ctx context.Context, path string, q url.Values) (*source.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.login.APIKey.IsZero() {
		return s.c.Get(ctx, path, q, http.Header{"X-Api-Key": {s.login.APIKey.Reveal()}})
	}
	for attempt := 0; ; attempt++ {
		if s.token.IsZero() {
			if err := s.signIn(ctx); err != nil {
				return nil, err
			}
		}
		resp, err := s.c.Get(ctx, path, q, http.Header{"Authorization": {"Bearer " + s.token.Reveal()}})
		if source.CodeOf(err) == source.CodeUnauthorized && attempt == 0 {
			s.token = secret.Value{}
			continue
		}
		return resp, err
	}
}

// CurrentAdmin is the admin whose login was given.
func (s *Source) CurrentAdmin(ctx context.Context) (panel.Admin, error) {
	resp, err := s.get(ctx, "/api/admin", nil)
	if err != nil {
		return panel.Admin{}, err
	}
	var a adminJSON
	if err := source.DecodeJSON(resp.Body, &a); err != nil {
		return panel.Admin{}, err
	}
	s.mu.Lock()
	s.shape = a.shape()
	s.mu.Unlock()
	return a.toAdmin()
}

// Users reads every user the login can see, paged by username with
// load_sub=true (without it PasarGuard leaves subscription_url empty).
func (s *Source) Users(ctx context.Context) ([]panel.User, error) {
	page := func(ctx context.Context, offset, limit int) ([]panel.User, int, error) {
		q := url.Values{"offset": {strconv.Itoa(offset)}, "limit": {strconv.Itoa(limit)},
			"sort": {"username"}, "load_sub": {"true"}}
		resp, err := s.get(ctx, "/api/users", q)
		if err != nil {
			return nil, 0, err
		}
		var body usersJSON
		if err := source.DecodeJSON(resp.Body, &body); err != nil {
			return nil, 0, err
		}
		if body.Total == nil {
			return nil, 0, source.BadResponse("users: no total")
		}
		out := make([]panel.User, 0, len(body.Users))
		for _, j := range body.Users {
			u, err := j.toUser()
			if err != nil {
				return nil, 0, err
			}
			out = append(out, u)
		}
		return out, *body.Total, nil
	}
	return source.ReadAll(ctx, PageSize, page, func(u panel.User) string { return strconv.FormatInt(u.SourceID, 10) })
}
