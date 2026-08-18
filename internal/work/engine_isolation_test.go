// The behavioral half of the work/engine firewall: work items are ephemeral
// coordination the engine never reads. The import direction is proven by the
// structural firewall test (scenario.work.providers.import-firewall); here
// the engine's own entry points run over the same project with work state
// present, changing, and absent — and their outputs never move.
package work_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/config"
	"github.com/markmals/speckit/internal/engine"
	"github.com/markmals/speckit/internal/specmodel"
	"github.com/markmals/speckit/internal/work"
	"github.com/markmals/speckit/internal/work/markdown"
)

const isolationSpec = `---
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

// danglingSpec depends on a spec that does not exist, so Scan always has a
// non-empty finding list — comparing real findings, not empty vs empty.
const danglingSpec = `---
id: domain.demo.item
kind: domain
depends-on: [domain.missing]
---

# Item
`

const isolationSource = `it("[scenario.demo.toggle.a] does a", () => {})
it("[scenario.demo.toggle.b] does b", () => {})
`

const isolationReport = `<testsuites><testsuite name="demo">` +
	`<testcase classname="demo" name="[scenario.demo.toggle.a] does a"/>` +
	`<testcase classname="demo" name="[scenario.demo.toggle.b] does b"/>` +
	`</testsuite></testsuites>`

func writeIsolationFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// setupIsolationProject builds a complete verifiable project: a green spec,
// a spec with a dangling dependency, a bound test source, and a pre-built
// junit report.
func setupIsolationProject(t *testing.T) (string, engine.VerifyConfig) {
	t.Helper()
	root := t.TempDir()
	writeIsolationFile(t, root, "features/0001-demo/stories/demo.toggle.md", isolationSpec)
	writeIsolationFile(t, root, "specs/models/demo-item.md", danglingSpec)
	writeIsolationFile(t, root, "web/test/demo.test.ts", isolationSource)
	writeIsolationFile(t, root, "web/report.junit.xml", isolationReport)
	return root, engine.VerifyConfig{Format: "junit", Report: "web/report.junit.xml", Source: []string{"web"}}
}

// engineOutputs captures every engine entry point's result over one project
// state, so two captures can be compared wholesale.
type engineOutputs struct {
	Library  []specmodel.Spec
	Scan     []specmodel.Finding
	Verify   engine.VerifyResult
	Locked   []specmodel.SpecID
	Drift    engine.DriftReport
	Cover    engine.CoverReport
	Parity   engine.ParityReport
	Firewall []engine.GateFinding
	Block    []engine.GateFinding
}

func captureEngineOutputs(t *testing.T, root string, cfg engine.VerifyConfig) engineOutputs {
	t.Helper()
	var out engineOutputs
	var err error
	if out.Library, err = specmodel.LoadLibrary(os.DirFS(root)); err != nil {
		t.Fatal(err)
	}
	if out.Scan, err = engine.Scan(os.DirFS(root)); err != nil {
		t.Fatal(err)
	}
	if out.Verify, out.Locked, err = engine.Verify(root, "web", cfg); err != nil {
		t.Fatal(err)
	}
	if out.Drift, err = engine.Drift(root, "web"); err != nil {
		t.Fatal(err)
	}
	if out.Cover, err = engine.Cover(root, "story.demo.toggle"); err != nil {
		t.Fatal(err)
	}
	if out.Parity, err = engine.Parity(root, "web", cfg); err != nil {
		t.Fatal(err)
	}
	changed := []string{"web/test/demo.test.ts", "src/app.ts", ".speckit/lock/web/story.demo.toggle.json"}
	if out.Firewall, err = engine.TestEditFirewall(root, changed); err != nil {
		t.Fatal(err)
	}
	out.Block = engine.GeneratedBlock(changed)
	return out
}

// The engine never reads work state: every engine entry point's output is
// identical before work items exist, while they are created, claimed, and
// moved, and after the work file is deleted again.
//
// [scenario.work-item.never-engine-input]
func TestEngineOutputNeverMovesWithWorkItems(t *testing.T) {
	root, cfg := setupIsolationProject(t)
	ctx := context.Background()

	before := captureEngineOutputs(t, root, cfg)
	if len(before.Scan) == 0 {
		t.Fatal("fixture must produce scan findings, or the comparison proves nothing")
	}

	// Create, claim, and move items — including one pointing at the very
	// spec the engine is verifying.
	p := markdown.New(root, "WORK.md")
	it, err := p.Create(ctx, work.CreateRequest{Title: "Advance the toggle story", Spec: "story.demo.toggle"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Claim(ctx, it.ID); err != nil {
		t.Fatal(err)
	}
	during := captureEngineOutputs(t, root, cfg)
	if !reflect.DeepEqual(before, during) {
		t.Errorf("engine output changed when work items were created and claimed:\nbefore = %+v\nduring = %+v", before, during)
	}

	if _, err := p.Move(ctx, it.ID, "done"); err != nil {
		t.Fatal(err)
	}
	moved := captureEngineOutputs(t, root, cfg)
	if !reflect.DeepEqual(before, moved) {
		t.Errorf("engine output changed when a work item moved")
	}

	// Deleting all work state changes nothing either.
	if err := os.Remove(filepath.Join(root, "WORK.md")); err != nil {
		t.Fatal(err)
	}
	after := captureEngineOutputs(t, root, cfg)
	if !reflect.DeepEqual(before, after) {
		t.Errorf("engine output changed when the work file was deleted")
	}
}

// The engine never requires a work provider: scan, verify, drift, cover,
// parity, and gate produce identical results whether the config has no work
// block, a markdown block, or a beads block with no bd on PATH — and an
// absent block is never a validation error.
//
// [scenario.work.providers.engine-never-requires-a-provider]
func TestEngineNeverRequiresAWorkProvider(t *testing.T) {
	// An empty PATH proves no engine entry point shells out to a provider
	// binary (bd, gh) — a beads config would fail loudly if one did.
	t.Setenv("PATH", t.TempDir())

	root, cfg := setupIsolationProject(t)
	variants := []struct {
		name    string
		content string
	}{
		{"absent", `{"version": 2, "targets": {}}`},
		{"markdown", `{"version": 2, "targets": {}, "work": {"provider": "markdown"}}`},
		{"beads", `{"version": 2, "targets": {}, "work": {"provider": "beads"}}`},
	}

	var baseline *engineOutputs
	var baselineErrs []string
	for _, v := range variants {
		writeIsolationFile(t, root, ".speckit/specs.json", v.content)

		loaded, found, err := config.Load(root)
		if err != nil || !found {
			t.Fatalf("%s: config load = %v, %v", v.name, found, err)
		}
		var errs []string
		for _, e := range loaded.Validate() {
			errs = append(errs, e.Error())
			if strings.Contains(e.Error(), "work") {
				t.Errorf("%s: config validation complained about the work block: %v", v.name, e)
			}
		}

		got := captureEngineOutputs(t, root, cfg)
		if baseline == nil {
			baseline = &got
			baselineErrs = errs
			continue
		}
		if !reflect.DeepEqual(*baseline, got) {
			t.Errorf("engine output for work config %q differs from the no-block baseline", v.name)
		}
		if !reflect.DeepEqual(baselineErrs, errs) {
			t.Errorf("config validation for %q differs from the no-block baseline: %v vs %v", v.name, errs, baselineErrs)
		}
	}

	// An absent work block resolves to the markdown default rather than an
	// error — the engine side of scenario.work.providers.markdown-is-default.
	var noBlock config.Config
	if err := json.Unmarshal([]byte(`{"version": 2}`), &noBlock); err != nil {
		t.Fatal(err)
	}
	w := noBlock.WorkConfig()
	if w.Provider != config.WorkMarkdown || w.File != config.DefaultWorkFile {
		t.Errorf("absent work block resolves to %+v, want the markdown default on %s", w, config.DefaultWorkFile)
	}
}
