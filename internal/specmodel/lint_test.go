package specmodel

import "testing"

func hasInvariant(fs []Finding, inv string) bool {
	for _, f := range fs {
		if f.Invariant == inv {
			return true
		}
	}
	return false
}

// SPEC: story.engine.scan (scenario.engine.scan.clean)
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
	if f := Lint(specs); len(f) != 0 {
		t.Errorf("expected clean, got %v", f)
	}
}

// SPEC: story.engine.scan (scenario.engine.scan.dangling-depends-on)
func TestLintDanglingDependsOn(t *testing.T) {
	specs := []Spec{{Frontmatter: Frontmatter{ID: "story.a.b", Kind: KindStory, DependsOn: []SpecID{"domain.missing"}}, Path: "features/x/stories/a.b.md"}}
	if !hasInvariant(Lint(specs), "I5") {
		t.Error("expected I5 for dangling depends-on")
	}
}

// SPEC: story.engine.scan (scenario.engine.scan.duplicate-id)
func TestLintDuplicateID(t *testing.T) {
	specs := []Spec{
		{Frontmatter: Frontmatter{ID: "domain.x", Kind: KindDomain}, Path: "specs/models/x.md"},
		{Frontmatter: Frontmatter{ID: "domain.x", Kind: KindDomain}, Path: "specs/models/dup.md"},
	}
	if !hasInvariant(Lint(specs), "I4") {
		t.Error("expected I4 for duplicate id")
	}
}

// SPEC: story.engine.scan (scenario.engine.scan.filename-id-mismatch)
func TestLintFilenameMismatch(t *testing.T) {
	specs := []Spec{{Frontmatter: Frontmatter{ID: "domain.item", Kind: KindDomain}, Path: "specs/models/wrong.md"}}
	if !hasInvariant(Lint(specs), "I1") {
		t.Error("expected I1 for filename/id mismatch")
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

// SPEC: story.engine.scan (scenario.engine.scan.missing-scenario-id)
func TestLintMissingScenarioID(t *testing.T) {
	specs := []Spec{{
		Frontmatter: Frontmatter{ID: "story.a.b", Kind: KindStory},
		Path:        "features/x/stories/a.b.md",
		Scenarios:   []Scenario{{Heading: "Scenario 1: x", SubID: ""}},
	}}
	if !hasInvariant(Lint(specs), "I6") {
		t.Error("expected I6 for scenario missing its sub-id")
	}
}

func TestLintUnknownKind(t *testing.T) {
	specs := []Spec{{Frontmatter: Frontmatter{ID: "bogus.x", Kind: "bogus"}, Path: "specs/x.md"}}
	if !hasInvariant(Lint(specs), "I2") {
		t.Error("expected I2 for unknown kind")
	}
}
