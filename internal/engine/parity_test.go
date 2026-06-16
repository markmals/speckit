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

// SPEC: story.engine.parity (conforming, declared-deviation, suspect-lying-marker, drifted, missing, independent-axes)
func TestParityFiveStates(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "features/0001-demo/stories/demo.x.md", paritySpec)
	writeSpecFile(t, root, "web/test/x.test.ts", parityTests)
	writeSpecFile(t, root, "web/src/x.ts", parityImpl)
	writeSpecFile(t, root, "web/report.junit.xml", parityReport)

	report, err := Parity(root, "web", VerifyConfig{Format: "junit", Report: "web/report.junit.xml", Source: []string{"web"}})
	if err != nil {
		t.Fatal(err)
	}
	got := map[specmodel.SpecID]string{}
	for _, c := range report.Cells {
		got[c.Scenario] = c.State
	}
	want := map[specmodel.SpecID]string{
		"scenario.demo.x.a": "conforming",
		"scenario.demo.x.b": "declared-deviation",
		"scenario.demo.x.c": "suspect", // marker over a FAILING test (D11)
		"scenario.demo.x.d": "drifted",
		"scenario.demo.x.e": "missing",
	}
	for s, w := range want {
		if got[s] != w {
			t.Errorf("%s: got %q, want %q", s, got[s], w)
		}
	}
	if !report.Gated() {
		t.Error("a matrix with non-conforming cells must fail --gate")
	}
}
