package paneldb

import (
	"fmt"
	"time"
)

// SQLAlchemy stores SQLite datetimes as text in this layout, naive UTC.
const sqliteLayout = "2006-01-02 15:04:05.000000"

var readLayouts = []string{
	sqliteLayout,
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05",
}

// NullTime scans any datetime both panels store, on any backend: time.Time
// from MySQL and PostgreSQL, text from SQLite. Naive values are UTC, as both
// panels write them.
type NullTime struct {
	Time  time.Time
	Valid bool
}

// Scan implements sql.Scanner.
func (n *NullTime) Scan(v any) error {
	n.Valid = false
	switch x := v.(type) {
	case nil:
		return nil
	case time.Time:
		n.Time, n.Valid = x.UTC(), true
		return nil
	case []byte:
		return n.parse(string(x))
	case string:
		return n.parse(x)
	}
	return fmt.Errorf("cannot scan %T as a datetime", v)
}

func (n *NullTime) parse(s string) error {
	for _, l := range readLayouts {
		if t, err := time.ParseInLocation(l, s, time.UTC); err == nil {
			n.Time, n.Valid = t.UTC(), true
			return nil
		}
	}
	return fmt.Errorf("cannot parse datetime %q", s)
}

// Ptr returns the time, or nil when NULL.
func (n NullTime) Ptr() *time.Time {
	if !n.Valid {
		return nil
	}
	t := n.Time
	return &t
}

// TimeArg is the value to bind for t in d: SQLAlchemy's text layout on
// SQLite, a UTC time.Time elsewhere. Every datetime column both panels
// create on MySQL and MariaDB is DATETIME(0), which rounds fractions (and
// can round up a second), so they are truncated here; the rest keep
// microseconds like the panels.
func TimeArg(d Dialect, t time.Time) any {
	t = t.UTC().Truncate(time.Microsecond)
	switch d {
	case SQLite:
		return t.Format(sqliteLayout)
	case MySQL:
		return t.Truncate(time.Second)
	}
	return t
}

// Stored is t as the database keeps it after TimeArg, for comparing a row
// read back with what was written.
func Stored(d Dialect, t time.Time) time.Time {
	t = t.UTC().Truncate(time.Microsecond)
	if d == MySQL {
		return t.Truncate(time.Second)
	}
	return t
}

// NullTimeArg is TimeArg for an optional value.
func NullTimeArg(d Dialect, t *time.Time) any {
	if t == nil {
		return nil
	}
	return TimeArg(d, *t)
}
