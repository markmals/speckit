package engine

import "testing"

const firewallSpec = `---
id: story.x.y
kind: story
---

# Acceptance Criteria

## Scenario 1: a

<!-- id: scenario.x.y.a -->

- Given a
`

// SPEC: story.engine.gate (scenario.engine.gate.test-edit-firewall)
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

	// test changed AND its spec changed -> clean
	f, _ = TestEditFirewall(root, []string{"web/test/x.test.ts", "features/0001-x/stories/x.y.md"})
	if len(f) != 0 {
		t.Errorf("changing the spec alongside the test must clear the firewall, got %v", f)
	}
}

// SPEC: story.engine.gate (scenario.engine.gate.generated-block)
func TestGateGeneratedBlock(t *testing.T) {
	f := GeneratedBlock([]string{".speckit/lock/web/domain.item.json", "src/app.ts", "web/convex/_generated/api.ts"})
	if len(f) != 2 {
		t.Fatalf("expected two generated-block findings, got %v", f)
	}
}

// SPEC: story.engine.gate (scenario.engine.gate.scoped-commit)
func TestGateScopedCommit(t *testing.T) {
	scopes := map[string]bool{"engine": true, "specs": true, "web": true, "treewide": true}
	for _, s := range []string{"engine: add the gate", "specs: tighten X", "web, engine: cross-area", "web (PROJ-9): fix"} {
		if f := ScopedCommit(s, scopes); len(f) != 0 {
			t.Errorf("%q should pass, got %v", s, f)
		}
	}
	for _, s := range []string{"add the gate", "nope: bad scope", "feat(web): conventional"} {
		if f := ScopedCommit(s, scopes); len(f) == 0 {
			t.Errorf("%q should be rejected", s)
		}
	}
}

func TestDefinedScopes(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "specs/models/item.md", "---\nid: domain.item\nkind: domain\n---\n# x\n")
	writeSpecFile(t, root, ".claude/commit-scopes", "engine   # internal/engine\ncli\n# a comment line\n")
	scopes, err := DefinedScopes(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"domain.item", "engine", "cli", "specs", "treewide", "docs"} {
		if !scopes[want] {
			t.Errorf("expected scope %q to be defined", want)
		}
	}
	if scopes["nope"] {
		t.Error("an undeclared scope must not be defined")
	}
}
