package specmodel

import (
	"encoding/json"
	"strings"
	"testing"
)

func hasInvariant(fs []Finding, inv string) bool {
	for _, f := range fs {
		if f.Invariant == inv {
			return true
		}
	}
	return false
}

// findingsFor returns every finding citing the given invariant.
func findingsFor(fs []Finding, inv string) []Finding {
	var out []Finding
	for _, f := range fs {
		if f.Invariant == inv {
			out = append(out, f)
		}
	}
	return out
}

// A clean library yields zero findings, and the findings slice is a non-nil
// empty array — so the scan command's --json projection emits [] and the
// zero-findings condition the CLI derives exit 0 from holds.
//
// SPEC: story.engine.scan
// [scenario.engine.scan.clean]
func TestLintClean(t *testing.T) {
	specs := []Spec{
		{Frontmatter: Frontmatter{ID: "domain.item", Kind: KindDomain}, Path: "specs/models/item.md"},
		{
			Frontmatter: Frontmatter{ID: "story.item.create", Kind: KindStory, DependsOn: []SpecID{"domain.item"}},
			Path:        "features/0001-x/stories/item.create.md",
			Scenarios:   []Scenario{{Heading: "Scenario 1", SubID: "scenario.item.create.ok"}},
		},
		{Frontmatter: Frontmatter{ID: "conventions", Kind: KindConventions}, Path: "specs/CONVENTIONS.md"},
	}
	f := Lint(specs)
	if len(f) != 0 {
		t.Errorf("expected clean, got %v", f)
	}
	// the JSON projection of a clean scan is an empty array, never null
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "[]" {
		t.Errorf("clean findings must marshal to [], got %s", b)
	}
}

// The finding names the offending file, cites I5, and identifies the
// unresolved id.
//
// SPEC: story.engine.scan
// [scenario.engine.scan.dangling-depends-on]
func TestLintDanglingDependsOn(t *testing.T) {
	specs := []Spec{{Frontmatter: Frontmatter{ID: "story.a.b", Kind: KindStory, DependsOn: []SpecID{"domain.missing"}}, Path: "features/x/stories/a.b.md"}}
	fs := findingsFor(Lint(specs), "I5")
	if len(fs) != 1 {
		t.Fatalf("expected one I5 finding for dangling depends-on, got %v", fs)
	}
	if fs[0].Path != "features/x/stories/a.b.md" {
		t.Errorf("finding must name the offending file, got %q", fs[0].Path)
	}
	if !strings.Contains(fs[0].Message, "domain.missing") {
		t.Errorf("finding must identify the unresolved id, got %q", fs[0].Message)
	}
}

// Two specs sharing an id yield a SINGLE finding citing I4 that lists both
// file paths (one as the finding's path, the other in the message). Both
// fixture filenames satisfy I1 (tail form and full-id form), so the duplicate
// is the library's only violation.
//
// SPEC: story.engine.scan
// [scenario.engine.scan.duplicate-id]
func TestLintDuplicateID(t *testing.T) {
	specs := []Spec{
		{Frontmatter: Frontmatter{ID: "domain.x", Kind: KindDomain}, Path: "specs/models/x.md"},
		{Frontmatter: Frontmatter{ID: "domain.x", Kind: KindDomain}, Path: "specs/models/domain.x.md"},
	}
	fs := Lint(specs)
	if len(fs) != 1 {
		t.Fatalf("expected a single finding for the duplicate id, got %v", fs)
	}
	f := fs[0]
	if f.Invariant != "I4" {
		t.Errorf("finding must cite I4, got %q", f.Invariant)
	}
	if !strings.Contains(f.Message, `"domain.x"`) {
		t.Errorf("finding must identify the duplicated id, got %q", f.Message)
	}
	if f.Path != "specs/models/domain.x.md" || !strings.Contains(f.Message, "specs/models/x.md") {
		t.Errorf("finding must list both file paths, got path=%q message=%q", f.Path, f.Message)
	}
}

// The finding cites I1 with both the id and the filename stem.
//
// SPEC: story.engine.scan
// [scenario.engine.scan.filename-id-mismatch]
func TestLintFilenameMismatch(t *testing.T) {
	specs := []Spec{{Frontmatter: Frontmatter{ID: "domain.item", Kind: KindDomain}, Path: "specs/models/wrong.md"}}
	fs := findingsFor(Lint(specs), "I1")
	if len(fs) != 1 {
		t.Fatalf("expected one I1 finding for filename/id mismatch, got %v", fs)
	}
	if fs[0].Path != "specs/models/wrong.md" {
		t.Errorf("finding must name the mismatched file, got %q", fs[0].Path)
	}
	if !strings.Contains(fs[0].Message, `"wrong"`) || !strings.Contains(fs[0].Message, `"domain.item"`) {
		t.Errorf("finding must carry both the filename stem and the id, got %q", fs[0].Message)
	}
}

// Both filename conventions pass I1: the kind-stripped tail (a.b.md) and the full
// dotted id (story.subscriptions.subscribe.md) — a library may use either or, like
// trove, a mix.
func TestLintAcceptsBothFilenameConventions(t *testing.T) {
	specs := []Spec{
		{Frontmatter: Frontmatter{ID: "story.subscriptions.subscribe", Kind: KindStory},
			Path: "features/0001/stories/story.subscriptions.subscribe.md"},
		{Frontmatter: Frontmatter{ID: "story.a.b", Kind: KindStory},
			Path: "features/0001/stories/a.b.md"},
	}
	if hasInvariant(Lint(specs), "I1") {
		t.Errorf("full-id and id-tail filenames must both pass I1: %+v", Lint(specs))
	}
}

// A well-formed protocol spec lints clean: protocol is a valid kind (no I2), its
// protocol. prefix agrees with the id (no I3), and as a contract kind it bears no
// scenarios so I6 never applies.
func TestLintProtocolKindClean(t *testing.T) {
	specs := []Spec{{Frontmatter: Frontmatter{ID: "protocol.troved.search", Kind: KindProtocol},
		Path: "specs/protocol/troved.search.md"}}
	if f := Lint(specs); len(f) != 0 {
		t.Errorf("a well-formed protocol spec should lint clean, got %+v", f)
	}
}

// The finding cites I6 and names the scenario heading whose sub-id is missing.
//
// SPEC: story.engine.scan
// [scenario.engine.scan.missing-scenario-id]
func TestLintMissingScenarioID(t *testing.T) {
	specs := []Spec{{
		Frontmatter: Frontmatter{ID: "story.a.b", Kind: KindStory},
		Path:        "features/x/stories/a.b.md",
		Scenarios:   []Scenario{{Heading: "Scenario 1: x", SubID: ""}},
	}}
	fs := findingsFor(Lint(specs), "I6")
	if len(fs) != 1 {
		t.Fatalf("expected one I6 finding for the missing sub-id, got %v", fs)
	}
	if !strings.Contains(fs[0].Message, `"Scenario 1: x"`) {
		t.Errorf("finding must name the scenario heading, got %q", fs[0].Message)
	}
}

func TestLintUnknownKind(t *testing.T) {
	specs := []Spec{{Frontmatter: Frontmatter{ID: "bogus.x", Kind: "bogus"}, Path: "specs/x.md"}}
	if !hasInvariant(Lint(specs), "I2") {
		t.Error("expected I2 for unknown kind")
	}
}
