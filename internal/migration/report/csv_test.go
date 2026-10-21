package report

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/migration/run"
)

func TestCSVFiles(t *testing.T) {
	r := &run.Report{Users: []run.UserLine{
		{Key: "1", SourceUsername: "علی", Username: "ali_2", Moved: true, DestID: 9, ChangesLink: true,
			Fixed: []string{"C02:suffix"}, NewLink: "https://new.example/sub/abc"},
		{Key: "2", SourceUsername: "=cmd()", Username: "=cmd()", Moved: false},
	}}
	var users, links bytes.Buffer
	if err := WriteUsers(&users, r); err != nil {
		t.Fatal(err)
	}
	if err := WriteLinks(&links, r); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(users.String(), bom) {
		t.Fatal("no byte order mark: Excel would garble Persian")
	}
	rows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(users.String(), bom))).ReadAll()
	if err != nil || len(rows) != 3 || rows[1][1] != "علی" || rows[1][3] != "yes" || rows[2][1] != "'=cmd()" {
		t.Fatalf("users csv: %v %v", rows, err)
	}
	lrows, _ := csv.NewReader(strings.NewReader(strings.TrimPrefix(links.String(), bom))).ReadAll()
	if len(lrows) != 2 || lrows[1][2] != "https://new.example/sub/abc" {
		t.Fatalf("links csv: %v", lrows)
	}
}

func TestCheckCSV(t *testing.T) {
	r := &run.CheckReport{Users: []run.CheckLine{
		{Key: "1", SourceUsername: "علی", Username: "ali_2", Moving: true, Targets: []string{"3", "5"}, ChangesLink: true,
			Fixed: []string{"C02:suffix"}, Conflicts: []engine.Conflict{{Type: "C26", Severity: "info", Fix: "keep"}}, Applied: []string{"op:time.add"}},
		{Key: "2", SourceUsername: "+cmd", Username: "+cmd", Adopted: true, Conflicts: []engine.Conflict{{Type: "C10", Severity: "warning"}}},
	}}
	var out bytes.Buffer
	if err := WriteCheck(&out, r); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), bom) {
		t.Fatal("no byte order mark")
	}
	rows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(out.String(), bom))).ReadAll()
	want := [][]string{
		{"key", "source_username", "username", "moves", "adopted", "targets", "config_changed", "link_changed", "fixes", "conflicts", "operations"},
		{"1", "علی", "ali_2", "yes", "no", "3 5", "no", "yes", "C02:suffix", "C26:keep", "op:time.add"},
		{"2", "'+cmd", "'+cmd", "no", "yes", "", "no", "no", "", "C10", ""},
	}
	if err != nil || fmt.Sprint(rows) != fmt.Sprint(want) {
		t.Fatalf("check csv: %q %v", rows, err)
	}
}
