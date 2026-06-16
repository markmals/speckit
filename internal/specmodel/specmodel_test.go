package specmodel

import "testing"

func TestKindsClosedSetIsValid(t *testing.T) {
	for _, k := range Kinds {
		if !k.Valid() {
			t.Errorf("Kind %q is listed in Kinds but Valid() is false", k)
		}
	}
	if Kind("bogus").Valid() {
		t.Error(`Kind("bogus").Valid() = true, want false`)
	}
}

func TestKindPrefixes(t *testing.T) {
	cases := map[Kind]string{
		KindDomain:      "domain.",
		KindViewModel:   "vm.",
		KindStory:       "story.",
		KindUseCase:     "usecase.",
		KindCommand:     "command.",
		KindError:       "error.",
		KindProtocol:    "protocol.",
		KindConventions: "", // singular cross-cutting kind: ID is just the kind name
	}

	// command is in the closed taxonomy (CLI command behavior specs).
	if !KindCommand.Valid() {
		t.Error("KindCommand must be in the closed Kinds taxonomy")
	}
	for k, want := range cases {
		if got := k.Prefix(); got != want {
			t.Errorf("%s.Prefix() = %q, want %q", k, got, want)
		}
	}
}

// CarriesScenarios is the scenario-parsing gate. Every non-singular spec kind is a
// per-file behavioral contract that may declare scenarios the engine joins —
// story/domain (already) plus protocol and the other behavioral kinds. The singular
// cross-cutting doc kinds (narrative, architecture, design-system, conventions) do
// not: CONVENTIONS.md itself carries an *illustrative* `<!-- id: scenario… -->`,
// which must never be parsed as a real declaration.
func TestCarriesScenarios(t *testing.T) {
	carries := []Kind{
		KindStory, KindDomain, KindProtocol, KindUseCase,
		KindFlow, KindViewModel, KindCommand, KindError,
	}
	for _, k := range carries {
		if !k.CarriesScenarios() {
			t.Errorf("%s is a behavioral (non-singular) kind and must carry scenarios", k)
		}
		if k.Singular() {
			t.Errorf("%s must not be Singular()", k)
		}
	}
	for _, k := range []Kind{KindNarrative, KindArchitecture, KindDesignSystem, KindConventions} {
		if k.CarriesScenarios() {
			t.Errorf("%s is a singular cross-cutting doc kind and must not carry scenarios", k)
		}
		if !k.Singular() {
			t.Errorf("%s must be Singular()", k)
		}
	}
}
