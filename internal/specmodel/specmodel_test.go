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
		KindError:       "error.",
		KindProtocol:    "protocol.",
		KindConventions: "", // singular cross-cutting kind: ID is just the kind name
	}
	for k, want := range cases {
		if got := k.Prefix(); got != want {
			t.Errorf("%s.Prefix() = %q, want %q", k, got, want)
		}
	}
}
