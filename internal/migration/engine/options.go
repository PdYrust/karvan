package engine

// Option is one way to fix a conflict, as the operator sees it.
type Option struct {
	ID            string   `json:"id"`
	Params        []string `json:"params,omitempty"`
	ChangesConfig bool     `json:"changesConfig"`
	ChangesLink   bool     `json:"changesLink"`
}

// Option ids.
const (
	OptPrefix        = "prefix"
	OptSuffix        = "suffix"
	OptRename        = "rename"
	OptSanitize      = "sanitize"
	OptSkip          = "skip"
	OptNewCredential = "new_credential" //nolint:gosec // an option id, not a credential
	OptDropProtocol  = "drop_protocol"
	OptChangeMethod  = "change_method"
	OptAccept        = "accept"
	OptAdopt         = "adopt"
	OptMoveAgain     = "move_again"
	OptNormalize     = "normalize"
	OptMerge         = "merge"
	OptKeep          = "keep"
	OptFit           = "fit"    // move values inside the role's limits
	OptRemove        = "remove" // drop settings the role does not allow
)

var (
	prefix   = Option{ID: OptPrefix, Params: []string{"text"}}
	suffix   = Option{ID: OptSuffix, Params: []string{"text"}}
	rename   = Option{ID: OptRename, Params: []string{"username"}}
	sanitize = Option{ID: OptSanitize}
	skip     = Option{ID: OptSkip}
	newCred  = Option{ID: OptNewCredential, ChangesConfig: true}
)

// rankContext is what ranking rules look at.
type rankContext struct {
	count map[string]int // conflicts per type in this build
}

// optionTable is the fixed rule table: each type's options in their
// default order. Reordering happens only in Options, by stated rules.
var optionTable = map[string][]Option{
	AdminTaken:     {rename, {ID: OptMerge}},
	NameTaken:      {suffix, prefix, rename, skip},
	CaseOnly:       {suffix, rename, skip},
	NameInvalid:    {sanitize, rename, skip},
	NameTwice:      {suffix, rename, skip},
	CredTwice:      {skip, newCred},
	CredTaken:      {skip, newCred},
	NoProtocol:     {skip, {ID: OptAccept, ChangesConfig: true}},
	WireGuardOnly:  {skip, {ID: OptAccept, ChangesConfig: true}},
	FlowDropped:    {{ID: OptAccept, ChangesConfig: true}, skip},
	ShadowsocksBad: {{ID: OptNewCredential, ChangesConfig: true}, {ID: OptChangeMethod, Params: []string{"method"}, ChangesConfig: true}, {ID: OptDropProtocol, ChangesConfig: true}, skip},
	AlreadyMoved:   {skip},
	MovedGone:      {skip, {ID: OptMoveAgain}},
	OnHoldInvalid:  {{ID: OptNormalize}},
	WireGuardIPs:   {{ID: OptAccept, ChangesConfig: true}, {ID: OptDropProtocol, ChangesConfig: true}, skip},
	SourceInvalid:  {skip, newCred},
	ResetEstimated: {{ID: OptKeep}},
	AlreadyHere:    {{ID: OptAdopt}, skip},
	RoleLimit:      {{ID: OptKeep}, {ID: OptFit}, skip},
	RoleFeature:    {{ID: OptKeep}, {ID: OptRemove}, skip},
}

// manyRenames is when "prefix all" beats "suffix each": with five or
// more taken names, one shared prefix reads better than numbered names.
const manyRenames = 5

// Options lists the fixes for t, most relevant first.
func Options(t string, rc rankContext) []Option {
	opts := append([]Option(nil), optionTable[t]...)
	if t == NameTaken && rc.count[NameTaken] >= manyRenames {
		opts = moveFirst(opts, OptPrefix)
	}
	return opts
}

func moveFirst(opts []Option, id string) []Option {
	for i, o := range opts {
		if o.ID == id {
			out := append([]Option{o}, opts[:i]...)
			return append(out, opts[i+1:]...)
		}
	}
	return opts
}

// defaultFix is the option applied when the operator chose nothing: the
// first option, but only if it changes no config and no user is lost
// silently. Blockers have none.
func defaultFix(t string, rc rankContext) (Option, bool) {
	switch t {
	case NameTaken, CaseOnly, NameInvalid, NameTwice, CredTwice, CredTaken,
		AlreadyMoved, MovedGone, OnHoldInvalid, SourceInvalid, ResetEstimated, AlreadyHere, RoleLimit, RoleFeature:
	default:
		return Option{}, false
	}
	o := Options(t, rc)[0]
	return o, !o.ChangesConfig
}
