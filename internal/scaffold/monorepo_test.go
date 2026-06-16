package scaffold

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestParseExprsSpansAndNames(t *testing.T) {
	doc := []byte("# top\nmonorepo_root = true\n\n[tools]\nnode = \"24\"\npnpm = \"11\"\n\n[tasks.\"fmt:check\"]\nrun = \"oxfmt --check app\"\n")
	ex, err := parseExprs(doc)
	if err != nil {
		t.Fatal(err)
	}
	// The KeyValue monorepo_root must round-trip its exact source via its span.
	var got string
	for _, e := range ex {
		if e.name == "monorepo_root" {
			got = string(doc[e.span.start:e.span.end])
		}
	}
	if got != "monorepo_root = true" {
		t.Errorf("monorepo_root span = %q", got)
	}
	// The quoted table name must join to tasks.fmt:check.
	found := false
	for _, e := range ex {
		if e.name == "tasks.fmt:check" {
			found = true
		}
	}
	if !found {
		t.Errorf("did not find table tasks.fmt:check; got %v", names(ex))
	}
}

func TestParseExprsArrayTableHeaderSpan(t *testing.T) {
	// Guard the [[..]] header-span derivation: the span must cover both brackets.
	doc := []byte("[[products]]\nname = \"x\"\n")
	ex, err := parseExprs(doc)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ex {
		if e.name == "products" && e.kind.isTable() {
			if got := string(doc[e.span.start:e.span.end]); got != "[[products]]" {
				t.Errorf("array-table header span = %q, want [[products]]", got)
			}
			return
		}
	}
	t.Fatalf("array table not found; got %v", names(ex))
}

func TestSectionEndInsertsAfterLastKey(t *testing.T) {
	doc := []byte("[tools]\nnode = \"24\"\npnpm = \"11\"\n\n[env]\nx = 1\n")
	ex, _ := parseExprs(doc)
	for i, e := range ex {
		if e.name == "tools" && e.kind.isTable() {
			at := sectionEnd(ex, i)
			out := splice(doc, at, "\ngo = \"1.26\"")
			if !strings.Contains(string(out), "pnpm = \"11\"\ngo = \"1.26\"") {
				t.Errorf("go key not inserted after pnpm:\n%s", out)
			}
			return
		}
	}
	t.Fatal("tools table not found")
}

func TestSubstituteVars(t *testing.T) {
	got := substituteVars("swift test --package-path {{ vars.pp }} --x", map[string]string{"pp": "Core"})
	if got != "swift test --package-path Core --x" {
		t.Errorf("substituteVars = %q", got)
	}
	// no-space form too
	if substituteVars("a {{vars.pp}} b", map[string]string{"pp": "Z"}) != "a Z b" {
		t.Error("no-space vars form not substituted")
	}
}

// names + a helper used across tests.
func names(ex []expr) []string {
	var out []string
	for _, e := range ex {
		out = append(out, e.name)
	}
	return out
}

func TestLoadFamilyParsesToolsAndTemplates(t *testing.T) {
	// A tiny in-memory family file via fstest.
	fsys := fstest.MapFS{
		"templates/monorepo/node.toml": {Data: []byte(`[tools]
node = "24"
pnpm = "11"

[task_templates."node:test"]
description = "run Vitest"
run = "vitest run --reporter=junit --outputFile=junit.xml"

[task_templates."node:fmt:check"]
run = "oxfmt --check app"
`)},
	}
	fam, err := LoadFamily(fsys, "node")
	if err != nil {
		t.Fatal(err)
	}
	if fam.Name != "node" {
		t.Errorf("Name = %q", fam.Name)
	}
	if len(fam.Tools) != 2 || fam.Tools[0].Key != "node" || fam.Tools[0].Val != "24" {
		t.Errorf("Tools = %+v", fam.Tools)
	}
	if fam.Templates["test"].Run != "vitest run --reporter=junit --outputFile=junit.xml" {
		t.Errorf("test template = %+v", fam.Templates["test"])
	}
	if fam.Templates["fmt:check"].Run != "oxfmt --check app" {
		t.Errorf("fmt:check template = %+v", fam.Templates["fmt:check"])
	}
	// Raw must hold the verbatim [task_templates.*] blocks for EOF append.
	if !strings.Contains(fam.Raw, `[task_templates."node:test"]`) || strings.Contains(fam.Raw, "[tools]") {
		t.Errorf("Raw should hold only the task_templates blocks:\n%s", fam.Raw)
	}
}
