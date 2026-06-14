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

const goDemoSpec = `---
id: story.demo.cli
kind: story
---

# Story: demo cli

## Acceptance Criteria

### Scenario 1: convert

<!-- id: scenario.demo.cli.convert -->

- Given a file
`

// A Go test suite that mixes a scenario-bound test (leading // [scenario] comment)
// with a plain untagged unit test, plus the go test -json report both produce.
const goDemoSource = "// [scenario.demo.cli.convert]\nfunc TestConvert(t *testing.T) {}\n\nfunc TestHelper(t *testing.T) {}\n"
const goDemoReport = "{\"Action\":\"pass\",\"Package\":\"x\",\"Test\":\"TestConvert\"}\n{\"Action\":\"pass\",\"Package\":\"x\",\"Test\":\"TestHelper\"}\n"

func setupGoVerifyProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeSpecFile(t, root, "features/0001-cli/stories/demo.cli.md", goDemoSpec)
	writeSpecFile(t, root, "cmd/x/x_test.go", goDemoSource)
	writeSpecFile(t, root, "cmd/x/report.gotest.json", goDemoReport)
	return root
}

// scoped bindings + the gotest format + a nested ### Scenario + a Go leading-comment
// binding, end to end: the untagged TestHelper is out of scope (not an unbound
// violation), so the suite still verifies and locks the scenario it does bind.
//
// SPEC: story.engine.verify (scenario.engine.verify.source-bound-join)
func TestVerifyGoScopedBindings(t *testing.T) {
	root := setupGoVerifyProject(t)
	cfg := VerifyConfig{Format: "gotest", Report: "cmd/x/report.gotest.json", Source: "cmd/x", Bindings: "scoped"}
	v, locked, err := Verify(root, "go", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Unbound) != 0 {
		t.Errorf("scoped mode must drop unbound (untagged) tests, got %+v", v.Unbound)
	}
	if !v.Green() {
		t.Fatalf("expected green under scoped bindings: %+v", v)
	}
	if len(locked) != 1 || locked[0] != "story.demo.cli" {
		t.Fatalf("expected story.demo.cli locked, got %v", locked)
	}
}

// strict (the default) flags the same untagged test as an unbound D12 violation —
// proving scoped is an opt-in relaxation, not a change to the default contract.
func TestVerifyStrictFlagsUntaggedTest(t *testing.T) {
	root := setupGoVerifyProject(t)
	cfg := VerifyConfig{Format: "gotest", Report: "cmd/x/report.gotest.json", Source: "cmd/x"} // Bindings "" = strict
	v, locked, err := Verify(root, "go", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Unbound) == 0 {
		t.Error("strict mode must flag the untagged TestHelper as unbound")
	}
	if v.Green() || len(locked) != 0 {
		t.Errorf("an unbound test must block green + lock in strict mode: green=%v locked=%v", v.Green(), locked)
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
