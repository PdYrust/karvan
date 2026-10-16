package panel

import (
	"log/slog"
	"strings"
	"testing"
)

func TestCanonicalUUID(t *testing.T) {
	a := CanonicalUUID(" 3F2504E0-4F89-11D3-9A0C-0305E82C3301 ")
	b := CanonicalUUID("3f2504e0-4f89-11d3-9a0c-0305e82c3301")
	if a != b {
		t.Fatalf("%q != %q", a, b)
	}
}

func TestUUIDIsRedactedInLogs(t *testing.T) {
	var b strings.Builder
	slog.New(slog.NewTextHandler(&b, nil)).Info("x", "uuid", CanonicalUUID("3f2504e0-4f89-11d3-9a0c-0305e82c3301"))
	if strings.Contains(b.String(), "3f2504e0") {
		t.Fatalf("log leaks the UUID: %s", b.String())
	}
}
