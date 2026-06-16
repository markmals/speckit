package engine

import "testing"

const multiSpec = `---
id: story.demo.multi
kind: story
---

# Story: demo multi

## Acceptance Criteria

### Scenario 1: alpha

<!-- id: scenario.demo.multi.alpha -->

- Given alpha

### Scenario 2: beta

<!-- id: scenario.demo.multi.beta -->

- Given beta
`

// One scenario is bound by a test under cmd/example, the other under
// internal/example — the two-source shape this feature exists for.
const multiAlphaSrc = "// [scenario.demo.multi.alpha]\nfunc TestAlpha(t *testing.T) {}\n"
const multiBetaSrc = "// [scenario.demo.multi.beta]\nfunc TestBeta(t *testing.T) {}\n"
const multiReport = "{\"Action\":\"pass\",\"Package\":\"cmd/example\",\"Test\":\"TestAlpha\"}\n" +
	"{\"Action\":\"pass\",\"Package\":\"internal/example\",\"Test\":\"TestBeta\"}\n"

func setupMultiSourceProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeSpecFile(t, root, "features/0001-multi/stories/demo.multi.md", multiSpec)
	writeSpecFile(t, root, "cmd/example/example_test.go", multiAlphaSrc)
	writeSpecFile(t, root, "internal/example/example_test.go", multiBetaSrc)
	writeSpecFile(t, root, "report.gotest.json", multiReport)
	return root
}

// Verify joins bindings from two separate source roots into one target: both
// scenarios pass and the spec locks.
func TestVerifyJoinsMultipleSourceRoots(t *testing.T) {
	root := setupMultiSourceProject(t)
	cfg := VerifyConfig{Format: "gotest", Report: "report.gotest.json", Source: []string{"cmd/example", "internal/example"}}

	v, locked, err := Verify(root, "go-service", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Green() {
		t.Fatalf("expected green across both roots: %+v", v)
	}
	if len(locked) != 1 || locked[0] != "story.demo.multi" {
		t.Fatalf("expected story.demo.multi locked, got %v", locked)
	}
}

// A dangling binding (to an undeclared scenario) from EITHER root still fails.
func TestVerifyMultiSourceDanglingFromAnyRoot(t *testing.T) {
	root := setupMultiSourceProject(t)
	// a second test in internal/example binds a scenario the spec never declares
	writeSpecFile(t, root, "internal/example/extra_test.go", "// [scenario.demo.multi.ghost]\nfunc TestGhost(t *testing.T) {}\n")

	cfg := VerifyConfig{Format: "gotest", Report: "report.gotest.json", Source: []string{"cmd/example", "internal/example"}}
	v, locked, err := Verify(root, "go-service", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Dangling) == 0 {
		t.Error("a dangling binding from a second root must be detected")
	}
	if v.Green() || len(locked) != 0 {
		t.Errorf("a dangling binding must block green + lock: green=%v locked=%v", v.Green(), locked)
	}
}

// scoped bindings drop an untagged test in one root while still joining the
// bound tests across both roots.
func TestVerifyMultiSourceScopedDropsUntagged(t *testing.T) {
	root := setupMultiSourceProject(t)
	// an untagged unit test alongside the bound one in cmd/example
	writeSpecFile(t, root, "cmd/example/unit_test.go", "func TestHelper(t *testing.T) {}\n")
	report := multiReport + "{\"Action\":\"pass\",\"Package\":\"cmd/example\",\"Test\":\"TestHelper\"}\n"
	writeSpecFile(t, root, "report.gotest.json", report)

	cfg := VerifyConfig{Format: "gotest", Report: "report.gotest.json", Source: []string{"cmd/example", "internal/example"}, Bindings: "scoped"}
	v, locked, err := Verify(root, "go-service", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Unbound) != 0 {
		t.Errorf("scoped must drop the untagged TestHelper, got %+v", v.Unbound)
	}
	if !v.Green() || len(locked) != 1 {
		t.Fatalf("expected green + lock under scoped multi-source: green=%v locked=%v", v.Green(), locked)
	}
}
