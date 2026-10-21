// Package report writes a run report as the CSV files an operator
// downloads (D52). They open in Excel with Persian intact: UTF-8 with a
// byte order mark.
package report

import (
	"encoding/csv"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/PdYrust/karvan/internal/migration/run"
)

const bom = "\ufeff"

// WriteUsers writes one row per user: what happened to them and why.
func WriteUsers(w io.Writer, r *run.Report) error {
	if _, err := io.WriteString(w, bom); err != nil {
		return err
	}
	c := csv.NewWriter(w)
	if err := c.Write([]string{"key", "source_username", "username", "moved", "destination_id",
		"config_changed", "link_changed", "fixes", "operations"}); err != nil {
		return err
	}
	for _, u := range r.Users {
		id := ""
		if u.DestID > 0 {
			id = strconv.FormatInt(u.DestID, 10)
		}
		if err := c.Write([]string{safe(u.Key), safe(u.SourceUsername), safe(u.Username), yes(u.Moved), id,
			yes(u.ChangesConfig), yes(u.ChangesLink), strings.Join(u.Fixed, " "), strings.Join(u.Applied, " ")}); err != nil {
			return err
		}
	}
	c.Flush()
	return c.Error()
}

// WriteLinks writes each moved user's new subscription link, for the
// reseller to send (the fallback when an old link cannot be bridged).
func WriteLinks(w io.Writer, r *run.Report) error {
	if _, err := io.WriteString(w, bom); err != nil {
		return err
	}
	c := csv.NewWriter(w)
	if err := c.Write([]string{"username", "old_username", "new_link"}); err != nil {
		return err
	}
	for _, u := range r.Users {
		if !u.Moved || u.NewLink == "" {
			continue
		}
		if err := c.Write([]string{safe(u.Username), safe(u.SourceUsername), u.NewLink}); err != nil {
			return err
		}
	}
	c.Flush()
	return c.Error()
}

// WriteCheck writes the pre-flight report, one row per user: what the run
// would do with them (B7).
func WriteCheck(w io.Writer, r *run.CheckReport) error {
	if _, err := io.WriteString(w, bom); err != nil {
		return err
	}
	c := csv.NewWriter(w)
	if err := c.Write([]string{"key", "source_username", "username", "moves", "adopted", "targets",
		"config_changed", "link_changed", "fixes", "conflicts", "operations"}); err != nil {
		return err
	}
	for _, u := range r.Users {
		targets := make([]string, len(u.Targets))
		for i, t := range u.Targets {
			targets[i] = safe(t) // a Marzban inbound tag is the panel's own text
		}
		conflicts := make([]string, len(u.Conflicts))
		for i, k := range u.Conflicts {
			conflicts[i] = strings.TrimSuffix(k.Type+":"+k.Fix, ":")
		}
		if err := c.Write([]string{safe(u.Key), safe(u.SourceUsername), safe(u.Username), yes(u.Moving), yes(u.Adopted),
			strings.Join(targets, " "), yes(u.ChangesConfig), yes(u.ChangesLink), strings.Join(u.Fixed, " "),
			strings.Join(conflicts, " "), strings.Join(u.Applied, " ")}); err != nil {
			return err
		}
	}
	c.Flush()
	return c.Error()
}

func yes(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// safe stops a spreadsheet from running a cell as a formula: usernames
// come from the old panel and may start with =, +, - or @.
func safe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

// WritePasswords writes the generated admin passwords, sorted by admin.
func WritePasswords(w io.Writer, passwords map[string]string) error {
	if _, err := io.WriteString(w, bom); err != nil {
		return err
	}
	c := csv.NewWriter(w)
	if err := c.Write([]string{"admin", "password"}); err != nil {
		return err
	}
	names := make([]string, 0, len(passwords))
	for n := range passwords {
		names = append(names, n)
	}
	slices.Sort(names)
	for _, n := range names {
		// Generated passwords never start with a formula character
		// (pasarguard.GeneratePassword), so they go out exactly as set.
		if err := c.Write([]string{safe(n), passwords[n]}); err != nil {
			return err
		}
	}
	c.Flush()
	return c.Error()
}
