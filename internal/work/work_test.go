package work

import (
	"slices"
	"testing"
)

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"In progress":       "in-progress",
		"In Progress":       "in-progress",
		"  Ready  ":         "ready",
		"Waiting   for\tQA": "waiting-for-qa",
		"":                  "",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHeading(t *testing.T) {
	cases := map[string]string{
		"in-progress": "In progress",
		"ready":       "Ready",
		"":            "",
	}
	for in, want := range cases {
		if got := Heading(in); got != want {
			t.Errorf("Heading(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHeadingRoundTripsCanonicalStates(t *testing.T) {
	for _, state := range CanonicalStates {
		if got := Slug(Heading(state)); got != state {
			t.Errorf("Slug(Heading(%q)) = %q", state, got)
		}
	}
}

func TestIsCanonical(t *testing.T) {
	for _, state := range CanonicalStates {
		if !IsCanonical(state) {
			t.Errorf("IsCanonical(%q) = false", state)
		}
	}
	for _, state := range []string{"", "open", "Ready", "waiting-for-qa"} {
		if IsCanonical(state) {
			t.Errorf("IsCanonical(%q) = true", state)
		}
	}
}

// The canonical vocabulary is exactly these four states, in board order —
// no fifth state, none dropped — and IsCanonical accepts exactly them.
//
// [scenario.work-item.canonical-states]
func TestCanonicalStatesAreExactlyTheFour(t *testing.T) {
	want := []string{"ready", "in-progress", "blocked", "done"}
	if !slices.Equal(CanonicalStates, want) {
		t.Fatalf("CanonicalStates = %v, want exactly %v in this order", CanonicalStates, want)
	}
	if StateReady != "ready" || StateInProgress != "in-progress" || StateBlocked != "blocked" || StateDone != "done" {
		t.Errorf("state constants diverge from the canonical vocabulary")
	}
	for _, state := range want {
		if !IsCanonical(state) {
			t.Errorf("IsCanonical(%q) = false, want true", state)
		}
	}
	for _, state := range []string{"", "open", "closed", "in_progress", "Ready", "DONE", "someday", "waiting-for-qa"} {
		if IsCanonical(state) {
			t.Errorf("IsCanonical(%q) = true — a fifth state has leaked into the vocabulary", state)
		}
	}
}
