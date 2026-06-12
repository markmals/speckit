package reports

import (
	"os"
	"strings"
	"testing"
)

// SPEC: story.engine.verify (scenario.engine.verify.normalizes-reports)
func TestParseJUnitVitest(t *testing.T) {
	data, err := os.ReadFile("testdata/vitest.junit.xml")
	if err != nil {
		t.Fatal(err)
	}
	results, err := ParseJUnit(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 4 {
		t.Fatalf("expected 4 testcases, got %d", len(results))
	}
	for _, r := range results {
		if !r.Pass {
			t.Errorf("expected all vitest cases to pass; %q failed", r.Name)
		}
	}
	if !anyNameContains(results, "[scenario.todo.toggle.complete]") {
		t.Error("expected the scenario tag to survive in the junit name (web is report-carried)")
	}
}

// SPEC: story.engine.verify (scenario.engine.verify.normalizes-reports, source-bound-join)
func TestParseSwiftEvents(t *testing.T) {
	data, err := os.ReadFile("testdata/swift.events.ndjson")
	if err != nil {
		t.Fatal(err)
	}
	results, err := ParseSwiftEvents(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 tests, got %d", len(results))
	}
	pass := map[string]bool{}
	for _, r := range results {
		pass[r.Name] = r.Pass
	}
	// guard-empty is rigged to fail; the other two pass. Identity is the raw-id
	// display name (the scenario binding is read from source, not the report).
	if pass["toggling an empty-label todo is rejected"] {
		t.Error("the guard-empty test should have failed")
	}
	if !pass["toggling an active todo completes it"] || !pass["toggling a completed todo reactivates it"] {
		t.Error("the complete and reactivate tests should have passed")
	}
}

func anyNameContains(rs []Result, sub string) bool {
	for _, r := range rs {
		if strings.Contains(r.Name, sub) {
			return true
		}
	}
	return false
}
