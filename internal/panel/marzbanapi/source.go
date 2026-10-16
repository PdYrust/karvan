// Package marzbanapi reads a Marzban panel through its API with a
// reseller's login (Idea 1 source). It never writes to the panel.
package marzbanapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/source"
	"github.com/PdYrust/karvan/internal/secret"
)

// PageSize is how many users one list call asks for. Marzban has no cap.
const PageSize = 500

// Source is one logged-in reseller on one Marzban panel.
type Source struct {
	c        *source.Client
	username string
	password secret.Value

	mu    sync.Mutex
	token secret.Value
}

// New prepares a source; nothing is sent until the first call.
func New(c *source.Client, username string, password secret.Value) *Source {
	return &Source{c: c, username: username, password: password}
}

// Close drops the credentials and token. The source is unusable after.
func (s *Source) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.password, s.token = secret.Value{}, secret.Value{}
}

// Info reports the panel kind. Marzban's API has no version endpoint;
// the response shapes are checked on every read instead.
func (s *Source) Info(context.Context) (panel.Info, error) {
	return panel.Info{Kind: panel.Marzban, Capabilities: panel.Capabilities{PerUserFlow: true}}, nil
}

func (s *Source) login(ctx context.Context) error {
	if s.password.IsZero() {
		return errors.New("source is closed")
	}
	resp, err := s.c.PostForm(ctx, "/api/admin/token", url.Values{
		"username": {s.username}, "password": {s.password.Reveal()}, "grant_type": {"password"},
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

// get calls path with the bearer token, logging in first if needed and
// once more if the token expired.
func (s *Source) get(ctx context.Context, path string, q url.Values) (*source.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for attempt := 0; ; attempt++ {
		if s.token.IsZero() {
			if err := s.login(ctx); err != nil {
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
	return a.toAdmin()
}

// Users reads every user the login can see (a reseller sees their own),
// paged by username, until two passes agree.
func (s *Source) Users(ctx context.Context) ([]panel.User, error) {
	page := func(ctx context.Context, offset, limit int) ([]panel.User, int, error) {
		q := url.Values{"offset": {strconv.Itoa(offset)}, "limit": {strconv.Itoa(limit)}, "sort": {"username"}}
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
	return source.ReadAll(ctx, PageSize, page, func(u panel.User) string { return u.Username })
}

// Skew is the panel's clock minus Karvan's, from the last response.
func (s *Source) Skew() time.Duration { return s.c.Skew() }
