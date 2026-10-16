package paneldb

import (
	"testing"
	"time"
)

func TestNullTimeScansEveryStoredForm(t *testing.T) {
	want := time.Date(2026, 10, 16, 8, 30, 15, 123456000, time.UTC)
	tehran := time.FixedZone("Tehran", 3*3600+1800)
	for _, v := range []any{
		"2026-10-16 08:30:15.123456",
		[]byte("2026-10-16 08:30:15.123456"),
		"2026-10-16 08:30:15.123456+00:00",
		"2026-10-16T08:30:15.123456Z",
		want.In(tehran),
	} {
		var n NullTime
		if err := n.Scan(v); err != nil || !n.Valid || !n.Time.Equal(want) || n.Time.Location() != time.UTC {
			t.Errorf("Scan(%v) = %v %v, %v", v, n.Time, n.Valid, err)
		}
	}
	var n NullTime
	if err := n.Scan(nil); err != nil || n.Valid || n.Ptr() != nil {
		t.Error("NULL must scan as invalid")
	}
	if err := n.Scan("yesterday"); err == nil {
		t.Error("garbage parsed")
	}
}

func TestTimeArgIsUTCWithMicroseconds(t *testing.T) {
	tehran := time.FixedZone("Tehran", 3*3600+1800)
	in := time.Date(2026, 10, 16, 12, 0, 15, 123456789, tehran)
	if got := TimeArg(SQLite, in); got != "2026-10-16 08:30:15.123456" {
		t.Errorf("sqlite arg = %v", got)
	}
	got, ok := TimeArg(MySQL, in).(time.Time)
	if !ok || got.Location() != time.UTC || got.Nanosecond() != 0 || got.Second() != 15 {
		t.Errorf("mysql arg = %v; DATETIME(0) needs truncated seconds", got)
	}
	if pg, _ := TimeArg(Postgres, in).(time.Time); pg.Nanosecond() != 123456000 {
		t.Errorf("postgres arg = %v", pg)
	}
	if NullTimeArg(Postgres, nil) != nil {
		t.Error("nil must bind as NULL")
	}
}
