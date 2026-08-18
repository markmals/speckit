package engine

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// gateBRunAll invokes every gate subcheck once over the same explicit inputs
// and returns their findings in a fixed order, so two runs can be compared.
func gateBRunAll(t *testing.T, root string, changed []string, subject string, scopes map[string]bool) [][]GateFinding {
	t.Helper()
	firewall, err := TestEditFirewall(root, changed)
	if err != nil {
		t.Fatal(err)
	}
	return [][]GateFinding{firewall, GeneratedBlock(changed), ScopedCommit(subject, scopes)}
}

// gateBStripAgentEnv removes every agent-ish variable from the process
// environment for the duration of the test (t.Setenv registers restoration).
func gateBStripAgentEnv(t *testing.T) {
	t.Helper()
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		agentish := k == "CI" || k == "GITHUB_ACTIONS"
		for _, prefix := range []string{"CLAUDE", "ANTHROPIC", "CURSOR", "COPILOT", "CODEX"} {
			agentish = agentish || strings.HasPrefix(k, prefix)
		}
		if agentish {
			t.Setenv(k, os.Getenv(k)) // register restore
			os.Unsetenv(k)
		}
	}
}

// Each gate subcheck is a pure function of its explicit inputs (a root, changed
// paths, a commit subject, a scope set): invoked with agent environment
// variables present and again with every agent-ish variable stripped, in a repo
// containing no agent directory at all, the findings are identical — so the
// check depends on no agent runtime (D8) and behaves the same from a pre-commit
// hook, a CI job, or a bare shell.
//
// [scenario.engine.gate.agent-agnostic]
func TestGateAgentAgnostic(t *testing.T) {
	// The fixture root deliberately contains no .claude/, .cursor/, .github/, or
	// any other agent directory — only the spec library and a tagged test.
	root := t.TempDir()
	writeSpecFile(t, root, "features/0001-x/stories/x.y.md", firewallSpec)
	writeSpecFile(t, root, "web/test/x.test.ts", `it("[scenario.x.y.a] a", () => {})`+"\n")

	changed := []string{"web/test/x.test.ts", ".speckit/lock/web/domain.item.json"}
	subject := "nope: undefined scope"
	scopes := map[string]bool{"engine": true}

	// Pass 1: an agent-saturated environment (Claude, CI, Cursor all "present").
	for _, kv := range [][2]string{
		{"CLAUDECODE", "1"}, {"CLAUDE_CODE_ENTRYPOINT", "cli"}, {"ANTHROPIC_API_KEY", "sk-test"},
		{"CURSOR_TRACE_ID", "x"}, {"GITHUB_ACTIONS", "true"}, {"CI", "true"},
	} {
		t.Setenv(kv[0], kv[1])
	}
	withAgent := gateBRunAll(t, root, changed, subject, scopes)

	// Pass 2: no agent-ish variable at all.
	gateBStripAgentEnv(t)
	withoutAgent := gateBRunAll(t, root, changed, subject, scopes)

	if !reflect.DeepEqual(withAgent, withoutAgent) {
		t.Fatalf("subcheck findings must not depend on the agent environment:\nwith agent: %+v\nwithout:    %+v", withAgent, withoutAgent)
	}
	// The comparison must not be vacuous: each subcheck found its violation.
	for i, name := range []string{"test-edit-firewall", "generated-block", "scoped-commit"} {
		if len(withoutAgent[i]) != 1 || withoutAgent[i][0].Check != name {
			t.Errorf("%s: expected one finding with no agent present, got %+v", name, withoutAgent[i])
		}
	}
}

// Each subcheck is independently invocable, and a violation visible to one is
// reported by that check alone — so hooks and CI can compose them à la carte.
//
// [scenario.engine.gate.subchecks]
func TestGateSubchecksIndependent(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "features/0001-x/stories/x.y.md", firewallSpec)
	writeSpecFile(t, root, "web/test/x.test.ts", `it("[scenario.x.y.a] a", () => {})`+"\n")
	scopes := map[string]bool{"engine": true}

	// A generated-path edit is a generated-block finding and nothing else.
	generatedOnly := []string{".speckit/lock/web/domain.item.json"}
	if f := GeneratedBlock(generatedOnly); len(f) != 1 || f[0].Check != "generated-block" {
		t.Errorf("generated-block must flag its own violation, got %+v", f)
	}
	if f, err := TestEditFirewall(root, generatedOnly); err != nil || len(f) != 0 {
		t.Errorf("the firewall must not report a generated-path edit, got %+v (err %v)", f, err)
	}
	if f := ScopedCommit("engine: fine", scopes); len(f) != 0 {
		t.Errorf("scoped-commit must not report a generated-path edit, got %+v", f)
	}

	// An untethered test edit is a firewall finding and nothing else.
	firewallOnly := []string{"web/test/x.test.ts"}
	if f, err := TestEditFirewall(root, firewallOnly); err != nil || len(f) != 1 || f[0].Check != "test-edit-firewall" {
		t.Errorf("the firewall must flag its own violation, got %+v (err %v)", f, err)
	}
	if f := GeneratedBlock(firewallOnly); len(f) != 0 {
		t.Errorf("generated-block must not report a test edit, got %+v", f)
	}

	// A bad commit subject is a scoped-commit finding and nothing else.
	if f := ScopedCommit("nope: bad", scopes); len(f) != 1 || f[0].Check != "scoped-commit" {
		t.Errorf("scoped-commit must flag its own violation, got %+v", f)
	}
	clean := []string{"src/app.ts"}
	if f := GeneratedBlock(clean); len(f) != 0 {
		t.Errorf("generated-block must stay quiet on a bad subject, got %+v", f)
	}
	if f, err := TestEditFirewall(root, clean); err != nil || len(f) != 0 {
		t.Errorf("the firewall must stay quiet on a bad subject, got %+v (err %v)", f, err)
	}
}
