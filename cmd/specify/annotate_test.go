package main

import "testing"

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
	for _, c := range []struct{ level, file, msg, want string }{
		// the firewall demo: a file-scoped error annotation on the test
		{"error", "apps/web/app/lib/greeting.test.ts", "test changed but its spec did not",
			"::error file=apps/web/app/lib/greeting.test.ts::test changed but its spec did not"},
		// no file (e.g. scoped-commit) attaches to the run
		{"error", "", `undefined commit scope "frontend"`, `::error::undefined commit scope "frontend"`},
		// property values escape : and , ; message escapes % and newlines
		{"warning", "a,b:c.ts", "x\ny%z", "::warning file=a%2Cb%3Ac.ts::x%0Ay%25z"},
	} {
		if got := ghCommand(c.level, c.file, c.msg); got != c.want {
			t.Errorf("ghCommand(%q,%q,%q) = %q, want %q", c.level, c.file, c.msg, got, c.want)
		}
	}
}
