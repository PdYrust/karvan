package engine

import "testing"

func TestRuleTableRanking(t *testing.T) {
	few := rankContext{count: map[string]int{NameTaken: 4}}
	many := rankContext{count: map[string]int{NameTaken: 5}}
	if Options(NameTaken, few)[0].ID != OptSuffix {
		t.Error("a few taken names: numbered names first")
	}
	if Options(NameTaken, many)[0].ID != OptPrefix {
		t.Error("five or more taken names: one shared prefix first")
	}
	if len(Options(NameTaken, many)) != len(optionTable[NameTaken]) {
		t.Error("ranking must reorder, never drop")
	}
}

func TestDefaultsNeverChangeAConfig(t *testing.T) {
	rc := rankContext{count: map[string]int{}}
	for typ := range optionTable {
		if o, ok := defaultFix(typ, rc); ok && (o.ChangesConfig || o.ID == "") {
			t.Errorf("%s defaults to %q, which changes a config", typ, o.ID)
		}
	}
	for _, typ := range []string{FlowDropped, ShadowsocksBad, WireGuardIPs, AdminTaken} {
		if _, ok := defaultFix(typ, rc); ok {
			t.Errorf("%s must wait for the operator", typ)
		}
	}
	for typ, want := range map[string]string{CredTaken: OptSkip, CredTwice: OptSkip, AlreadyHere: OptAdopt, NameInvalid: OptSanitize, MovedGone: OptSkip} {
		if o, _ := defaultFix(typ, rc); o.ID != want {
			t.Errorf("%s default = %q, want %q", typ, o.ID, want)
		}
	}
}

func TestEveryConfigChangeIsFlagged(t *testing.T) {
	for typ, opts := range optionTable {
		for _, o := range opts {
			switch o.ID {
			case OptNewCredential, OptDropProtocol, OptChangeMethod:
				if !o.ChangesConfig {
					t.Errorf("%s/%s changes a config but is not flagged", typ, o.ID)
				}
			case OptAccept:
				if (typ == FlowDropped || typ == WireGuardIPs) && !o.ChangesConfig {
					t.Errorf("%s/accept changes a config but is not flagged", typ)
				}
			}
		}
	}
}
