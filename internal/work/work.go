// Package work is the provider-neutral work-tracking contract: the item
// shape, the canonical state vocabulary, and the Provider interface every
// adapter implements. It is types plus pure helpers — no provider, no
// network, no filesystem.
package work

import (
	"context"
	"slices"
	"strings"
	"unicode"
)

// Item is one unit of work, normalized across providers.
type Item struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	State string `json:"state"`
	Type  string `json:"type,omitempty"` // "" == task
	Spec  string `json:"spec,omitempty"` // spec id this item advances
	URL   string `json:"url,omitempty"`  // provider-native link, when it has one
}

// Canonical states: the vocabulary create/claim and the fixed-vocabulary
// providers use. Providers with free-form states (markdown) may carry others.
const (
	StateReady      = "ready"
	StateInProgress = "in-progress"
	StateBlocked    = "blocked"
	StateDone       = "done"
)

// CanonicalStates lists the canonical states in board order.
var CanonicalStates = []string{StateReady, StateInProgress, StateBlocked, StateDone}

// Item types. Task is the zero value: an Item with Type "" is a task.
const (
	TypeTask   = "task"
	TypeDefect = "defect"
)

// CreateRequest describes a new item; Create lands it in the ready state.
type CreateRequest struct {
	Title string
	Type  string // task | defect; "" == task
	Spec  string
}

// Provider is one work-tracking backend. Create lands an item in ready;
// Claim moves it to in-progress.
type Provider interface {
	Name() string
	Ready(ctx context.Context) ([]Item, error)
	Create(ctx context.Context, req CreateRequest) (Item, error)
	Claim(ctx context.Context, id string) (Item, error)
	Move(ctx context.Context, id, state string) (Item, error)
	List(ctx context.Context, state string) ([]Item, error) // state == "" → all
}

// Slug maps a section heading to a state name: lowercased, runs of
// whitespace collapsed to a single '-', surrounding whitespace trimmed.
// "In progress" → "in-progress".
func Slug(heading string) string {
	return strings.Join(strings.Fields(strings.ToLower(heading)), "-")
}

// Heading is the inverse rendering: '-' to space, first letter upper.
// "in-progress" → "In progress". Round-trips every canonical state
// through Slug.
func Heading(state string) string {
	r := []rune(strings.ReplaceAll(state, "-", " "))
	if len(r) > 0 {
		r[0] = unicode.ToUpper(r[0])
	}
	return string(r)
}

// IsCanonical reports whether state is one of the canonical four.
func IsCanonical(state string) bool {
	return slices.Contains(CanonicalStates, state)
}
