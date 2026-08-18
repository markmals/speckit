package engine

import (
	"testing"

	"github.com/markmals/speckit/internal/specmodel"
)

const paritySpec = `---
id: story.demo.x
kind: story
---

# Story

# Acceptance Criteria

## Scenario a

<!-- id: scenario.demo.x.a -->

- Given a

## Scenario b

<!-- id: scenario.demo.x.b -->

- Given b

## Scenario c

<!-- id: scenario.demo.x.c -->

- Given c

## Scenario d

<!-- id: scenario.demo.x.d -->

- Given d

## Scenario e

<!-- id: scenario.demo.x.e -->

- Given e
`

// a,b,c,d are tested (e has no test -> missing).
const parityTests = `it("[scenario.demo.x.a] a", () => {})
it("[scenario.demo.x.b] b", () => {})
it("[scenario.demo.x.c] c", () => {})
it("[scenario.demo.x.d] d", () => {})
`

// b deviates honestly (its test passes); c carries a LYING marker (its test fails).
const parityImpl = `// SPEC: scenario.demo.x.b (deviates: web uses a button)
// SPEC: scenario.demo.x.c (deviates: web validates inline)
`

// a passes, b passes, c fails, d fails.
const parityReport = `<testsuites><testsuite name="x">
<testcase classname="x" name="[scenario.demo.x.a] a"/>
<testcase classname="x" name="[scenario.demo.x.b] b"/>
<testcase classname="x" name="[scenario.demo.x.c] c"><failure message="x"/></testcase>
<testcase classname="x" name="[scenario.demo.x.d] d"><failure message="x"/></testcase>
</testsuite></testsuites>`

// parityBMatrix builds the shared five-state fixture — every (marker × test
// outcome) combination plus an untested scenario — and runs Parity over it,
// returning the report and a scenario -> cell index.
func parityBMatrix(t *testing.T) (ParityReport, map[specmodel.SpecID]ParityCell) {
	t.Helper()
	root := t.TempDir()
	writeSpecFile(t, root, "features/0001-demo/stories/demo.x.md", paritySpec)
	writeSpecFile(t, root, "web/test/x.test.ts", parityTests)
	writeSpecFile(t, root, "web/src/x.ts", parityImpl)
	writeSpecFile(t, root, "web/report.junit.xml", parityReport)

	report, err := Parity(root, "web", VerifyConfig{Format: "junit", Report: "web/report.junit.xml", Source: []string{"web"}})
	if err != nil {
		t.Fatal(err)
	}
	cells := map[specmodel.SpecID]ParityCell{}
	for _, c := range report.Cells {
		cells[c.Scenario] = c
	}
	return report, cells
}

// A passing joined test with no deviation marker is conforming.
//
// [scenario.engine.parity.conforming]
func TestParityConforming(t *testing.T) {
	_, cells := parityBMatrix(t)
	if got := cells["scenario.demo.x.a"]; got.State != "conforming" {
		t.Errorf("passing + no marker: got %q, want conforming", got.State)
	}
}

// A passing joined test with a deviation marker is declared-deviation, shown
// with its reason — and treated as needing sign-off, never as green: a matrix
// whose only non-conforming cell is a declared-deviation still gates (D11).
//
// [scenario.engine.parity.declared-deviation]
func TestParityDeclaredDeviation(t *testing.T) {
	_, cells := parityBMatrix(t)
	got := cells["scenario.demo.x.b"]
	if got.State != "declared-deviation" {
		t.Errorf("passing + marker: got %q, want declared-deviation", got.State)
	}
	if got.Reason != "web uses a button" {
		t.Errorf("the cell must carry the declared reason, got %q", got.Reason)
	}

	// Never green (D11): a report whose sole deviation is honest still gates.
	honest := ParityReport{Target: "web", Cells: []ParityCell{
		{Scenario: "scenario.demo.x.a", State: "conforming"},
		{Scenario: "scenario.demo.x.b", State: "declared-deviation", Reason: "web uses a button"},
	}}
	if !honest.Gated() {
		t.Error("a declared-deviation needs sign-off — it must never pass --gate as green")
	}
}

// A deviation marker over a FAILING test is suspect, never declared-deviation
// — the marker cannot be machine-verified as intentional — and the matrix
// fails the --gate predicate.
//
// [scenario.engine.parity.suspect-lying-marker]
func TestParitySuspectLyingMarker(t *testing.T) {
	report, cells := parityBMatrix(t)
	got := cells["scenario.demo.x.c"]
	if got.State != "suspect" {
		t.Errorf("failing + marker: got %q, want suspect", got.State)
	}
	if got.State == "declared-deviation" {
		t.Error("a lying marker must never launder a failing test into declared-deviation")
	}
	if got.Reason == "" {
		t.Error("the suspect cell must surface the unverifiable marker's reason")
	}
	if !report.Gated() {
		t.Error("a matrix with a suspect cell must fail --gate (non-zero exit)")
	}
}

// A failing joined test with no marker is drifted.
//
// [scenario.engine.parity.drifted]
func TestParityDrifted(t *testing.T) {
	_, cells := parityBMatrix(t)
	if got := cells["scenario.demo.x.d"]; got.State != "drifted" {
		t.Errorf("failing + no marker: got %q, want drifted", got.State)
	}
}

// A scenario with no joined test at all is missing — distinct from drifted.
//
// [scenario.engine.parity.missing]
func TestParityMissing(t *testing.T) {
	_, cells := parityBMatrix(t)
	got := cells["scenario.demo.x.e"]
	if got.State != "missing" {
		t.Errorf("untested scenario: got %q, want missing", got.State)
	}
	if got.State == "drifted" {
		t.Error("missing must be distinct from drifted")
	}
}

// Marker presence and test outcome are independent axes, crossed: all four
// (marker × outcome) combinations classify distinctly, and a marker never
// overrides or suppresses a failing test result.
//
// [scenario.engine.parity.independent-axes]
func TestParityIndependentAxes(t *testing.T) {
	_, cells := parityBMatrix(t)
	cross := []struct {
		scenario specmodel.SpecID
		combo    string
		want     string
	}{
		{"scenario.demo.x.a", "pass × no marker", "conforming"},
		{"scenario.demo.x.b", "pass × marker", "declared-deviation"},
		{"scenario.demo.x.c", "fail × marker", "suspect"},
		{"scenario.demo.x.d", "fail × no marker", "drifted"},
	}
	for _, c := range cross {
		if got := cells[c.scenario].State; got != c.want {
			t.Errorf("%s: got %q, want %q", c.combo, got, c.want)
		}
	}
	if cells["scenario.demo.x.c"].State == "declared-deviation" {
		t.Error("the marker axis must never suppress the failing-test axis")
	}
}
