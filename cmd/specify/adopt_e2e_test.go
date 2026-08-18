package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/markmals/speckit/internal/engine"
)

// TestAdoptExistingProjectEndToEnd is the fresh-adoption smoke test: a repo whose
// code and tests already exist becomes a verified SpecKit project through
// init → target add → scan → verify, with no platform, stack, or scaffold chosen
// at any point and not one line of code generated.
//
// It is deliberately end-to-end rather than a unit test of any one command: the
// product's central promise is that these four steps compose on a project SpecKit
// did not create, and nothing short of running them proves that.
func TestAdoptExistingProjectEndToEnd(t *testing.T) {
	root := t.TempDir()
	writeAdoptFile(t, root, "go.mod", "module example.com/adopt\n\ngo 1.24\n")
	writeAdoptFile(t, root, "src/cart.go", adoptSource)
	writeAdoptFile(t, root, "src/cart_test.go", adoptTests)

	t.Chdir(root)

	// 1. init into a NON-EMPTY directory — adoption always needs --force, because
	// the guard exists to protect a directory the user already owns.
	if err := cliExecSpecify(t, "init", "--here", "--integration", "claude", "--force"); err != nil {
		t.Fatalf("init: %v", err)
	}

	// 2. The whole adoption vocabulary: where it lives, how to run it, how to read
	// the result. No platform question.
	if err := cliExecSpecify(t, "target", "add", "app",
		"--dir", ".",
		"--format", "gotest",
		"--report", "report.gotest.json",
		"--source", "src",
		"--command", "go test -json ./... > report.gotest.json",
		"--bindings", "scoped",
	); err != nil {
		t.Fatalf("target add: %v", err)
	}

	// 3. A spec whose scenario the pre-existing test already proves.
	writeAdoptFile(t, root, "features/0001-cart/NARRATIVE.md", adoptNarrative)
	writeAdoptFile(t, root, "features/0001-cart/stories/cart.add.md", adoptStory)

	if err := cliExecSpecify(t, "scan"); err != nil {
		t.Fatalf("scan on an adopted library should be clean: %v", err)
	}

	// 4. The join: the untagged plain unit test is out of scope under scoped
	// bindings, the tagged one proves the scenario, and green writes the lock.
	if err := cliExecSpecify(t, "verify", "app"); err != nil {
		t.Fatalf("verify: %v", err)
	}

	shard := filepath.Join(root, ".speckit", "lock", "app", "story.cart.add.json")
	data, err := os.ReadFile(shard)
	if err != nil {
		t.Fatalf("verify went green but wrote no lock shard: %v", err)
	}
	var lock struct {
		SpecHash  string            `json:"spec_hash"`
		Scenarios map[string]string `json:"scenarios"`
	}
	if err := json.Unmarshal(data, &lock); err != nil {
		t.Fatalf("lock shard is not valid JSON: %v", err)
	}
	if lock.Scenarios["scenario.cart.add.appends"] != "pass" {
		t.Errorf("lock scenarios = %v, want scenario.cart.add.appends: pass", lock.Scenarios)
	}
	specSrc, err := os.ReadFile(filepath.Join(root, "features/0001-cart/stories/cart.add.md"))
	if err != nil {
		t.Fatal(err)
	}
	if want := engine.Hash(specSrc); lock.SpecHash != want {
		t.Errorf("lock spec_hash = %q, want the spec's current content hash %q", lock.SpecHash, want)
	}

	// 5. Nothing drifted the moment after locking.
	if err := cliExecSpecify(t, "drift", "app"); err != nil {
		t.Errorf("drift right after a green verify: %v", err)
	}
}

func writeAdoptFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const adoptSource = `package src

// Cart holds line items.
type Cart struct{ items []string }

func (c *Cart) Add(item string) { c.items = append(c.items, item) }

func (c *Cart) Count() int { return len(c.items) }
`

// adoptTests carries one scenario-bound test and one plain unit test, so the
// fixture also proves --bindings scoped tolerates a suite SpecKit did not author.
//
// The scenario tag is split across a concatenation so this file's own fixture
// data never registers as a binding of this file — the leading-comment form is
// language-agnostic by design, so Go binding syntax inside a Go string literal is
// indistinguishable from the real thing without parsing.
const adoptTests = `package src

import "testing"

// [scen` + `ario.cart.add.appends]
func TestAddAppendsAnItem(t *testing.T) {
	var c Cart
	c.Add("apple")
	if c.Count() != 1 {
		t.Fatalf("Count() = %d, want 1", c.Count())
	}
}

func TestCountStartsAtZero(t *testing.T) {
	var c Cart
	if c.Count() != 0 {
		t.Fatal("empty cart should count 0")
	}
}
`

const adoptNarrative = `---
id: narrative.cart
kind: narrative
---

# Narrative: Cart

A shopper collects items before checking out.
`

const adoptStory = `---
id: story.cart.add
kind: story
---

# Story: Add an item to the cart

As a shopper,
I want to add an item to my cart,
So that I can buy it later.

# Acceptance Criteria

## Scenario 1: Adding an item appends it

<!-- id: scenario.cart.add.appends -->

- Given an empty cart
- When the shopper adds an item
- Then the cart contains exactly that one item
`
