package pasarguardapi

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/source"
)

// ErrGroupsUnknown means the login cannot read the panel's groups or
// inbounds: a 403 (no groups.read or system.read), or a panel without
// the routes.
var ErrGroupsUnknown = errors.New("the old panel's groups cannot be read")

// groupsJSON is GroupsResponse (app/models/group.py:39).
type groupsJSON struct {
	Groups []struct {
		ID          *int64   `json:"id"`
		InboundTags []string `json:"inbound_tags"`
		IsDisabled  bool     `json:"is_disabled"`
	} `json:"groups"`
	Total *int `json:"total"`
}

// inboundJSON is the part of InboundSummary (app/models/system.py:4)
// Karvan keeps. The answer also holds WireGuard interface keys, which
// are never decoded.
type inboundJSON struct {
	Tag      string `json:"tag"`
	Protocol string `json:"protocol"`
}

// Groups reads every group with the protocols of its inbounds, from GET
// /api/groups (no offset or limit: the whole list) and GET
// /api/inbounds/details, so a user's current protocols can be told (D36).
func (s *Source) Groups(ctx context.Context) ([]panel.SourceGroup, error) {
	var gs groupsJSON
	if err := s.getJSON(ctx, "/api/groups", &gs); err != nil {
		return nil, err
	}
	if gs.Total == nil || *gs.Total != len(gs.Groups) {
		return nil, source.BadResponse("groups: total does not match the list")
	}
	var ins []inboundJSON
	if err := s.getJSON(ctx, "/api/inbounds/details", &ins); err != nil {
		return nil, err
	}
	protocol := make(map[string]string, len(ins))
	for _, in := range ins {
		protocol[in.Tag] = in.Protocol
	}
	out := make([]panel.SourceGroup, 0, len(gs.Groups))
	for _, g := range gs.Groups {
		if g.ID == nil {
			return nil, source.BadResponse("groups: no id")
		}
		sg := panel.SourceGroup{ID: *g.ID, Disabled: g.IsDisabled}
		for _, tag := range g.InboundTags {
			if p := protocol[tag]; p != "" && !slices.Contains(sg.Protocols, p) {
				sg.Protocols = append(sg.Protocols, p)
			}
		}
		out = append(out, sg)
	}
	return out, nil
}

func (s *Source) getJSON(ctx context.Context, path string, v any) error {
	resp, err := s.get(ctx, path, nil)
	if err != nil {
		if c := source.CodeOf(err); c == source.CodeForbidden || c == source.CodeNotFound {
			return fmt.Errorf("%w: %s: %w", ErrGroupsUnknown, path, err)
		}
		return err
	}
	return source.DecodeJSON(resp.Body, v)
}
