package engine

import (
	"strings"
	"testing"
)

const firewallSpec = `---
id: story.x.y
kind: story
---

# Acceptance Criteria

## Scenario 1: a

<!-- id: scenario.x.y.a -->

- Given a
`

// A commit that changes a scenario-tagged test without its owning spec fails
// the firewall, and the finding names both the test path and the spec it
// should have accompanied; changing the spec alongside clears it.
//
// [scenario.engine.gate.test-edit-firewall]
func TestGateFirewall(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "features/0001-x/stories/x.y.md", firewallSpec)
	writeSpecFile(t, root, "web/test/x.test.ts", `it("[scenario.x.y.a] a", () => {})`+"\n")

	// test changed, spec NOT changed -> violation
	f, err := TestEditFirewall(root, []string{"web/test/x.test.ts"})
	if err != nil {
		t.Fatal(err)
	}
	if len(f) != 1 {
		t.Fatalf("expected one firewall finding, got %v", f)
	}
	if f[0].Check != "test-edit-firewall" {
		t.Errorf("finding check: got %q", f[0].Check)
	}
	if f[0].Path != "web/test/x.test.ts" {
		t.Errorf("the finding must name the test path, got %q", f[0].Path)
	}
	if !strings.Contains(f[0].Message, "features/0001-x/stories/x.y.md") {
		t.Errorf("the finding must name the spec the test should have accompanied, got %q", f[0].Message)
	}

	// test changed AND its spec changed -> clean
	f, _ = TestEditFirewall(root, []string{"web/test/x.test.ts", "features/0001-x/stories/x.y.md"})
	if len(f) != 0 {
		t.Errorf("changing the spec alongside the test must clear the firewall, got %v", f)
	}
}

// Edits to engine-generated paths — the lock under .speckit/lock/ (which only
// `specify lock` may write) and codegen output — are refused, each finding
// naming the generated path; ordinary source paths pass.
//
// [scenario.engine.gate.generated-block]
// [scenario.engine.lock.generated-guard]
func TestGateGeneratedBlock(t *testing.T) {
	f := GeneratedBlock([]string{".speckit/lock/app/domain.item.json", "src/app.ts", "app/gen/_generated/api.ts"})
	if len(f) != 2 {
		t.Fatalf("expected two generated-block findings, got %v", f)
	}
	flagged := map[string]bool{}
	for _, finding := range f {
		if finding.Check != "generated-block" {
			t.Errorf("finding check: got %q", finding.Check)
		}
		if finding.Path == "" {
			t.Errorf("a finding must name the generated path, got %+v", finding)
		}
		flagged[finding.Path] = true
	}
	if !flagged[".speckit/lock/app/domain.item.json"] {
		t.Error("a hand-edit under .speckit/lock/ must be refused by name — the lock is engine-owned")
	}
	if !flagged["app/gen/_generated/api.ts"] {
		t.Error("codegen output must be refused by name")
	}
	if flagged["src/app.ts"] {
		t.Error("an ordinary source path must not be blocked")
	}
}

// A commit subject whose scope is not a defined scope is rejected with a
// message that explains the scope rule; defined scopes pass.
//
// [scenario.engine.gate.scoped-commit]
func TestGateScopedCommit(t *testing.T) {
	scopes := map[string]bool{"engine": true, "specs": true, "web": true, "treewide": true}
	for _, s := range []string{"engine: add the gate", "specs: tighten X", "web, engine: cross-area", "web (PROJ-9): fix"} {
		if f := ScopedCommit(s, scopes); len(f) != 0 {
			t.Errorf("%q should pass, got %v", s, f)
		}
	}
	rejected := []struct{ subject, explains string }{
		{"add the gate", "<scope>: <description>"},
		{"nope: bad scope", `undefined commit scope "nope"`},
		{"feat(web): conventional", "<scope>: <description>"},
	}
	for _, r := range rejected {
		f := ScopedCommit(r.subject, scopes)
		if len(f) == 0 {
			t.Errorf("%q should be rejected", r.subject)
			continue
		}
		if f[0].Check != "scoped-commit" {
			t.Errorf("%q: finding check got %q", r.subject, f[0].Check)
		}
		if !strings.Contains(f[0].Message, r.explains) {
			t.Errorf("%q: the rejection must explain the scope rule, got %q", r.subject, f[0].Message)
		}
	}
}

// The defined-scope set is exactly what the scenario states: every spec id, each
// features/<slug> dir, the fixed harness areas, `specs`, `treewide`, and the
// scopes declared in .claude/commit-scopes — and nothing undeclared.
//
// [scenario.engine.gate.scoped-commit]
func TestDefinedScopes(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "specs/models/item.md", "---\nid: domain.item\nkind: domain\n---\n# x\n")
	writeSpecFile(t, root, "features/0001-x/stories/x.y.md", firewallSpec)
	writeSpecFile(t, root, ".claude/commit-scopes", "engine   # internal/engine\ncli\n# a comment line\n")
	scopes, err := DefinedScopes(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"domain.item",     // a spec id
		"story.x.y",       // a feature-owned spec id
		"features/0001-x", // a features/<slug> dir
		"engine", "cli",   // declared in .claude/commit-scopes
		"specs", "treewide", "docs", // fixed areas
	} {
		if !scopes[want] {
			t.Errorf("expected scope %q to be defined", want)
		}
	}
	if scopes["nope"] {
		t.Error("an undeclared scope must not be defined")
	}
}
