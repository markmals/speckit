package engine

import "testing"

const demoSpec = `---
id: story.demo.toggle
kind: story
---

# Story: demo

# Acceptance Criteria

## Scenario 1: a

<!-- id: scenario.demo.toggle.a -->

- Given a

## Scenario 2: b

<!-- id: scenario.demo.toggle.b -->

- Given b
`

const demoSource = `it("[scenario.demo.toggle.a] does a", () => {})
it("[scenario.demo.toggle.b] does b", () => {})
`

func junitReport(aPass, bPass bool) string {
	cell := func(name string, pass bool) string {
		if pass {
			return `<testcase classname="demo" name="` + name + `"/>`
		}
		return `<testcase classname="demo" name="` + name + `"><failure message="x"/></testcase>`
	}
	return `<testsuites><testsuite name="demo">` +
		cell("[scenario.demo.toggle.a] does a", aPass) +
		cell("[scenario.demo.toggle.b] does b", bPass) +
		`</testsuite></testsuites>`
}

func setupVerifyProject(t *testing.T, report string) string {
	t.Helper()
	root := t.TempDir()
	writeSpecFile(t, root, "features/0001-demo/stories/demo.toggle.md", demoSpec)
	writeSpecFile(t, root, "web/test/demo.test.ts", demoSource)
	writeSpecFile(t, root, "web/report.junit.xml", report)
	return root
}

// SPEC: story.engine.verify (scenario.engine.verify.green-writes-lock)
// SPEC: story.engine.lock (scenario.engine.lock.writes-on-green)
func TestVerifyGreenWritesLock(t *testing.T) {
	root := setupVerifyProject(t, junitReport(true, true))
	cfg := VerifyConfig{Format: "junit", Report: "web/report.junit.xml", Source: "web"}

	v, locked, err := Verify(root, "web", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Green() {
		t.Fatalf("expected green: %+v", v)
	}
	if len(locked) != 1 || locked[0] != "story.demo.toggle" {
		t.Fatalf("expected story.demo.toggle locked, got %v", locked)
	}
	// the lock is real: drift is clean immediately after
	d, err := Drift(root, "web")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(d.Clean, "story.demo.toggle") || d.HasDrift() {
		t.Errorf("expected clean drift after verify, got %+v", d)
	}
	// and a shard exists on disk
	if _, ok, _ := ReadShard(root, "web", "story.demo.toggle"); !ok {
		t.Error("expected a lock shard after green verify")
	}
}

// SPEC: story.engine.lock (scenario.engine.lock.no-write-on-red)
func TestVerifyRedWritesNoLock(t *testing.T) {
	root := setupVerifyProject(t, junitReport(true, false)) // scenario b fails
	cfg := VerifyConfig{Format: "junit", Report: "web/report.junit.xml", Source: "web"}

	v, locked, err := Verify(root, "web", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if v.Green() {
		t.Error("expected not green (b failed)")
	}
	if len(locked) != 0 {
		t.Errorf("a spec with a failing scenario must not be locked, got %v", locked)
	}
	if _, ok, _ := ReadShard(root, "web", "story.demo.toggle"); ok {
		t.Error("no shard should be written on red")
	}
}
