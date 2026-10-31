package handlers

import (
	"net/http"
	"slices"
	"strconv"

	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/netports"
)

// Who holds a listening port.
const (
	ownerPanel = "panel"
	ownerRule  = "rule"
	ownerACME  = "acme"
	ownerOther = "other"
)

type portJSON struct {
	Port  int    `json:"port"`
	Owner string `json:"owner"`
}

type portsJSON struct {
	Ports []portJSON `json:"ports"`
	// Proposal is the port netports.Propose picks for a new rule; 0 when none is free.
	Proposal int `json:"proposal"`
}

// Ports lists the TCP ports this host listens on, who holds each, and the
// port a new rule should take. ?prefer= is the old domain's own port.
func (a *API) Ports(w http.ResponseWriter, r *http.Request) {
	prefer := 0
	if s := r.URL.Query().Get("prefer"); s != "" {
		p, err := strconv.Atoi(s)
		if err != nil || p < 1 || p > 65535 {
			respond.BadRequest(w)
			return
		}
		prefer = p
	}
	listening, err := netports.Listening()
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	owners, err := a.portOwners(r)
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	out := portsJSON{Ports: make([]portJSON, 0, len(listening))}
	for _, p := range listening {
		owner, ok := owners[p]
		if !ok {
			owner = ownerOther
		}
		out.Ports = append(out.Ports, portJSON{Port: p, Owner: owner})
	}
	out.Proposal = netports.Propose(func(p int) bool {
		return owners[p] == ownerRule || (owners[p] != ownerPanel && !slices.Contains(listening, p))
	}, prefer)
	respond.JSON(w, http.StatusOK, out)
}

// portOwners maps the ports Karvan holds to who holds them. A rule wins
// over the panel: a shared panel address is on a rule's port, which more
// rules can use.
func (a *API) portOwners(r *http.Request) (map[int]string, error) {
	owners := map[int]string{}
	if a.Bridge != nil && a.Bridge.ChallengeHeld() {
		owners[a.Bridge.ChallengePort] = ownerACME
	}
	if a.Address != nil {
		st := a.Address.Status()
		owners[st.Current.Port] = ownerPanel
		if st.Pending != nil {
			owners[st.Pending.Port] = ownerPanel
		}
	}
	if a.Bridge == nil {
		return owners, nil
	}
	rules, err := a.Bridge.RulePorts(r.Context())
	if err != nil {
		return nil, err
	}
	for _, rp := range rules {
		owners[rp.Port] = ownerRule
	}
	return owners, nil
}
