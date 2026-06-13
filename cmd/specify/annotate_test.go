package main

import (
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/engine"
	"github.com/markmals/speckit/internal/specmodel"
)

func TestVerifyAnnotations(t *testing.T) {
	locs := map[specmodel.SpecID]engine.SpecLocation{
		"scenario.x.a": {File: "features/x.md", Line: 9},
	}
	v := engine.VerifyResult{
		Unjoinable: []specmodel.SpecID{"scenario.x.a"},
		Dangling:   []engine.Binding{{Scenario: "scenario.x.b", File: "app/x.test.ts", Line: 3}},
	}
	got := strings.Join(verifyAnnotations(v, locs), "\n")
	for _, want := range []string{
		"::error file=features/x.md,line=9::scenario scenario.x.a is declared but has no bound test",
		"::error file=app/x.test.ts,line=3::test binds undeclared scenario scenario.x.b",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("verifyAnnotations missing %q\n%s", want, got)
		}
	}
}

func TestParityAnnotations(t *testing.T) {
	locs := map[specmodel.SpecID]engine.SpecLocation{"scenario.x.a": {File: "features/x.md", Line: 9}}
	r := engine.ParityReport{Cells: []engine.ParityCell{
		{Scenario: "scenario.x.a", State: "drifted"},
		{Scenario: "scenario.x.b", State: "conforming"}, // conforming cells are not annotated
	}}
	got := parityAnnotations(r, locs)
	if len(got) != 1 || got[0] != "::error file=features/x.md,line=9::scenario scenario.x.a is drifted" {
		t.Errorf("parityAnnotations = %v", got)
	}
}

func TestParseFormat(t *testing.T) {
	for _, c := range []struct {
		in   string
		want outputFormat
		ok   bool
	}{
		{"", formatText, true},
		{"text", formatText, true},
		{"json", formatJSON, true},
		{"github", formatGitHub, true},
		{"yaml", "", false},
	} {
		got, err := parseFormat(c.in)
		if c.ok && (err != nil || got != c.want) {
			t.Errorf("parseFormat(%q) = (%q, %v), want (%q, nil)", c.in, got, err, c.want)
		}
		if !c.ok && err == nil {
			t.Errorf("parseFormat(%q) = nil err, want error", c.in)
		}
	}
}

func TestGhCommand(t *testing.T) {
	for _, c := range []struct {
		level, file string
		line        int
		msg, want   string
	}{
		// file + line: the firewall annotation points at the exact test line
		{"error", "apps/web/app/lib/greeting.test.ts", 6, "test changed but its spec did not",
			"::error file=apps/web/app/lib/greeting.test.ts,line=6::test changed but its spec did not"},
		// file, no line (line ≤ 0 omitted)
		{"error", "specs/x.md", 0, "scenario x has no bound test",
			"::error file=specs/x.md::scenario x has no bound test"},
		// no file (e.g. scoped-commit) attaches to the run; line ignored
		{"error", "", 3, `undefined commit scope "frontend"`, `::error::undefined commit scope "frontend"`},
		// property values escape : and , ; message escapes % and newlines
		{"warning", "a,b:c.ts", 12, "x\ny%z", "::warning file=a%2Cb%3Ac.ts,line=12::x%0Ay%25z"},
	} {
		if got := ghCommand(c.level, c.file, c.line, c.msg); got != c.want {
			t.Errorf("ghCommand(%q,%q,%d,%q) = %q, want %q", c.level, c.file, c.line, c.msg, got, c.want)
		}
	}
}
