package source

import (
	"bytes"
	"encoding/json"
	"time"
)

// DecodeJSON decodes a panel response into v. Unknown fields are allowed
// (panels add fields between versions); wrong types are not.
func DecodeJSON(body []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(body))
	if err := d.Decode(v); err != nil {
		return BadResponse("decode: %v", err)
	}
	return nil
}

var apiLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05.999999999",
}

// ParseTime reads a datetime as both panels serialize it: ISO 8601, with an
// offset (PasarGuard) or naive UTC (Marzban).
func ParseTime(s string) (time.Time, error) {
	for _, l := range apiLayouts {
		if t, err := time.ParseInLocation(l, s, time.UTC); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, BadResponse("datetime %q", s)
}

// ParseTimePtr is ParseTime for an optional field.
func ParseTimePtr(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := ParseTime(*s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
