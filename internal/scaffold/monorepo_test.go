package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	toml "github.com/pelletier/go-toml/v2"
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

// valid reports whether s re-parses as TOML — a sanity check that the surgical
// splices did not corrupt the document.
func valid(t *testing.T, s string) bool {
	t.Helper()
	var v map[string]any
	return toml.Unmarshal([]byte(s), &v) == nil
}

func nodeFam(t *testing.T, hoist bool) Family {
	t.Helper()
	fsys := fstest.MapFS{"templates/monorepo/node.toml": {Data: []byte(`[tools]
node = "24"
pnpm = "11"

[task_templates."node:test"]
run = "vitest run --reporter=junit --outputFile=junit.xml"
`)}}
	fam, err := LoadFamily(fsys, "node")
	if err != nil {
		t.Fatal(err)
	}
	fam.Hoist = hoist
	return fam
}

func TestEnsureRootMiseCreatesSkeleton(t *testing.T) {
	root := t.TempDir()
	changed, err := EnsureRootMise(root, []Family{nodeFam(t, false)}, []string{"apps/web"})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("creating the root config should report changed")
	}
	got := read(t, filepath.Join(root, "mise.toml"))
	for _, want := range []string{"monorepo_root = true", "[monorepo]", `config_roots = ["apps/*"]`, "[tools]", `node = "24"`, `pnpm = "11"`} {
		if !strings.Contains(got, want) {
			t.Errorf("root mise.toml missing %q:\n%s", want, got)
		}
	}
	// One member: no node templates yet.
	if strings.Contains(got, "[task_templates") {
		t.Errorf("single-member root must not hoist templates:\n%s", got)
	}
	// Re-parse sanity.
	if !valid(t, got) {
		t.Error("root mise.toml does not re-parse")
	}
}

func TestEnsureRootMiseIsIdempotent(t *testing.T) {
	root := t.TempDir()
	if _, err := EnsureRootMise(root, []Family{nodeFam(t, false)}, []string{"apps/web"}); err != nil {
		t.Fatal(err)
	}
	first := read(t, filepath.Join(root, "mise.toml"))
	changed, err := EnsureRootMise(root, []Family{nodeFam(t, false)}, []string{"apps/web"})
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("second identical run should be a no-op (changed=false)")
	}
	if read(t, filepath.Join(root, "mise.toml")) != first {
		t.Error("second run mutated the file")
	}
}

func TestEnsureRootMiseMergesGlobToolsAndTemplatesPreservingComments(t *testing.T) {
	root := t.TempDir()
	// A user-authored root with their own comment, a hand-pinned tool version, and
	// a hand-added [tasks.*] table — all must round-trip untouched.
	seed := `# my notes — keep me
monorepo_root = true

[monorepo]
config_roots = ["apps/*"]

[tools]
node = "22"   # I pinned this on purpose

# my own root task — keep me too
[tasks.deploy]
run = "echo ship it"
`
	if err := os.WriteFile(filepath.Join(root, "mise.toml"), []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	// Add a second member dir (cmd/api) and hoist node templates (2 members).
	changed, err := EnsureRootMise(root, []Family{nodeFam(t, true)}, []string{"apps/web", "cmd/api"})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("merging new glob/templates should report changed")
	}
	got := read(t, filepath.Join(root, "mise.toml"))
	for _, want := range []string{
		"my notes — keep me",       // user comment preserved
		`node = "22"`,              // user pin NOT overwritten
		"I pinned this on purpose", // inline comment preserved
		`pnpm = "11"`,              // missing family tool added
		`"apps/*"`, `"cmd/*"`,      // both globs
		`[task_templates."node:test"]`, // templates hoisted (Hoist=true)
		"[tasks.deploy]",               // user's hand-added task preserved
		`run = "echo ship it"`,         // …with its body intact
		"keep me too",                  // …and its comment
	} {
		if !strings.Contains(got, want) {
			t.Errorf("merge missing/regressed %q:\n%s", want, got)
		}
	}
	if !valid(t, got) {
		t.Errorf("merged root mise.toml does not re-parse:\n%s", got)
	}
}

func TestPromoteMemberConvertsCanonicalTasks(t *testing.T) {
	dir := t.TempDir()
	member := filepath.Join(dir, "mise.toml")
	// An inline member whose test body is canonical but whose typecheck was edited.
	src := `[env]
_.path = ['{{config_root}}/node_modules/.bin']

[tasks.test]
description = "run Vitest with the junit reporter the engine joins"
run = "vitest run --reporter=junit --outputFile=junit.xml"

[tasks.typecheck]
depends = ["routes"]
run = "tsgo --noEmit --strict"   # user added --strict
`
	if err := os.WriteFile(member, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	fam := Family{Name: "node", Templates: map[string]Template{
		"test":      {Run: "vitest run --reporter=junit --outputFile=junit.xml"},
		"typecheck": {Run: "tsgo --noEmit"},
	}}
	changed, err := PromoteMember(member, fam)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("a canonical task should have been converted")
	}
	got := read(t, member)
	// test was canonical -> extends; its run line is gone.
	if !strings.Contains(got, `extends = "node:test"`) || strings.Contains(got, "vitest run --reporter") {
		t.Errorf("test not converted to extends:\n%s", got)
	}
	// typecheck was user-edited -> left inline; depends preserved.
	if strings.Contains(got, `extends = "node:typecheck"`) {
		t.Errorf("edited typecheck must NOT be converted:\n%s", got)
	}
	if !strings.Contains(got, "--strict") || !strings.Contains(got, `depends = ["routes"]`) {
		t.Errorf("user edit / depends not preserved:\n%s", got)
	}
	if !valid(t, got) {
		t.Errorf("promoted member does not re-parse:\n%s", got)
	}
}

func TestPromoteMemberIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	member := filepath.Join(dir, "mise.toml")
	src := "[tasks.test]\nrun = \"vitest run\"\n"
	os.WriteFile(member, []byte(src), 0o644)
	fam := Family{Name: "node", Templates: map[string]Template{"test": {Run: "vitest run"}}}
	if _, err := PromoteMember(member, fam); err != nil {
		t.Fatal(err)
	}
	after := read(t, member)
	changed, err := PromoteMember(member, fam)
	if err != nil {
		t.Fatal(err)
	}
	if changed || read(t, member) != after {
		t.Error("second promote should be a no-op")
	}
}

func TestPromoteMemberSubstitutesVars(t *testing.T) {
	dir := t.TempDir()
	member := filepath.Join(dir, "mise.toml")
	// Member's [vars] feed the family template's {{ vars.pp }}.
	src := `[vars]
pp = "Core"

[tasks.test]
run = "swift test --package-path Core --x"
`
	os.WriteFile(member, []byte(src), 0o644)
	fam := Family{Name: "swift", Templates: map[string]Template{
		"test": {Run: "swift test --package-path {{ vars.pp }} --x"},
	}}
	changed, err := PromoteMember(member, fam)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || !strings.Contains(read(t, member), `extends = "swift:test"`) {
		t.Errorf("vars-parameterized canonical task not converted:\n%s", read(t, member))
	}
}
