package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/markmals/speckit/internal/specmodel"
)

// GateFinding is a single enforcement-gate violation (D8).
type GateFinding struct {
	Check   string `json:"check"`
	Path    string `json:"path,omitempty"`
	Line    int    `json:"line,omitempty"` // 1-based line, for CI annotations
	Message string `json:"message"`
}

// TestEditFirewall flags changed scenario-tagged test files whose owning spec
// file did not also change — preventing a test from being quietly altered to
// pass without touching the spec it pins. `changed` are repo-relative paths.
//
// SPEC: story.engine.gate (scenario.engine.gate.test-edit-firewall)
func TestEditFirewall(root string, changed []string) ([]GateFinding, error) {
	specs, err := specmodel.LoadLibrary(os.DirFS(root))
	if err != nil {
		return nil, err
	}
	scenarioSpecPath := map[specmodel.SpecID]string{}
	for _, s := range specs {
		for _, sc := range s.Scenarios {
			if sc.SubID != "" {
				scenarioSpecPath[specmodel.SpecID(sc.SubID)] = s.Path
			}
		}
	}
	changedSet := map[string]bool{}
	for _, p := range changed {
		changedSet[filepath.ToSlash(p)] = true
	}

	var findings []GateFinding
	seen := map[string]bool{}
	for _, p := range changed {
		content, err := os.ReadFile(filepath.Join(root, p))
		if err != nil {
			continue // deleted or unreadable
		}
		for _, b := range bindingsInContent(p, string(content)) {
			specPath, ok := scenarioSpecPath[b.Scenario]
			if !ok || changedSet[specPath] {
				continue // dangling ref is a scan/verify concern; or the spec did change
			}
			key := p + "→" + specPath
			if seen[key] {
				continue
			}
			seen[key] = true
			findings = append(findings, GateFinding{
				Check:   "test-edit-firewall",
				Path:    filepath.ToSlash(p),
				Line:    b.Line,
				Message: fmt.Sprintf("test changed but its spec %s (scenario %s) did not", specPath, b.Scenario),
			})
		}
	}
	return findings, nil
}

// generatedPrefixes are the engine-owned paths edits are blocked on (L4).
var generatedPrefixes = []string{".speckit/lock/", ".speckit/ledger"}

// GeneratedBlock flags changes to engine-generated paths.
//
// SPEC: story.engine.gate (scenario.engine.gate.generated-block)
func GeneratedBlock(changed []string) []GateFinding {
	var findings []GateFinding
	for _, p := range changed {
		s := filepath.ToSlash(p)
		switch {
		case hasAnyPrefix(s, generatedPrefixes):
			findings = append(findings, GateFinding{Check: "generated-block", Path: s, Message: "edit to a generated path (L4)"})
		case strings.Contains(s, "/_generated/"):
			findings = append(findings, GateFinding{Check: "generated-block", Path: s, Message: "edit to generated codegen output"})
		}
	}
	return findings
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// ScopedCommit validates a commit subject against the defined scopes (D8): the
// form is "<scope>: <description>" where scope is a defined scope (or a
// comma-separated list of them); the Conventional "type(scope):" form is
// rejected, while a trailing "(TICKET)" qualifier is allowed.
//
// SPEC: story.engine.gate (scenario.engine.gate.scoped-commit)
func ScopedCommit(subject string, scopes map[string]bool) []GateFinding {
	subject = strings.TrimSpace(subject)
	colon := strings.Index(subject, ":")
	if colon < 0 {
		return []GateFinding{{Check: "scoped-commit", Message: "subject is not in <scope>: <description> form"}}
	}
	scopePart := strings.TrimSpace(subject[:colon])
	if i := strings.Index(scopePart, "("); i >= 0 {
		// "feat(web)" (word immediately followed by paren) is Conventional Commits;
		// "web (PROJ-12)" (space before paren) is an allowed ticket qualifier.
		if i > 0 && !strings.ContainsAny(scopePart[:i], " ,") {
			return []GateFinding{{Check: "scoped-commit",
				Message: fmt.Sprintf("%q looks like Conventional Commits — use <scope>: <description>", scopePart)}}
		}
		scopePart = strings.TrimSpace(scopePart[:i])
	}
	for _, s := range strings.Split(scopePart, ",") {
		if s = strings.TrimSpace(s); s == "" || !scopes[s] {
			return []GateFinding{{Check: "scoped-commit", Message: fmt.Sprintf("undefined commit scope %q", s)}}
		}
	}
	return nil
}

// DefinedScopes computes the valid commit scopes (D8): every spec/feature id,
// each features/<slug>, the harness areas, `specs`, `treewide`, and the scopes
// declared in .claude/commit-scopes.
func DefinedScopes(root string) (map[string]bool, error) {
	scopes := map[string]bool{"specs": true, "treewide": true}
	for _, a := range []string{"hooks", "skills", "commands", "agents", "templates", "rules", "docs", "mise", "readme"} {
		scopes[a] = true
	}
	specs, err := specmodel.LoadLibrary(os.DirFS(root))
	if err != nil {
		return nil, err
	}
	for _, s := range specs {
		scopes[string(s.ID)] = true
	}
	if entries, err := os.ReadDir(filepath.Join(root, "features")); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				scopes["features/"+e.Name()] = true
			}
		}
	}
	if data, err := os.ReadFile(filepath.Join(root, ".claude", "commit-scopes")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if i := strings.Index(line, "#"); i >= 0 {
				line = line[:i]
			}
			if line = strings.TrimSpace(line); line != "" {
				scopes[line] = true
			}
		}
	}
	return scopes, nil
}
