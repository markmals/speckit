# Mise monorepo adoption — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** From the first `specify target add`, a SpecKit repo is a real [mise monorepo](https://mise.jdx.dev/tasks/monorepo.html) — a generated, comment-preserving root `mise.toml` that declares `monorepo_root`, `config_roots`, and the union of present families' toolchains; member task bodies stay inline until a family gains a second member, then hoist behind per-family `[task_templates]`.

**Architecture:** A new `internal/scaffold/monorepo.go` owns a surgical, comment-preserving TOML merge engine (built on `go-toml/v2`'s `unstable` byte-range parser) plus a `Family` model loaded from new `internal/coreassets/templates/monorepo/<family>.toml` contribution files. `specify target add` / `target register` call one `wireMonorepo` orchestrator after recording the target: it merges the root config and, for any family that has reached two members, writes that family's templates and converts each member's still-canonical inline tasks to `extends`. Each scaffold's `target.command` changes from `cd {{.Dir}} && mise run test` to the native `mise //{{.Dir}}:test`.

**Tech Stack:** Go 1.26, `github.com/pelletier/go-toml/v2` v2.3.1 (new dependency; the `unstable` package for byte-range splicing), Cobra CLI, mise 2026.6.6, `text/template` scaffolding.

---

## Background the implementer needs

Read these before starting; the plan assumes you know them.

- **Spec:** `docs/design/mise-monorepo.md` is the source of truth for *why*. This plan is the *how*. If they ever conflict, stop and flag it.
- **How a member is placed today:** `specify target add <name> --stack <s>` (in `cmd/specify/main.go`, `targetAddCmd`) renders a stack's `files/` tree into `<memberDir>/<name>` via `scaffold.Render`, seeds an example feature, drops `.github/`, then records the target in `.speckit/specs.json` via `config.AddTarget`. `targetRegisterCmd` records an *existing* member with no render. `init` places no members.
- **`memberDir` per stack** (from each `scaffold.json`): `web` → `apps` (default), `apple` → `apps`, `swift-package`/`swift-cli` → `packages`, `go-service` → `cmd` (and `sharedModule: true`).
- **The families that exist today** (the spec lists `node-cli` and `go-tui` as *future* — do not invent them):
  | Family | Stacks present today |
  | --- | --- |
  | `node` | `web` |
  | `swift` | `apple`, `swift-package`, `swift-cli` |
  | `go` | `go-service` |
  So a `node`-family second member is reachable only by a **second `web`** target today; a `swift` second member by e.g. `apple` + `swift-package`; a `go` second member by two `go-service` targets.
- **There is no golden-file harness.** The scaffold tests (`internal/scaffold/*_test.go`) are assertion-based — they render the real embedded scaffold and assert specific strings. "Golden regen" in the spec means *update those assertions and add new ones*, then run the CI gate. There is no `-update` flag to run.
- **The CI gate:** `mise run ci` (runs `fmt:check`, `go build ./...`, `go vet ./...`, `go test ./...`). Run it before every commit that touches Go. Per the project's `run-ci-before-push` memory, it is mandatory before pushing.
- **The `unstable` parser, verified at v2.3.1** (this plan's splice code was prototyped end-to-end against it):
  - `p := &unstable.Parser{KeepComments: true}`, `p.Reset(data)`, loop `for p.NextExpression() { e := p.Expression(); … }`.
  - Top-level expressions arrive **flat**: a `Table` node, then each of its keys as its own `KeyValue` node, then the next `Table`, interleaved with `Comment` nodes.
  - `e.Kind` ∈ {`unstable.Table`, `unstable.ArrayTable`, `unstable.KeyValue`, `unstable.Comment`, …}.
  - `e.Raw` is a `unstable.Range{Offset, Length uint32}` into the source. **A `Table` node's `Raw` is zero** — derive the header span from its `Key` child. A `KeyValue`'s `Raw` covers the full `key = value` (including multi-line `'''…'''`).
  - `e.Key()` returns an `Iterator`; loop `for it.Next() { it.Node() }` to read dotted/quoted key parts (`[task_templates."node:test"]` → parts `["task_templates", "node:test"]`).
  - `e.Value()` returns the value `*Node`; for a string, `string(e.Value().Data)` is the **decoded** value (no quotes).

## File structure

| Path | New/Modified | Responsibility |
| --- | --- | --- |
| `go.mod` / `go.sum` | Modified | Add `github.com/pelletier/go-toml/v2 v2.3.1`. |
| `internal/scaffold/monorepo.go` | **New** | The merge engine: `Family`, `LoadFamily`, `EnsureRootMise`, `PromoteMember`, and the splice primitives (`parseExprs`, `sectionEnd`, `splice`, `substituteVars`). |
| `internal/scaffold/monorepo_test.go` | **New** | Unit tests for the engine: create/merge/idempotency/comment-preservation, glob derivation, promotion (stays-inline / converts / edited-left-inline). |
| `internal/scaffold/scaffold.go` | Modified | Add `Family string` to `Manifest` (the `family` field). |
| `internal/coreassets/templates/monorepo/node.toml` | **New** (Stage 1) | The `node` family contribution: `[tools]` + `[task_templates."node:*"]`. |
| `internal/coreassets/templates/monorepo/swift.toml` | **New** (Stage 2) | The `swift` family contribution (no `[tools]`; vars-parameterized + superset-walk templates). |
| `internal/coreassets/templates/monorepo/go.toml` | **New** (Stage 3) | The `go` family contribution. |
| `internal/coreassets/templates/scaffolds/web/scaffold.json` | Modified (S1) | Add `"family": "node"`; change `target.command` to `mise //{{.Dir}}:test`. |
| `internal/coreassets/templates/scaffolds/web/files/mise.toml` | Modified (S1) | Drop the `[tools]` block (hoisted to root). |
| `internal/coreassets/templates/scaffolds/{apple,swift-package,swift-cli}/scaffold.json` | Modified (S2) | Add `"family": "swift"`; change `target.command`. |
| `internal/coreassets/templates/scaffolds/{apple,swift-package,swift-cli}/files/mise.toml.tmpl` | Modified (S2) | Add `[vars]`; align `test`/`fmt`/`lint` bodies to the shared canonical; keep stack-specific tasks inline. |
| `internal/coreassets/templates/scaffolds/go-service/scaffold.json` | Modified (S3) | Add `"family": "go"`; change `target.command`. |
| `internal/coreassets/templates/scaffolds/go-service/files/mise.toml.tmpl` | Modified (S3) | Drop the `[tools]` block. |
| `cmd/specify/monorepo.go` | **New** (S1) | CLI orchestration: `wireMonorepo`, `familyForStack`, family-membership counting. |
| `cmd/specify/main.go` | Modified (S1) | Call `wireMonorepo` after `config.AddTarget` in `targetAddCmd` and `registerTarget`. |
| `internal/engine/verify_run.go` | Modified (S1) | Refresh the `cfg.Command` comment to mention the `mise //dir:task` form. |
| `docs/design/scaffolds/web.md`, `node-cli.md` | Modified (S4) | Bring mise-monorepo docs to the current API. |
| `.claude/memory/mise-monorepo.md` + `MEMORY.md` | **New** (S4) | Project-memory topic for the invariant + the `unstable`-parser gotcha. |

---

# Stage 1 — Mechanism + the `node` family

This stage builds and de-risks the entire engine, then wires it for the one stack in the `node` family (`web`). It exercises both the one-member (inline) path and the two-member (promotion) path using two `web` targets.

## Task 1.1: Add the `go-toml/v2` dependency

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add the dependency**

Run:
```bash
cd /Users/orion/Developer/Templates/speckit
go get github.com/pelletier/go-toml/v2@v2.3.1
```
Expected: `go.mod` gains `require github.com/pelletier/go-toml/v2 v2.3.1`.

- [ ] **Step 2: Verify it builds**

Run: `go build ./...`
Expected: success (no use yet, just resolves the module).

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add pelletier/go-toml/v2 for comment-preserving TOML merge"
```

## Task 1.2: The splice primitives (`parseExprs`, `sectionEnd`, `splice`, `substituteVars`)

These are the proven building blocks. Write them first with focused tests.

**Files:**
- Create: `internal/scaffold/monorepo.go`
- Create: `internal/scaffold/monorepo_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/scaffold/monorepo_test.go`:

```go
package scaffold

import (
	"strings"
	"testing"
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
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/scaffold/ -run 'TestParseExprs|TestSectionEnd|TestSubstituteVars' -v`
Expected: FAIL — `undefined: parseExprs` / `splice` / `substituteVars`.

- [ ] **Step 3: Implement the primitives**

Create `internal/scaffold/monorepo.go`:

```go
package scaffold

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2/unstable"
)

// span is a half-open byte range [start,end) into a TOML document.
type span struct{ start, end int }

// exprKind classifies a parsed top-level expression. We only branch on a few
// kinds, so we wrap unstable.Kind with the predicates we need.
type exprKind unstable.Kind

func (k exprKind) isTable() bool {
	return unstable.Kind(k) == unstable.Table || unstable.Kind(k) == unstable.ArrayTable
}
func (k exprKind) isKeyValue() bool { return unstable.Kind(k) == unstable.KeyValue }

// expr is one top-level parsed expression with a real source span. For a table,
// name is its dotted key joined by "." (e.g. tasks.fmt:check); span is the
// derived [..] header line. For a key/value, name is the key, span is the whole
// "key = value", and val is the decoded string value (empty for non-strings).
type expr struct {
	kind exprKind
	span span
	name string
	val  string
}

// parseExprs returns every top-level expression with its source span. Tables get
// a derived header span (the unstable parser gives table nodes a zero Raw).
func parseExprs(data []byte) ([]expr, error) {
	p := &unstable.Parser{KeepComments: true}
	p.Reset(data)
	var out []expr
	for p.NextExpression() {
		e := p.Expression()
		switch e.Kind {
		case unstable.Table, unstable.ArrayTable:
			keyOff, keyEnd := -1, -1
			var parts []string
			it := e.Key()
			for it.Next() {
				k := it.Node()
				parts = append(parts, string(k.Data))
				if keyOff == -1 {
					keyOff = int(k.Raw.Offset)
				}
				keyEnd = int(k.Raw.Offset + k.Raw.Length)
			}
			hs := bytes.LastIndexByte(data[:keyOff], '[')
			he := len(data)
			if nl := bytes.IndexByte(data[keyEnd:], '\n'); nl >= 0 {
				he = keyEnd + nl
			}
			out = append(out, expr{exprKind(e.Kind), span{hs, he}, strings.Join(parts, "."), ""})
		case unstable.KeyValue:
			var parts []string
			it := e.Key()
			for it.Next() {
				parts = append(parts, string(it.Node().Data))
			}
			val := ""
			if v := e.Value(); v != nil {
				val = string(v.Data)
			}
			out = append(out, expr{exprKind(e.Kind), span{int(e.Raw.Offset), int(e.Raw.Offset + e.Raw.Length)}, strings.Join(parts, "."), val})
		default:
			out = append(out, expr{exprKind(e.Kind), span{int(e.Raw.Offset), int(e.Raw.Offset + e.Raw.Length)}, "", ""})
		}
	}
	return out, p.Error()
}

// sectionEnd returns the byte offset just after the last key/value belonging to
// the table whose header is exprs[i] — i.e. the insertion point for a new key,
// before the next table header or EOF.
func sectionEnd(exprs []expr, i int) int {
	end := exprs[i].span.end
	for j := i + 1; j < len(exprs); j++ {
		if exprs[j].kind.isTable() {
			break
		}
		if exprs[j].kind.isKeyValue() && exprs[j].span.end > end {
			end = exprs[j].span.end
		}
	}
	return end
}

// splice returns data with ins inserted at byte offset at.
func splice(data []byte, at int, ins string) []byte {
	out := make([]byte, 0, len(data)+len(ins))
	out = append(out, data[:at]...)
	out = append(out, ins...)
	out = append(out, data[at:]...)
	return out
}

var varsRe = regexp.MustCompile(`\{\{\s*vars\.(\w+)\s*\}\}`)

// substituteVars resolves mise's {{ vars.X }} interpolations against vars — used
// to compute a family template's canonical run for a given member before the
// promotion equality check.
func substituteVars(s string, vars map[string]string) string {
	return varsRe.ReplaceAllStringFunc(s, func(m string) string {
		name := varsRe.FindStringSubmatch(m)[1]
		if v, ok := vars[name]; ok {
			return v
		}
		return m
	})
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/scaffold/ -run 'TestParseExprs|TestSectionEnd|TestSubstituteVars' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/scaffold/monorepo.go internal/scaffold/monorepo_test.go
git commit -m "scaffold: surgical TOML splice primitives (parseExprs/sectionEnd/splice)"
```

## Task 1.3: The `Family` model + `family` manifest field + `LoadFamily`

**Files:**
- Modify: `internal/scaffold/scaffold.go:21-45` (add `Family` to `Manifest`)
- Modify: `internal/scaffold/monorepo.go` (add `Family`, `LoadFamily`)
- Modify: `internal/scaffold/monorepo_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/scaffold/monorepo_test.go`:

```go
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
```

Add imports `"io/fs"`, `"testing/fstest"` to the test file.

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/scaffold/ -run TestLoadFamily -v`
Expected: FAIL — `undefined: LoadFamily`.

- [ ] **Step 3: Add `Family` to the manifest**

In `internal/scaffold/scaffold.go`, inside the `Manifest` struct (after the `NameRule` field, around line 31), add:

```go
	// Family groups stacks that share a mise monorepo contribution — toolchain
	// pins (hoisted to the root mise.toml) and [task_templates] (hoisted once the
	// family has two members). e.g. "node" (web), "swift" (apple/swift-package/
	// swift-cli), "go" (go-service). Empty = the stack contributes no family.
	Family string `json:"family,omitempty"`
```

- [ ] **Step 4: Implement `Family` + `LoadFamily`**

Add to `internal/scaffold/monorepo.go`:

```go
import (
	// add to the existing import block:
	"fmt"
	"io/fs"
	"sort"
)

// ToolPin is one [tools] entry from a family contribution, kept ordered.
type ToolPin struct{ Key, Val string }

// Template is one [task_templates] body: the run string (possibly with mise
// {{ vars.X }} interpolations) and an optional description.
type Template struct{ Run, Description string }

// Family is a stack family's mise contribution, parsed from
// templates/monorepo/<name>.toml: its toolchain pins, its task templates (keyed
// by the bare task name, i.e. without the "<family>:" prefix), and the verbatim
// [task_templates.*] block text for EOF append. Hoist is set by the caller when
// the family has reached two members (so EnsureRootMise appends Raw).
type Family struct {
	Name      string
	Tools     []ToolPin
	Templates map[string]Template
	Raw       string
	Hoist     bool
}

// LoadFamily reads templates/monorepo/<name>.toml from the assets FS.
func LoadFamily(assets fs.FS, name string) (Family, error) {
	data, err := fs.ReadFile(assets, "templates/monorepo/"+name+".toml")
	if err != nil {
		return Family{}, fmt.Errorf("family %q: %w", name, err)
	}
	fam := Family{Name: name, Templates: map[string]Template{}}

	ex, err := parseExprs(data)
	if err != nil {
		return Family{}, fmt.Errorf("family %q: %w", name, err)
	}
	// Walk sections: collect [tools] pins (ordered) and each [task_templates."fam:task"].
	var rawStart = -1
	for i, e := range ex {
		if !e.kind.isTable() {
			continue
		}
		switch {
		case e.name == "tools":
			for j := i + 1; j < len(ex); j++ {
				if ex[j].kind.isTable() {
					break
				}
				if ex[j].kind.isKeyValue() {
					fam.Tools = append(fam.Tools, ToolPin{ex[j].name, ex[j].val})
				}
			}
		case strings.HasPrefix(e.name, "task_templates."):
			if rawStart == -1 {
				rawStart = e.span.start
			}
			// the template's dotted name part after task_templates. e.g. node:test
			full := strings.TrimPrefix(e.name, "task_templates.")
			task := full
			if c := strings.IndexByte(full, ':'); c >= 0 {
				task = full[c+1:] // strip the "<family>:" prefix
			}
			tpl := Template{}
			for j := i + 1; j < len(ex); j++ {
				if ex[j].kind.isTable() {
					break
				}
				if ex[j].kind.isKeyValue() {
					switch ex[j].name {
					case "run":
						tpl.Run = ex[j].val
					case "description":
						tpl.Description = ex[j].val
					}
				}
			}
			fam.Templates[task] = tpl
		}
	}
	if rawStart >= 0 {
		fam.Raw = strings.TrimRight(string(data[rawStart:]), "\n") + "\n"
	}
	// stable order for any callers that range over Tools indirectly
	_ = sort.StringsAreSorted
	return fam, nil
}
```

> Note: `Templates` keys are bare task names (`test`, `fmt:check`). The `<family>:` prefix is stripped because the member's task table is `[tasks.test]` / `[tasks."fmt:check"]`, named without the family prefix — promotion matches on the bare name.

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./internal/scaffold/ -run TestLoadFamily -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/scaffold/scaffold.go internal/scaffold/monorepo.go internal/scaffold/monorepo_test.go
git commit -m "scaffold: Family model + family manifest field + LoadFamily"
```

## Task 1.4: `EnsureRootMise` — create + merge the root config

**Files:**
- Modify: `internal/scaffold/monorepo.go`
- Modify: `internal/scaffold/monorepo_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/scaffold/monorepo_test.go`:

```go
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
	// A user-authored root with their own comment + a hand-pinned tool version.
	seed := `# my notes — keep me
monorepo_root = true

[monorepo]
config_roots = ["apps/*"]

[tools]
node = "22"   # I pinned this on purpose
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
		"my notes — keep me",          // user comment preserved
		`node = "22"`,                 // user pin NOT overwritten
		"I pinned this on purpose",    // inline comment preserved
		`pnpm = "11"`,                 // missing family tool added
		`"apps/*"`, `"cmd/*"`,         // both globs
		`[task_templates."node:test"]`, // templates hoisted (Hoist=true)
	} {
		if !strings.Contains(got, want) {
			t.Errorf("merge missing/regressed %q:\n%s", want, got)
		}
	}
	if !valid(t, got) {
		t.Errorf("merged root mise.toml does not re-parse:\n%s", got)
	}
}
```

Add helpers to the test file (and imports `"os"`, `"path/filepath"`, and `toml "github.com/pelletier/go-toml/v2"`):

```go
func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func valid(t *testing.T, s string) bool {
	t.Helper()
	var v map[string]any
	return toml.Unmarshal([]byte(s), &v) == nil
}
```

> If `read` already exists in the `scaffold` test package (it does, in `scaffold_test.go`), delete the duplicate here and reuse it.

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/scaffold/ -run TestEnsureRootMise -v`
Expected: FAIL — `undefined: EnsureRootMise`.

- [ ] **Step 3: Implement `EnsureRootMise`**

Add to `internal/scaffold/monorepo.go` (and add `"os"`, `"path"`, `"path/filepath"` to imports):

```go
// EnsureRootMise creates or merges the repo-root mise.toml so it declares
// monorepo_root, the config_roots globs for every target dir, and the union of
// the present families' [tools]. A family's [task_templates] are appended only
// when that family is marked Hoist (it has reached two members). Idempotent:
// re-running adds nothing. Preserves all existing comments and user content via
// surgical byte-range splices. Reports whether it changed the file.
func EnsureRootMise(root string, families []Family, targetDirs []string) (bool, error) {
	path := filepath.Join(root, "mise.toml")
	orig, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		orig = nil
	} else if err != nil {
		return false, err
	}

	data := orig
	if len(data) == 0 {
		data = []byte(rootSkeleton(families, targetDirs))
	} else {
		if data, err = ensureMonorepoRoot(data); err != nil {
			return false, err
		}
		for _, g := range globsFor(targetDirs) {
			if data, err = ensureConfigRoot(data, g); err != nil {
				return false, err
			}
		}
		for _, fam := range families {
			if data, err = ensureTools(data, fam.Tools); err != nil {
				return false, err
			}
		}
	}
	// Append any hoisted family's templates that aren't present yet.
	for _, fam := range families {
		if !fam.Hoist || fam.Raw == "" {
			continue
		}
		if !bytes.Contains(data, []byte(`[task_templates."`+fam.Name+`:`)) {
			if !bytes.HasSuffix(data, []byte("\n")) {
				data = append(data, '\n')
			}
			data = append(data, '\n')
			data = append(data, fam.Raw...)
		}
	}

	if bytes.Equal(data, orig) {
		return false, nil
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// rootSkeleton renders a fresh managed root config.
func rootSkeleton(families []Family, targetDirs []string) string {
	var b strings.Builder
	b.WriteString("# Managed by `specify target` — your edits and comments are preserved.\n")
	b.WriteString("# Task bodies live in each member's mise.toml until a family has two members,\n")
	b.WriteString("# then move to [task_templates] here.\n")
	b.WriteString("monorepo_root = true\n\n")
	b.WriteString("[monorepo]\n")
	globs := globsFor(targetDirs)
	quoted := make([]string, len(globs))
	for i, g := range globs {
		quoted[i] = `"` + g + `"`
	}
	b.WriteString("config_roots = [" + strings.Join(quoted, ", ") + "]\n")
	// Union of family tools, in family order then pin order.
	var pins []ToolPin
	seen := map[string]bool{}
	for _, fam := range families {
		for _, p := range fam.Tools {
			if !seen[p.Key] {
				seen[p.Key] = true
				pins = append(pins, p)
			}
		}
	}
	if len(pins) > 0 {
		b.WriteString("\n[tools]\n")
		for _, p := range pins {
			b.WriteString(p.Key + " = \"" + p.Val + "\"\n")
		}
	}
	return b.String()
}

// globsFor maps target dirs to their covering config_roots globs (parent + "/*"),
// deduped and sorted. A dir at the repo root falls back to an explicit entry.
func globsFor(dirs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, d := range dirs {
		d = filepath.ToSlash(d)
		parent := path.Dir(d)
		g := parent + "/*"
		if parent == "." || parent == "" {
			g = d // repo-root member: name it explicitly
		}
		if !seen[g] {
			seen[g] = true
			out = append(out, g)
		}
	}
	sort.Strings(out)
	return out
}

// ensureMonorepoRoot splices `monorepo_root = true` after any leading comments
// if the key is absent.
func ensureMonorepoRoot(data []byte) ([]byte, error) {
	ex, err := parseExprs(data)
	if err != nil {
		return nil, err
	}
	for _, e := range ex {
		if e.kind.isKeyValue() && e.name == "monorepo_root" {
			return data, nil
		}
	}
	// insert at the first non-comment position (or EOF preamble).
	at := 0
	for _, e := range ex {
		if unstable.Kind(e.kind) == unstable.Comment {
			if e.span.end > at {
				at = e.span.end
			}
			continue
		}
		break
	}
	ins := "monorepo_root = true\n"
	if at > 0 {
		ins = "\n" + ins
	}
	return splice(data, at, ins), nil
}

// ensureConfigRoot ensures glob is an element of [monorepo].config_roots,
// splicing it before the array's closing ] when missing. Creates the [monorepo]
// table + key if absent.
func ensureConfigRoot(data []byte, glob string) ([]byte, error) {
	ex, err := parseExprs(data)
	if err != nil {
		return nil, err
	}
	for _, e := range ex {
		if e.kind.isKeyValue() && e.name == "config_roots" {
			seg := data[e.span.start:e.span.end]
			if bytes.Contains(seg, []byte(`"`+glob+`"`)) {
				return data, nil
			}
			rb := bytes.LastIndexByte(seg, ']')
			// handle empty array "[]" vs "[ ... ]"
			inner := bytes.TrimSpace(seg[bytes.IndexByte(seg, '[')+1 : rb])
			ins := `, "` + glob + `"`
			if len(inner) == 0 {
				ins = `"` + glob + `"`
			}
			return splice(data, e.span.start+rb, ins), nil
		}
	}
	// No config_roots key — ensure a [monorepo] table holds one.
	for i, e := range ex {
		if e.kind.isTable() && e.name == "monorepo" {
			at := sectionEnd(ex, i)
			return splice(data, at, "\nconfig_roots = [\""+glob+"\"]"), nil
		}
	}
	// No [monorepo] table at all — append one.
	out := data
	if !bytes.HasSuffix(out, []byte("\n")) {
		out = append(out, '\n')
	}
	return append(out, []byte("\n[monorepo]\nconfig_roots = [\""+glob+"\"]\n")...), nil
}

// ensureTools splices each missing pin after [tools]'s last key (never
// overwriting a user-pinned version). Creates [tools] if absent.
func ensureTools(data []byte, pins []ToolPin) ([]byte, error) {
	for _, pin := range pins {
		ex, err := parseExprs(data)
		if err != nil {
			return nil, err
		}
		toolsIdx := -1
		present := false
		for i, e := range ex {
			if e.kind.isTable() && e.name == "tools" {
				toolsIdx = i
				for j := i + 1; j < len(ex); j++ {
					if ex[j].kind.isTable() {
						break
					}
					if ex[j].kind.isKeyValue() && ex[j].name == pin.Key {
						present = true
					}
				}
			}
		}
		if present {
			continue
		}
		if toolsIdx == -1 {
			out := data
			if !bytes.HasSuffix(out, []byte("\n")) {
				out = append(out, '\n')
			}
			data = append(out, []byte("\n[tools]\n"+pin.Key+" = \""+pin.Val+"\"\n")...)
			continue
		}
		at := sectionEnd(ex, toolsIdx)
		data = splice(data, at, "\n"+pin.Key+" = \""+pin.Val+"\"")
	}
	return data, nil
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/scaffold/ -run TestEnsureRootMise -v`
Expected: PASS (all three).

- [ ] **Step 5: Commit**

```bash
git add internal/scaffold/monorepo.go internal/scaffold/monorepo_test.go
git commit -m "scaffold: EnsureRootMise — create + comment-preserving merge of root config"
```

## Task 1.5: `PromoteMember` — convert still-canonical inline tasks to `extends`

**Files:**
- Modify: `internal/scaffold/monorepo.go`
- Modify: `internal/scaffold/monorepo_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/scaffold/monorepo_test.go`:

```go
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
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/scaffold/ -run TestPromoteMember -v`
Expected: FAIL — `undefined: PromoteMember`.

- [ ] **Step 3: Implement `PromoteMember`**

Add to `internal/scaffold/monorepo.go`:

```go
// PromoteMember rewrites the member mise.toml at path in place, converting each
// inline [tasks.X] whose `run` still equals the family template X's canonical run
// (after the member's own [vars] substitution) into `extends = "<family>:X"`.
// Only the `run` key/value is replaced; description, depends, custom tasks, and
// all comments are preserved (mise merges the member's own keys over the
// template's). A task the user edited away from canonical is left untouched.
// Idempotent. Reports whether it changed the file.
func PromoteMember(path string, fam Family) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	vars := memberVars(data)

	// Collect replacements (run span -> extends) for every canonical task, then
	// apply right-to-left so earlier spans stay valid.
	type repl struct {
		s   span
		ins string
	}
	var repls []repl
	ex, err := parseExprs(data)
	if err != nil {
		return false, err
	}
	for task, tpl := range fam.Templates {
		canonical := substituteVars(tpl.Run, vars)
		tableName := "tasks." + task
		for i, e := range ex {
			if !e.kind.isTable() || e.name != tableName {
				continue
			}
			for j := i + 1; j < len(ex); j++ {
				if ex[j].kind.isTable() {
					break
				}
				if ex[j].kind.isKeyValue() && ex[j].name == "run" && ex[j].val == canonical {
					repls = append(repls, repl{ex[j].span, `extends = "` + fam.Name + ":" + task + `"`})
				}
			}
		}
	}
	if len(repls) == 0 {
		return false, nil
	}
	sort.Slice(repls, func(a, b int) bool { return repls[a].s.start > repls[b].s.start })
	for _, r := range repls {
		out := make([]byte, 0, len(data))
		out = append(out, data[:r.s.start]...)
		out = append(out, r.ins...)
		out = append(out, data[r.s.end:]...)
		data = out
	}
	return true, os.WriteFile(path, data, 0o644)
}

// memberVars reads the member's [vars] table into a map (for canonical-run
// substitution). Best-effort: a malformed file yields an empty map.
func memberVars(data []byte) map[string]string {
	out := map[string]string{}
	ex, err := parseExprs(data)
	if err != nil {
		return out
	}
	for i, e := range ex {
		if e.kind.isTable() && e.name == "vars" {
			for j := i + 1; j < len(ex); j++ {
				if ex[j].kind.isTable() {
					break
				}
				if ex[j].kind.isKeyValue() {
					out[ex[j].name] = ex[j].val
				}
			}
		}
	}
	return out
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/scaffold/ -run TestPromoteMember -v`
Expected: PASS (all three).

- [ ] **Step 5: Run the whole scaffold package**

Run: `go test ./internal/scaffold/ -v`
Expected: PASS (existing tests unaffected).

- [ ] **Step 6: Commit**

```bash
git add internal/scaffold/monorepo.go internal/scaffold/monorepo_test.go
git commit -m "scaffold: PromoteMember — convert still-canonical inline tasks to extends"
```

## Task 1.6: The `node` family contribution file + drop web's `[tools]`

**Files:**
- Create: `internal/coreassets/templates/monorepo/node.toml`
- Modify: `internal/coreassets/templates/scaffolds/web/files/mise.toml` (drop `[tools]`)
- Modify: `internal/coreassets/templates/scaffolds/web/scaffold.json` (add `family`, change `command`)

- [ ] **Step 1: Author the family file**

Create `internal/coreassets/templates/monorepo/node.toml` — the bodies must match web's current inline task bodies **exactly** (a drift test in Task 1.7 enforces this). Note `1password` replaces the old `op` registry name (the binary stays `op`):

```toml
# node family — web (and future node-cli). Toolchain hoisted to the repo-root
# mise.toml from the first member; these task templates are written there only
# once the family has a second member. Bodies must match the member scaffolds'
# inline tasks (a drift test enforces it).
[tools]
node = "24"
pnpm = "11"
gh = "2.94"
1password = "2"

[task_templates."node:routes"]
description = "generate the TanStack Router tree (app/routes.gen.ts)"
run = "tsr generate"

[task_templates."node:dev"]
run = "vite dev"

[task_templates."node:test"]
description = "run Vitest with the junit reporter the engine joins"
run = "vitest run --reporter=junit --outputFile=junit.xml"

[task_templates."node:typecheck"]
description = "type-check with tsgo (needs the generated route tree)"
run = "tsgo --noEmit"

[task_templates."node:fmt"]
description = "format app/ in place (Oxfmt)"
run = "oxfmt app"

[task_templates."node:fmt:check"]
description = "fail if app/ needs formatting (Oxfmt)"
run = "oxfmt --check app"

[task_templates."node:lint"]
description = "lint app/ (Oxlint); needs the generated route tree"
run = "oxlint app"

[task_templates."node:build"]
run = "vite build"
```

- [ ] **Step 2: Drop `[tools]` from web's member mise.toml**

Edit `internal/coreassets/templates/scaffolds/web/files/mise.toml` — remove the first 7 lines (the `[tools]` block through `op = "2"`), leaving the file to start at the `# Call npm binaries…` comment + `[env]`. The member keeps all inline task bodies (they are the one-member shape). The result begins:

```toml
# Call npm binaries (vite, vitest, tsgo, oxlint, oxfmt, tsr) by bare name — no npx.
[env]
_.path = ['{{config_root}}/node_modules/.bin']

[tasks.routes]
…
```

- [ ] **Step 3: Add `family` + change `command` in web's scaffold.json**

Edit `internal/coreassets/templates/scaffolds/web/scaffold.json`:
- Add `"family": "node",` near the top (after `"stack": "web",`).
- Change the `target.command` from `"cd {{.Dir}} && mise run test"` to `"mise //{{.Dir}}:test"`.

- [ ] **Step 4: Verify it builds (embedded assets)**

Run: `go build ./...`
Expected: success (the new file is embedded via `internal/coreassets`).

- [ ] **Step 5: Commit**

```bash
git add internal/coreassets/templates/monorepo/node.toml \
  internal/coreassets/templates/scaffolds/web/files/mise.toml \
  internal/coreassets/templates/scaffolds/web/scaffold.json
git commit -m "scaffold(web): node family file; hoist [tools]; native mise //dir:test"
```

## Task 1.7: Drift guard — family templates match member inline bodies

This catches the maintenance coupling: the family `.toml` bodies and the member scaffolds' inline bodies must agree, or promotion silently stops converting.

**Files:**
- Modify: `internal/scaffold/web_test.go` (or a new `internal/scaffold/monorepo_assets_test.go`)

- [ ] **Step 1: Write the test**

Create `internal/scaffold/monorepo_assets_test.go`:

```go
package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

// TestNodeFamilyMatchesWebInline asserts the node family templates' run strings
// equal the web member scaffold's inline task bodies — the coupling promotion
// relies on. If you change one, change the other.
func TestNodeFamilyMatchesWebInline(t *testing.T) {
	fam, err := LoadFamily(coreassets.FS, "node")
	if err != nil {
		t.Fatal(err)
	}
	sub, _ := fs.Sub(coreassets.FS, "templates/scaffolds/web")
	dir := t.TempDir()
	if _, err := Render(sub, dir, Data{Name: "web", Dir: "apps/web"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "mise.toml"))
	if err != nil {
		t.Fatal(err)
	}
	// web member must no longer carry [tools] (hoisted to root).
	if strings.Contains(string(data), "[tools]") {
		t.Errorf("web member must not declare [tools] (hoisted to root):\n%s", data)
	}
	ex, _ := parseExprs(data)
	vars := memberVars(data)
	for task, tpl := range fam.Templates {
		want := substituteVars(tpl.Run, vars)
		var got string
		found := false
		for i, e := range ex {
			if e.kind.isTable() && e.name == "tasks."+task {
				for j := i + 1; j < len(ex); j++ {
					if ex[j].kind.isTable() {
						break
					}
					if ex[j].kind.isKeyValue() && ex[j].name == "run" {
						got, found = ex[j].val, true
					}
				}
			}
		}
		if !found {
			t.Errorf("web member has no inline [tasks.%s] for family template node:%s", task, task)
			continue
		}
		if got != want {
			t.Errorf("drift: node:%s\n  family:  %q\n  member:  %q", task, want, got)
		}
	}
}
```

Add `"io/fs"` to the imports.

- [ ] **Step 2: Run it**

Run: `go test ./internal/scaffold/ -run TestNodeFamilyMatchesWebInline -v`
Expected: PASS. If it FAILS, the family file and `web/files/mise.toml` disagree — reconcile them (they were authored to match in Task 1.6).

- [ ] **Step 3: Commit**

```bash
git add internal/scaffold/monorepo_assets_test.go
git commit -m "scaffold: drift guard — node family templates == web inline bodies"
```

## Task 1.8: Wire `wireMonorepo` into the CLI

**Files:**
- Create: `cmd/specify/monorepo.go`
- Modify: `cmd/specify/main.go` (call `wireMonorepo` in `targetAddCmd` and `registerTarget`)

- [ ] **Step 1: Implement the orchestrator**

Create `cmd/specify/monorepo.go`:

```go
package main

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/markmals/speckit/internal/config"
	"github.com/markmals/speckit/internal/coreassets"
	"github.com/markmals/speckit/internal/scaffold"
)

// wireMonorepo brings the repo-root mise.toml in line with the current set of
// targets: it merges monorepo_root + the config_roots globs + the union of
// present families' [tools], and — for any family that now has two or more
// members — writes that family's [task_templates] and converts each member's
// still-canonical inline tasks to `extends`. Called after a target is recorded
// (so the new member is counted). Idempotent. A repo whose targets declare no
// family (e.g. only stacks without one) is left untouched.
func wireMonorepo(root string) error {
	cfg, found, err := config.Load(root)
	if err != nil || !found {
		return err
	}

	// Map each target to its family + dir; count members per family.
	type member struct{ dir, family string }
	var members []member
	count := map[string]int{}
	famNames := map[string]bool{}
	var dirs []string
	for _, t := range cfg.Targets {
		if t.Stack == "" {
			continue
		}
		fam, dir, err := familyAndDir(t)
		if err != nil {
			return err
		}
		if fam == "" {
			continue
		}
		members = append(members, member{dir, fam})
		count[fam]++
		famNames[fam] = true
		dirs = append(dirs, dir)
	}
	if len(famNames) == 0 {
		return nil // no family-bearing targets; nothing to wire
	}

	// Load each present family, marking Hoist when it has >=2 members.
	var families []scaffold.Family
	loaded := map[string]scaffold.Family{}
	var names []string
	for n := range famNames {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fam, err := scaffold.LoadFamily(coreassets.FS, n)
		if err != nil {
			return err
		}
		fam.Hoist = count[n] >= 2
		families = append(families, fam)
		loaded[n] = fam
	}

	if _, err := scaffold.EnsureRootMise(root, families, dirs); err != nil {
		return err
	}
	// Promote members of any hoisted family (safe: only canonical tasks convert).
	for _, m := range members {
		fam := loaded[m.family]
		if !fam.Hoist {
			continue
		}
		mise := filepath.Join(root, m.dir, "mise.toml")
		if _, err := scaffold.PromoteMember(mise, fam); err != nil {
			return err
		}
	}
	return nil
}

// familyAndDir resolves a target's family (from its stack's scaffold manifest)
// and member dir (the recorded report's parent, falling back to the stack's
// memberDir/<name> is not needed — the target's Source/Report already encode the
// dir, but we derive it from the stack manifest for robustness).
func familyAndDir(t config.Target) (family, dir string, err error) {
	sub, err := fs.Sub(coreassets.FS, "templates/scaffolds/"+t.Stack)
	if err != nil {
		return "", "", nil // stack without a scaffold (go-cli, node-cli): no family
	}
	m, err := scaffold.LoadManifest(sub)
	if err != nil {
		return "", "", fmt.Errorf("stack %q: %w", t.Stack, err)
	}
	return m.Family, filepath.Dir(t.Report), nil
}
```

> `filepath.Dir(t.Report)` recovers the member dir: every scaffold's `report` is `{{.Dir}}/<file>` (e.g. `apps/web/junit.xml` → `apps/web`; `cmd/api/test.gotest.json` → `cmd/api`). This avoids re-deriving `memberDir/<name>` and works for `--dir` overrides too.

- [ ] **Step 2: Call it from `targetAddCmd`**

In `cmd/specify/main.go`, in `targetAddCmd`'s `RunE`, immediately **after** the `config.AddTarget(...)` block (around line 339, before the pack projection), add:

```go
				if err := wireMonorepo("."); err != nil {
					return fmt.Errorf("wiring mise monorepo: %w", err)
				}
```

- [ ] **Step 3: Call it from `registerTarget`**

In `registerTarget`, after the `config.AddTarget(root, o.name, t)` block (around line 483, before the pack projection), add:

```go
	if err := wireMonorepo(root); err != nil {
		return fmt.Errorf("wiring mise monorepo: %w", err)
	}
```

- [ ] **Step 4: Build + run existing CLI tests**

Run: `go build ./... && go test ./cmd/... ./internal/...`
Expected: PASS. (No CLI test asserts the old `cd … && mise run test` command directly; if one does, update it to `mise //<dir>:test`.)

- [ ] **Step 5: Commit**

```bash
git add cmd/specify/monorepo.go cmd/specify/main.go
git commit -m "cli: wire mise monorepo root config + promotion into target add/register"
```

## Task 1.9: Refresh the engine comment + end-to-end CLI test

**Files:**
- Modify: `internal/engine/verify_run.go:37-40` (comment only)
- Create: `cmd/specify/monorepo_e2e_test.go`

- [ ] **Step 1: Refresh the engine comment**

In `internal/engine/verify_run.go`, update the comment above the `if cfg.Command != ""` block (line ~37) to note the native form. Change:

```go
		// cfg.Command is a shell string from the project's own .speckit/specs.json
		// target (developer-controlled, like a Mise task's `run`), so shell
		// interpretation is intended — the project owner is the trust boundary.
```
to:

```go
		// cfg.Command is a shell string from the project's own .speckit/specs.json
		// target (developer-controlled, like a Mise task's `run`), so shell
		// interpretation is intended — the project owner is the trust boundary.
		// Newly-scaffolded targets record the native monorepo form `mise //<dir>:test`
		// (run with cwd = the member dir); pre-existing targets may record the older
		// `cd <dir> && mise run test` — both are valid here.
```

- [ ] **Step 2: Write the end-to-end test**

This drives the real CLI helpers without a network. Create `cmd/specify/monorepo_e2e_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/config"
)

// TestWireMonorepoInlineThenPromote exercises both paths: one node member stays
// inline (no templates), and a second node member triggers promotion.
func TestWireMonorepoInlineThenPromote(t *testing.T) {
	root := t.TempDir()
	mustChdir(t, root)

	// --- member 1: apps/web (inline) ---
	writeWebMember(t, root, "apps/web")
	if err := config.AddTarget(root, "web", config.Target{
		Stack: "web", Command: "mise //apps/web:test", Format: "junit",
		Report: "apps/web/junit.xml", Source: "apps/web/app",
	}); err != nil {
		t.Fatal(err)
	}
	if err := wireMonorepo(root); err != nil {
		t.Fatal(err)
	}
	rootMise := read(t, filepath.Join(root, "mise.toml"))
	if !strings.Contains(rootMise, "monorepo_root = true") || !strings.Contains(rootMise, `config_roots = ["apps/*"]`) {
		t.Errorf("root config wrong after member 1:\n%s", rootMise)
	}
	if strings.Contains(rootMise, "[task_templates") {
		t.Errorf("one member must not hoist templates:\n%s", rootMise)
	}
	if !strings.Contains(rootMise, `node = "24"`) || !strings.Contains(rootMise, "1password") {
		t.Errorf("root [tools] missing node family pins:\n%s", rootMise)
	}
	if strings.Contains(read(t, filepath.Join(root, "apps/web/mise.toml")), "extends =") {
		t.Error("single member must stay inline (no extends)")
	}

	// --- member 2: apps/web2 (promotion) ---
	writeWebMember(t, root, "apps/web2")
	if err := config.AddTarget(root, "web2", config.Target{
		Stack: "web", Command: "mise //apps/web2:test", Format: "junit",
		Report: "apps/web2/junit.xml", Source: "apps/web2/app",
	}); err != nil {
		t.Fatal(err)
	}
	if err := wireMonorepo(root); err != nil {
		t.Fatal(err)
	}
	rootMise = read(t, filepath.Join(root, "mise.toml"))
	if !strings.Contains(rootMise, `[task_templates."node:test"]`) {
		t.Errorf("two members must hoist node templates:\n%s", rootMise)
	}
	if !strings.Contains(rootMise, `config_roots = ["apps/*"]`) {
		t.Errorf("one apps/* glob must still cover both members:\n%s", rootMise)
	}
	// both members now extend the shared templates.
	for _, d := range []string{"apps/web", "apps/web2"} {
		m := read(t, filepath.Join(root, d, "mise.toml"))
		if !strings.Contains(m, `extends = "node:test"`) {
			t.Errorf("%s not promoted to extends:\n%s", d, m)
		}
	}
}

// writeWebMember renders the real embedded web member into root/dir (mise.toml only
// is needed for this test; reuse the scaffold render in a focused helper).
func writeWebMember(t *testing.T, root, dir string) {
	t.Helper()
	// Render the web scaffold's mise.toml by reading the embedded member file.
	src, err := coreassetsReadMember("web")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, dir, "mise.toml"), src, 0o644); err != nil {
		t.Fatal(err)
	}
}
```

Add the two small test helpers (in the same file):

```go
func mustChdir(t *testing.T, dir string) {
	t.Helper()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// coreassetsReadMember returns a stack's embedded files/mise.toml (web) verbatim.
func coreassetsReadMember(stack string) ([]byte, error) {
	return coreassets.FS.ReadFile("templates/scaffolds/" + stack + "/files/mise.toml")
}
```

Add imports `"github.com/markmals/speckit/internal/coreassets"`.

> The web member `mise.toml` is a plain file (not `.tmpl`), so copying it verbatim is faithful. If `read` already exists elsewhere in `package main` tests, drop the duplicate.

- [ ] **Step 3: Run the e2e test**

Run: `go test ./cmd/specify/ -run TestWireMonorepo -v`
Expected: PASS (both inline and promotion assertions).

- [ ] **Step 4: Run the full gate**

Run: `mise run ci`
Expected: PASS (fmt:check, build, vet, test).

- [ ] **Step 5: Commit**

```bash
git add internal/engine/verify_run.go cmd/specify/monorepo_e2e_test.go
git commit -m "engine: note native mise //dir:task; e2e test for inline→promotion"
```

## Task 1.10: Update the existing web scaffold assertion test

**Files:**
- Modify: `internal/scaffold/web_test.go`

- [ ] **Step 1: Update the mise.toml assertions**

`TestWebScaffold` (line ~132) asserts the member `mise.toml` contains the quality task names. Those still exist (inline). Add an assertion that `[tools]` is **gone** from the member, and that `target.command` is the native form. After the existing `for _, task := range []string{...}` block (line ~140), add:

```go
	if strings.Contains(string(mise), "[tools]") {
		t.Errorf("web member mise.toml must not declare [tools] (hoisted to root):\n%s", mise)
	}
```

And update the `RenderTarget` assertion (line ~158) to also check the command:

```go
	if rt.Command != "mise //apps/web:test" {
		t.Errorf("web target.command = %q, want mise //apps/web:test", rt.Command)
	}
```

- [ ] **Step 2: Run it**

Run: `go test ./internal/scaffold/ -run TestWebScaffold -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/scaffold/web_test.go
git commit -m "test(web): assert member drops [tools] + records native mise command"
```

---

# Stage 2 — The `swift` family

The hardest family: three stacks (`apple`, `swift-package`, `swift-cli`) with overlapping-but-not-identical tasks. The shared set is `test` (unified via a `package_path` var), `fmt`, and `lint` (a superset directory walk). Stack-specific tasks (`apple`'s tuist `build`/`generate`/`test:app`/`launch:macos`; the `swift build` of package/cli; `swift-cli`'s `run`) stay inline. `tuist` is apple-specific, so it stays a **member-level** `[tools]` in apple — the `swift` family contributes **no** `[tools]`.

## Task 2.1: Author `monorepo/swift.toml`

**Files:**
- Create: `internal/coreassets/templates/monorepo/swift.toml`

- [ ] **Step 1: Author the file**

Create `internal/coreassets/templates/monorepo/swift.toml`. No `[tools]` table; bodies use mise `{{ vars.X }}` (raw — never Go-rendered) and the superset dir walk:

```toml
# swift family — apple, swift-package, swift-cli. No [tools]: swift + swift-format
# come from the active toolchain, and tuist is apple-specific (stays a member-level
# [tools] in the apple scaffold). Templates are written to the root mise.toml once
# the family has a second member. Bodies must match the member scaffolds' inline
# tasks (a drift test enforces it).

[task_templates."swift:test"]
description = "run the test suite, writing the event-stream report the engine joins"
run = "swift test --package-path {{ vars.package_path }} --event-stream-output-path test.swift-events.ndjson --event-stream-version 0"

[task_templates."swift:build"]
description = "build the package"
run = "swift build"

[task_templates."swift:fmt"]
description = "format Swift sources in place (swift-format). Pass paths to format just those."
run = '''
if [ "$#" -gt 0 ]; then
  for f in "$@"; do case "$f" in *.swift) swift format --in-place "$f" ;; esac; done
else
  for d in Core macOS iOS Sources Tests; do
    if [ -d "$d" ]; then swift format --in-place --recursive "$d"; fi
  done
fi
'''

[task_templates."swift:lint"]
description = "lint Swift sources (swift-format, strict)"
run = '''
dirs=""
for d in Core macOS iOS Sources Tests; do
  if [ -d "$d" ]; then dirs="$dirs $d"; fi
done
if [ -n "$dirs" ]; then swift format lint --strict --recursive $dirs; fi
'''
```

> `swift:test` uses `--package-path {{ vars.package_path }}`: apple sets `package_path = "Core"`; swift-package/cli set `package_path = "."` (equivalent to the bare `swift test`). `swift:build` (`swift build`) is shared by package + cli; apple's tuist build stays inline and won't match the canonical, so promotion leaves it alone.

## Task 2.2: Align the swift member scaffolds (vars + canonical bodies)

**Files:**
- Modify: `internal/coreassets/templates/scaffolds/apple/files/mise.toml.tmpl`
- Modify: `internal/coreassets/templates/scaffolds/swift-package/files/mise.toml.tmpl`
- Modify: `internal/coreassets/templates/scaffolds/swift-cli/files/mise.toml.tmpl`

- [ ] **Step 1: apple — add `[vars]`, align `test`, keep tuist tasks inline**

Edit `apple/files/mise.toml.tmpl`. Keep the `[tools] tuist = "4.196.1"` (apple-specific). After it, add a `[vars]` table, and change the `test` task's `run` to the canonical `--package-path Core` form (it already is `--package-path Core`; ensure it matches the template exactly). The `fmt`/`lint` bodies already use `Core macOS iOS` — change them to the **superset** `Core macOS iOS Sources Tests` so they equal the family canonical:

```toml
[tools]
tuist = "4.196.1"

[vars]
package_path = "Core"
scheme = "{{pascal .Name}}"
```

In the `test` task ensure exactly:
```toml
[tasks.test]
description = "run the test suite, writing the event-stream report the engine joins"
run = "swift test --package-path Core --event-stream-output-path test.swift-events.ndjson --event-stream-version 0"
```
(Change the description from the current `"run the Core suite, …"` to match the family template's description string — the drift test compares `run` only, but keep them consistent for readers.)

In `fmt` and `lint`, change the dir lists from `Core macOS iOS` to `Core macOS iOS Sources Tests`. Leave `generate`, `build`, `test:app`, `launch:macos` exactly as they are (apple-specific, stay inline).

- [ ] **Step 2: swift-package — add `[vars]`, align `test`/`fmt`/`lint`/`build`**

Edit `swift-package/files/mise.toml.tmpl`. Add at the top (it has no `[tools]`):

```toml
[vars]
package_path = "."
```

Change `test` to the canonical form (add `--package-path .`):
```toml
[tasks.test]
description = "run the test suite, writing the event-stream report the engine joins"
run = "swift test --package-path . --event-stream-output-path test.swift-events.ndjson --event-stream-version 0"
```
Change `build` description to `"build the package"` and `run = "swift build"` (already `swift build`). Change `fmt`/`lint` dir lists from `Sources Tests` to the superset `Core macOS iOS Sources Tests`.

- [ ] **Step 3: swift-cli — add `[vars]`, align `test`/`fmt`/`lint`; keep `run`/`build` inline as needed**

Edit `swift-cli/files/mise.toml.tmpl`. Add:
```toml
[vars]
package_path = "."
```
Change `test` to the canonical `swift test --package-path . --event-stream-output-path test.swift-events.ndjson --event-stream-version 0`. The `build` description here is `"build the package (library + executable)"` — the family `swift:build` description differs, but promotion compares `run` only (`swift build` == canonical), so it will convert; the member keeps its own description (mise merges it over the template). Keep `run` (`swift run {{kebab .Name}}`) inline (no family template). Change `fmt`/`lint` dir lists to the superset.

- [ ] **Step 4: Build (embedded assets)**

Run: `go build ./...`
Expected: success.

- [ ] **Step 5: Commit**

```bash
git add internal/coreassets/templates/monorepo/swift.toml \
  internal/coreassets/templates/scaffolds/apple/files/mise.toml.tmpl \
  internal/coreassets/templates/scaffolds/swift-package/files/mise.toml.tmpl \
  internal/coreassets/templates/scaffolds/swift-cli/files/mise.toml.tmpl
git commit -m "scaffold(swift): family file + align member test/fmt/lint to canonical (vars+superset)"
```

## Task 2.3: `family` + native command for the swift scaffolds; drift guard

**Files:**
- Modify: `internal/coreassets/templates/scaffolds/{apple,swift-package,swift-cli}/scaffold.json`
- Modify: `internal/scaffold/monorepo_assets_test.go`

- [ ] **Step 1: Add `family` + change `command` in each swift scaffold.json**

In each of `apple`, `swift-package`, `swift-cli` `scaffold.json`:
- Add `"family": "swift",`.
- Change `target.command` to `"mise //{{.Dir}}:test"`.

(Confirm the current swift commands first — they follow the `cd {{.Dir}} && mise run test` pattern; replace with the native form.)

- [ ] **Step 2: Extend the drift guard to the swift family**

Add to `internal/scaffold/monorepo_assets_test.go` a table-driven test that renders each swift member and checks its inline `test`/`fmt`/`lint` (and `build` for package/cli) bodies equal the `swift:*` canonical after that member's `[vars]`:

```go
func TestSwiftFamilyMatchesMemberInline(t *testing.T) {
	fam, err := LoadFamily(coreassets.FS, "swift")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		stack string
		name  string
		dir   string
		tasks []string
	}{
		{"apple", "Photos", "apps/Photos", []string{"test", "fmt", "lint"}},
		{"swift-package", "Widgets", "packages/Widgets", []string{"test", "build", "fmt", "lint"}},
		{"swift-cli", "Tool", "packages/Tool", []string{"test", "build", "fmt", "lint"}},
	}
	for _, c := range cases {
		sub, _ := fs.Sub(coreassets.FS, "templates/scaffolds/"+c.stack)
		dir := t.TempDir()
		if _, err := Render(sub, dir, Data{Name: c.name, Dir: c.dir}); err != nil {
			t.Fatalf("%s render: %v", c.stack, err)
		}
		data, _ := os.ReadFile(filepath.Join(dir, "mise.toml"))
		ex, _ := parseExprs(data)
		vars := memberVars(data)
		for _, task := range c.tasks {
			want := substituteVars(fam.Templates[task].Run, vars)
			got, found := inlineRun(ex, task)
			if !found {
				t.Errorf("%s: no inline [tasks.%s]", c.stack, task)
				continue
			}
			if got != want {
				t.Errorf("drift %s swift:%s\n  family: %q\n  member: %q", c.stack, task, want, got)
			}
		}
	}
}

// inlineRun returns the decoded run value of [tasks.<task>] from parsed exprs.
func inlineRun(ex []expr, task string) (string, bool) {
	for i, e := range ex {
		if e.kind.isTable() && e.name == "tasks."+task {
			for j := i + 1; j < len(ex); j++ {
				if ex[j].kind.isTable() {
					break
				}
				if ex[j].kind.isKeyValue() && ex[j].name == "run" {
					return ex[j].val, true
				}
			}
		}
	}
	return "", false
}
```

> Refactor the Stage-1 `TestNodeFamilyMatchesWebInline` to reuse `inlineRun` (DRY).

- [ ] **Step 3: Run it**

Run: `go test ./internal/scaffold/ -run 'TestSwiftFamilyMatches|TestNodeFamilyMatches' -v`
Expected: PASS. A drift failure means a member body and `swift.toml` disagree — reconcile.

- [ ] **Step 4: Commit**

```bash
git add internal/coreassets/templates/scaffolds/apple/scaffold.json \
  internal/coreassets/templates/scaffolds/swift-package/scaffold.json \
  internal/coreassets/templates/scaffolds/swift-cli/scaffold.json \
  internal/scaffold/monorepo_assets_test.go
git commit -m "scaffold(swift): family field + native command; drift guard across 3 stacks"
```

## Task 2.4: Update swift scaffold assertion tests + e2e promotion across stacks

**Files:**
- Modify: `internal/scaffold/apple_test.go`, `swift_package_test.go`, `swift_cli_test.go`
- Modify: `cmd/specify/monorepo_e2e_test.go`

- [ ] **Step 1: Update each swift scaffold test**

In each test, the member `mise.toml` now carries `[vars]` and the canonical bodies. Assert:
- `apple_test.go`: `[tools]` still present (tuist stays); `[vars]` present with `package_path = "Core"` and `scheme`; `target.command` == `mise //<dir>:test`.
- `swift_package_test.go` / `swift_cli_test.go`: `[vars] package_path = "."` present; `test` run contains `--package-path .`; command native.

Add to each (adjust the expected dir/name to what the test already uses):

```go
	if !strings.Contains(string(mise), "[vars]") {
		t.Errorf("member mise.toml missing [vars]:\n%s", mise)
	}
```

- [ ] **Step 2: Add a cross-stack promotion e2e**

Add to `cmd/specify/monorepo_e2e_test.go` a test that registers an `apple` member then a `swift-package` member and asserts the swift templates hoist and both members convert their shared tasks:

```go
func TestWireMonorepoSwiftCrossStackPromotion(t *testing.T) {
	root := t.TempDir()
	mustChdir(t, root)

	renderMember(t, root, "apple", "Photos", "apps/Photos")
	if err := config.AddTarget(root, "Photos", config.Target{
		Stack: "apple", Command: "mise //apps/Photos:test", Format: "swift",
		Report: "apps/Photos/test.swift-events.ndjson", Source: "apps/Photos/Core",
	}); err != nil {
		t.Fatal(err)
	}
	if err := wireMonorepo(root); err != nil {
		t.Fatal(err)
	}
	// one swift member: inline, root has no swift templates and no [tools] tuist
	// (tuist stays in the apple member).
	rootMise := read(t, filepath.Join(root, "mise.toml"))
	if strings.Contains(rootMise, "[task_templates") {
		t.Errorf("one swift member must stay inline:\n%s", rootMise)
	}

	renderMember(t, root, "swift-package", "Widgets", "packages/Widgets")
	if err := config.AddTarget(root, "Widgets", config.Target{
		Stack: "swift-package", Command: "mise //packages/Widgets:test", Format: "swift",
		Report: "packages/Widgets/test.swift-events.ndjson", Source: "packages/Widgets/Sources",
	}); err != nil {
		t.Fatal(err)
	}
	if err := wireMonorepo(root); err != nil {
		t.Fatal(err)
	}
	rootMise = read(t, filepath.Join(root, "mise.toml"))
	if !strings.Contains(rootMise, `[task_templates."swift:test"]`) {
		t.Errorf("two swift members must hoist swift templates:\n%s", rootMise)
	}
	// both globs present (apps/* and packages/*).
	if !strings.Contains(rootMise, `"apps/*"`) || !strings.Contains(rootMise, `"packages/*"`) {
		t.Errorf("config_roots missing a swift glob:\n%s", rootMise)
	}
	for _, d := range []string{"apps/Photos", "packages/Widgets"} {
		m := read(t, filepath.Join(root, d, "mise.toml"))
		if !strings.Contains(m, `extends = "swift:test"`) {
			t.Errorf("%s test not promoted:\n%s", d, m)
		}
	}
	// apple keeps its tuist build inline (not converted).
	ap := read(t, filepath.Join(root, "apps/Photos/mise.toml"))
	if strings.Contains(ap, `extends = "swift:build"`) {
		t.Errorf("apple tuist build must stay inline:\n%s", ap)
	}
}

// renderMember renders a stack's real embedded member tree into root/dir.
func renderMember(t *testing.T, root, stack, name, dir string) {
	t.Helper()
	sub, err := fs.Sub(coreassets.FS, "templates/scaffolds/"+stack)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scaffold.Render(sub, filepath.Join(root, dir), scaffold.Data{Name: name, Dir: dir}); err != nil {
		t.Fatal(err)
	}
}
```

Add imports `"io/fs"`, `"github.com/markmals/speckit/internal/scaffold"`. Replace the Stage-1 `writeWebMember` with `renderMember(t, root, "web", "web", dir)` for consistency, or keep both.

- [ ] **Step 3: Run + gate**

Run: `go test ./internal/scaffold/ ./cmd/specify/ -run 'Swift|Apple|WireMonorepo' -v && mise run ci`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/scaffold/apple_test.go internal/scaffold/swift_package_test.go internal/scaffold/swift_cli_test.go cmd/specify/monorepo_e2e_test.go
git commit -m "test(swift): member [vars] assertions + cross-stack promotion e2e"
```

---

# Stage 3 — The `go` family

One stack today (`go-service`, `sharedModule: true`). The `--with openapi` `generate` task stays inline (member-specific). `ensureRootGoMod` is unchanged and independent of this wiring.

## Task 3.1: Author `monorepo/go.toml` + drop go-service `[tools]`

**Files:**
- Create: `internal/coreassets/templates/monorepo/go.toml`
- Modify: `internal/coreassets/templates/scaffolds/go-service/files/mise.toml.tmpl`
- Modify: `internal/coreassets/templates/scaffolds/go-service/scaffold.json`

- [ ] **Step 1: Author the family file**

Create `internal/coreassets/templates/monorepo/go.toml`:

```toml
# go family — go-service (and future go-tui). Toolchain hoisted to the root
# mise.toml; task templates written there once the family has a second member.
# The go-service --with openapi `generate` task is member-specific and stays inline.
[tools]
go = "1.26"

[task_templates."go:dev"]
description = "run the service"
run = "go run ."

[task_templates."go:build"]
description = "build the service"
run = "go build ./..."

[task_templates."go:test"]
description = "run the Go suite, writing the JSON report the engine joins"
run = "go test -json ./... > test.gotest.json"

[task_templates."go:vet"]
description = "go vet"
run = "go vet ./..."

[task_templates."go:fmt"]
description = "format in place (gofmt)"
run = "gofmt -w ."

[task_templates."go:fmt:check"]
description = "fail if any file needs formatting"
run = 'test -z "$(gofmt -l .)"'
```

- [ ] **Step 2: Drop `[tools]` from go-service member**

Edit `go-service/files/mise.toml.tmpl` — remove the `[tools]\ngo = "1.26"` block (lines ~4-5). Keep all inline tasks (including the conditional `{{if .Features.openapi}} … generate … {{end}}` block). The file now begins with the header comment then `[tasks.dev]`.

- [ ] **Step 3: `family` + native command in go-service scaffold.json**

Edit `go-service/scaffold.json`:
- Add `"family": "go",`.
- Change `target.command` from `"cd {{.Dir}} && mise run test"` to `"mise //{{.Dir}}:test"`.

- [ ] **Step 4: Build**

Run: `go build ./...`
Expected: success.

- [ ] **Step 5: Commit**

```bash
git add internal/coreassets/templates/monorepo/go.toml \
  internal/coreassets/templates/scaffolds/go-service/files/mise.toml.tmpl \
  internal/coreassets/templates/scaffolds/go-service/scaffold.json
git commit -m "scaffold(go): go family file; hoist [tools]; native mise //dir:test"
```

## Task 3.2: Drift guard + go-service test updates + e2e

**Files:**
- Modify: `internal/scaffold/monorepo_assets_test.go`
- Modify: `internal/scaffold/go_service_test.go`
- Modify: `cmd/specify/monorepo_e2e_test.go`

- [ ] **Step 1: Extend the drift guard**

Add to `monorepo_assets_test.go`:

```go
func TestGoFamilyMatchesMemberInline(t *testing.T) {
	fam, err := LoadFamily(coreassets.FS, "go")
	if err != nil {
		t.Fatal(err)
	}
	sub, _ := fs.Sub(coreassets.FS, "templates/scaffolds/go-service")
	dir := t.TempDir()
	if _, err := Render(sub, dir, Data{Name: "api", Dir: "cmd/api"}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "mise.toml"))
	if strings.Contains(string(data), "[tools]") {
		t.Errorf("go-service member must not declare [tools]:\n%s", data)
	}
	ex, _ := parseExprs(data)
	for _, task := range []string{"dev", "build", "test", "vet", "fmt", "fmt:check"} {
		got, found := inlineRun(ex, task)
		if !found {
			t.Errorf("no inline [tasks.%s]", task)
			continue
		}
		if got != fam.Templates[task].Run {
			t.Errorf("drift go:%s\n  family: %q\n  member: %q", task, fam.Templates[task].Run, got)
		}
	}
}
```

- [ ] **Step 2: Update `go_service_test.go`**

Assert the member dropped `[tools]` and records the native command:

```go
	if strings.Contains(string(mise), "[tools]") {
		t.Errorf("go-service member must not declare [tools] (hoisted to root):\n%s", mise)
	}
```
And update any `RenderTarget`/command assertion to `mise //cmd/<name>:test`. (Find the existing target assertion in the test and adjust.)

- [ ] **Step 3: Add a go e2e (two go-service members → promotion, shared go.mod intact)**

Add to `monorepo_e2e_test.go`:

```go
func TestWireMonorepoGoPromotion(t *testing.T) {
	root := t.TempDir()
	mustChdir(t, root)
	for _, m := range []struct{ name, dir string }{{"api", "cmd/api"}, {"worker", "cmd/worker"}} {
		renderMember(t, root, "go-service", m.name, m.dir)
		if err := config.AddTarget(root, m.name, config.Target{
			Stack: "go-service", Command: "mise //" + m.dir + ":test", Format: "gotest",
			Report: m.dir + "/test.gotest.json", Source: m.dir, Bindings: "scoped",
		}); err != nil {
			t.Fatal(err)
		}
		if err := wireMonorepo(root); err != nil {
			t.Fatal(err)
		}
	}
	rootMise := read(t, filepath.Join(root, "mise.toml"))
	if !strings.Contains(rootMise, `[task_templates."go:test"]`) || !strings.Contains(rootMise, `config_roots = ["cmd/*"]`) {
		t.Errorf("go family not hoisted / glob wrong:\n%s", rootMise)
	}
	for _, d := range []string{"cmd/api", "cmd/worker"} {
		if !strings.Contains(read(t, filepath.Join(root, d, "mise.toml")), `extends = "go:test"`) {
			t.Errorf("%s not promoted", d)
		}
	}
}
```

- [ ] **Step 4: Run + gate**

Run: `go test ./internal/scaffold/ ./cmd/specify/ -v && mise run ci`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/scaffold/monorepo_assets_test.go internal/scaffold/go_service_test.go cmd/specify/monorepo_e2e_test.go
git commit -m "test(go): drift guard + member assertions + two-member promotion e2e"
```

---

# Stage 4 — Docs + memory

## Task 4.1: Update the scaffold design docs

**Files:**
- Modify: `docs/design/scaffolds/web.md`
- Modify: `docs/design/scaffolds/node-cli.md` (if present; else skip and note)

- [ ] **Step 1: Confirm the docs exist + read them**

Run: `ls docs/design/scaffolds/ && sed -n '1,60p' docs/design/scaffolds/web.md`
Expected: see the current mise text. (If `node-cli.md` is absent — node-cli is a future stack — note that and skip it; update only what exists.)

- [ ] **Step 2: Rewrite the mise sections to the current API**

In each doc's mise/monorepo section, replace any stale claims with:
- A SpecKit repo is a **mise monorepo** from the first `target add`: a generated root `mise.toml` with `monorepo_root = true` and a **required** `[monorepo].config_roots` (filesystem auto-discovery is deprecated and warns).
- Members are invoked with the **target-path** form `mise //apps/<name>:test`, not `cd … && mise run test`.
- Toolchains hoist to the root `[tools]` from member #1; shared task bodies hoist to root `[task_templates]` at member #2 (inline before that).
- Cross-link `docs/design/mise-monorepo.md` and this plan.

- [ ] **Step 3: Commit**

```bash
git add docs/design/scaffolds/web.md docs/design/scaffolds/node-cli.md
git commit -m "docs: scaffold docs to current mise monorepo API (config_roots, //dir:task, templates)"
```

## Task 4.2: Add the project-memory topic

**Files:**
- Create: `.claude/memory/mise-monorepo.md`
- Modify: `.claude/memory/MEMORY.md`

- [ ] **Step 1: Follow the managing-memory skill**

Invoke the `managing-memory` skill (per `CLAUDE.md`) and write a topic capturing the non-obvious, durable knowledge:
- The invariant: a SpecKit repo is a mise monorepo from member #1; tools hoist at #1, task templates at #2 (promotion).
- The `go-toml/v2` `unstable` byte-splice approach (comment-preserving; `Table` nodes have zero `Raw` — derive header spans from the `Key` child); the version pin (v2.3.1) and why it's `unstable`.
- The drift coupling: family `monorepo/<f>.toml` bodies must match member inline bodies (enforced by `TestNodeFamilyMatchesWebInline` etc.).
- `tuist` is apple-specific (member-level `[tools]`, not in the swift family).

- [ ] **Step 2: Add the one-line index pointer to `MEMORY.md`**

```markdown
- [Mise monorepo](mise-monorepo.md) — generated root config invariant; the unstable-parser comment-preserving merge; the family↔member drift coupling
```

- [ ] **Step 3: Commit**

```bash
git add .claude/memory/mise-monorepo.md .claude/memory/MEMORY.md
git commit -m "memory: mise monorepo invariant + unstable-parser merge gotcha"
```

## Task 4.3: Final gate + README/docs sweep

- [ ] **Step 1: Update README if it documents target wiring**

Run: `grep -n "mise run test\|cd .*mise\|target add" README.md docs/*.md 2>/dev/null`
Expected: review hits; update any that show the old `cd … && mise run test` invocation to `mise //dir:test`. (Per the `keep-docs-updated` memory.)

- [ ] **Step 2: Full gate**

Run: `mise run ci`
Expected: PASS.

- [ ] **Step 3: Commit any doc fixes**

```bash
git add -A
git commit -m "docs: sweep target invocation to native mise //dir:task form"
```

---

## Self-review (run before handing off)

**Spec coverage** — every spec section maps to a task:
- Decisions 1 (root from member #1) → Task 1.4/1.8; 2 (memberDir globs) → `globsFor` (1.4); 3 (tools always, templates lazy) → 1.4 (`Hoist`) + 1.8 (count≥2); 4 (native command) → 1.6/2.3/3.1; 5 (real TOML lib, comment-preserving) → 1.2–1.4.
- Family model → 1.3; family files → 1.6/2.1/3.1.
- Inline-first / promotion → 1.5 + 1.8; promotion safety (canonical-body) → 1.5 tests.
- Engine verification command change → 1.6/2.3/3.1; comment refresh → 1.9.
- Trust (root propagates) → covered by the scaffold's existing phase-0 `mise trust` at root once the root config exists; **note:** the per-member `mise trust` in scaffold scripts is now redundant — verify whether any member scaffold runs `mise trust` in `files/`/scripts and, if so, leave it (harmless) or remove in a follow-up; flagged, not silently changed.
- Merge engine ops (create/monorepo_root/config_roots/tools/templates/promotion) → all in 1.4/1.5, each with a unit test.
- Testing section → unit (1.2–1.5), drift (1.7/2.3/3.2), e2e (1.9/2.4/3.2).

**Placeholder scan:** none — every code step carries complete, runnable code (verified against v2.3.1).

**Type consistency:** `Family{Name, Tools []ToolPin, Templates map[string]Template, Raw string, Hoist bool}`; `Template{Run, Description}`; `ToolPin{Key, Val}`; `expr{kind exprKind, span span, name, val string}`; functions `parseExprs`, `sectionEnd`, `splice`, `substituteVars`, `LoadFamily`, `EnsureRootMise`, `PromoteMember`, `memberVars`, `globsFor` — names used consistently across all tasks. `wireMonorepo`/`familyAndDir`/`renderMember`/`inlineRun` consistent across CLI tasks.

**Open coupling to watch during execution:** the family `monorepo/<f>.toml` `run` strings must stay byte-identical to the member scaffolds' inline `run` strings (after vars). The drift tests (1.7, 2.3, 3.2) fail loudly if they diverge — treat a drift failure as "reconcile the two files," never as "loosen the test."
