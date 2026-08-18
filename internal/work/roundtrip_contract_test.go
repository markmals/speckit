// The provider-agnostic five-verb contract (story.work.roundtrip): the same
// table of behavior is exercised against every hermetically-testable
// provider — markdown for real on a temp dir, github-projects through a
// stateful fake of its client seam. The beads adapter's injectable runner is
// package-private, so its identical verb behavior is proven in-package by
// internal/work/beads's own round-trip tests.
package work_test

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/github"
	"github.com/markmals/speckit/internal/work"
	"github.com/markmals/speckit/internal/work/ghprojects"
	"github.com/markmals/speckit/internal/work/markdown"
)

// boardFake is a stateful ghprojects.Client: issues live in memory, AddItem
// puts them on the board, SetSingleSelect moves their column — so the five
// verbs behave observably, not just record calls.
type boardFake struct {
	nextNumber int
	issues     map[int]github.Issue
	status     map[int]string // issue number → column name
	onBoard    map[int]bool
	created    []github.CreateIssueInput
}

func newBoardFake() *boardFake {
	return &boardFake{
		nextNumber: 1,
		issues:     map[int]github.Issue{},
		status:     map[int]string{},
		onBoard:    map[int]bool{},
	}
}

func (f *boardFake) project() github.Project {
	return github.Project{ID: "PVT_1", Title: "Work", Number: 3, Fields: []github.Field{{
		ID: "F_status", Name: "Status", Options: []github.FieldOption{
			{ID: "opt_ready", Name: "Ready"},
			{ID: "opt_doing", Name: "In Progress"},
			{ID: "opt_hold", Name: "On Hold"},
			{ID: "opt_closed", Name: "Closed"},
		},
	}}}
}

func (f *boardFake) ResolveProject(context.Context, string, int) (github.Project, error) {
	return f.project(), nil
}

func (f *boardFake) ListItems(context.Context, string, string) ([]github.Item, error) {
	var items []github.Item
	for n, iss := range f.issues {
		if !f.onBoard[n] {
			continue
		}
		items = append(items, github.Item{
			Number: n, Title: iss.Title, Status: f.status[n], State: iss.State, URL: iss.HTMLURL,
		})
	}
	return items, nil
}

func (f *boardFake) AddItem(_ context.Context, _ string, contentNodeID string) (string, error) {
	for n, iss := range f.issues {
		if iss.NodeID == contentNodeID {
			f.onBoard[n] = true
			return "ITEM_" + strconv.Itoa(n), nil
		}
	}
	return "", fmt.Errorf("no issue with node id %q", contentNodeID)
}

func (f *boardFake) SetSingleSelect(_ context.Context, _ string, itemID, _ string, optionID string) error {
	n, err := strconv.Atoi(strings.TrimPrefix(itemID, "ITEM_"))
	if err != nil {
		return fmt.Errorf("bad item id %q", itemID)
	}
	for _, opt := range f.project().Fields[0].Options {
		if opt.ID == optionID {
			f.status[n] = opt.Name
			return nil
		}
	}
	return fmt.Errorf("no option %q", optionID)
}

func (f *boardFake) GetIssue(_ context.Context, _ github.Repo, number int) (github.Issue, error) {
	iss, ok := f.issues[number]
	if !ok {
		return github.Issue{}, fmt.Errorf("no issue #%d", number)
	}
	return iss, nil
}

func (f *boardFake) CreateIssue(_ context.Context, _ github.Repo, in github.CreateIssueInput) (github.Issue, error) {
	n := f.nextNumber
	f.nextNumber++
	iss := github.Issue{
		Number: n, Title: in.Title, Body: in.Body, State: "OPEN",
		NodeID: "I_" + strconv.Itoa(n), HTMLURL: "https://github.com/o/r/issues/" + strconv.Itoa(n),
	}
	f.issues[n] = iss
	f.created = append(f.created, in)
	return iss, nil
}

func (f *boardFake) AssignIssue(_ context.Context, _ github.Repo, number int, assignees []string) error {
	iss := f.issues[number]
	for _, a := range assignees {
		iss.Assignees = append(iss.Assignees, github.User{Login: a})
	}
	f.issues[number] = iss
	return nil
}

func (f *boardFake) Viewer(context.Context) (string, error) { return "octocat", nil }

// contractProvider is one provider under the shared contract; fake is nil
// for providers with real local storage.
type contractProvider struct {
	name string
	new  func(t *testing.T) (work.Provider, *boardFake)
}

func contractProviders() []contractProvider {
	return []contractProvider{
		{"markdown", func(t *testing.T) (work.Provider, *boardFake) {
			return markdown.New(t.TempDir(), "WORK.md"), nil
		}},
		{"github-projects", func(t *testing.T) (work.Provider, *boardFake) {
			f := newBoardFake()
			p, err := ghprojects.New(f, github.Repo{Owner: "o", Name: "r"}, ghprojects.Options{Project: 3})
			if err != nil {
				t.Fatal(err)
			}
			return p, f
		}},
	}
}

func ids(items []work.Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.ID
	}
	return out
}

// [scenario.work.roundtrip.create-lands-ready]
func TestCreateLandsReadyOnEveryProvider(t *testing.T) {
	ctx := context.Background()
	for _, cp := range contractProviders() {
		t.Run(cp.name, func(t *testing.T) {
			p, _ := cp.new(t)
			it, err := p.Create(ctx, work.CreateRequest{Title: "Wire the junit adapter"})
			if err != nil {
				t.Fatal(err)
			}
			if it.State != work.StateReady {
				t.Fatalf("created item state = %q, want %q", it.State, work.StateReady)
			}
			ready, err := p.Ready(ctx)
			if err != nil || len(ready) != 1 || ready[0].ID != it.ID {
				t.Fatalf("Ready() = %+v, %v — created item must land in ready", ready, err)
			}
			byState, err := p.List(ctx, work.StateReady)
			if err != nil || len(byState) != 1 || byState[0].ID != it.ID {
				t.Fatalf("List(ready) = %+v, %v", byState, err)
			}
		})
	}
}

// [scenario.work.roundtrip.claim-moves-to-in-progress]
func TestClaimMovesToInProgressOnEveryProvider(t *testing.T) {
	ctx := context.Background()
	for _, cp := range contractProviders() {
		t.Run(cp.name, func(t *testing.T) {
			p, _ := cp.new(t)
			it, err := p.Create(ctx, work.CreateRequest{Title: "Event log append path"})
			if err != nil {
				t.Fatal(err)
			}
			claimed, err := p.Claim(ctx, it.ID)
			if err != nil {
				t.Fatal(err)
			}
			if claimed.State != work.StateInProgress {
				t.Fatalf("claimed state = %q, want %q", claimed.State, work.StateInProgress)
			}
			if ready, err := p.Ready(ctx); err != nil || len(ready) != 0 {
				t.Errorf("Ready() after claim = %+v, %v — claimed item must leave ready", ready, err)
			}
			doing, err := p.List(ctx, work.StateInProgress)
			if err != nil || len(doing) != 1 || doing[0].ID != it.ID {
				t.Errorf("List(in-progress) = %+v, %v", doing, err)
			}
		})
	}
}

// [scenario.work.roundtrip.move-to-state]
func TestMoveReachesEveryCanonicalStateOnEveryProvider(t *testing.T) {
	ctx := context.Background()
	for _, cp := range contractProviders() {
		t.Run(cp.name, func(t *testing.T) {
			p, _ := cp.new(t)
			it, err := p.Create(ctx, work.CreateRequest{Title: "Fix the drift exit code"})
			if err != nil {
				t.Fatal(err)
			}
			for _, state := range work.CanonicalStates {
				moved, err := p.Move(ctx, it.ID, state)
				if err != nil {
					t.Fatalf("Move(%s): %v", state, err)
				}
				if moved.State != state {
					t.Fatalf("Move(%s) returned state %q", state, moved.State)
				}
				listed, err := p.List(ctx, state)
				if err != nil || len(listed) != 1 || listed[0].ID != it.ID {
					t.Fatalf("List(%s) after move = %+v, %v — the move must be reflected", state, listed, err)
				}
				for _, other := range work.CanonicalStates {
					if other == state {
						continue
					}
					if elsewhere, err := p.List(ctx, other); err != nil || len(elsewhere) != 0 {
						t.Fatalf("List(%s) after move to %s = %+v, %v — the item sits in exactly one state", other, state, elsewhere, err)
					}
				}
			}
		})
	}
}

// [scenario.work.roundtrip.list-all]
func TestListWithoutAStateReturnsEverythingOnEveryProvider(t *testing.T) {
	ctx := context.Background()
	for _, cp := range contractProviders() {
		t.Run(cp.name, func(t *testing.T) {
			p, _ := cp.new(t)
			var created []string
			for _, title := range []string{"A", "B", "C"} {
				it, err := p.Create(ctx, work.CreateRequest{Title: title})
				if err != nil {
					t.Fatal(err)
				}
				created = append(created, it.ID)
			}
			// Spread the items across several states.
			if _, err := p.Claim(ctx, created[1]); err != nil {
				t.Fatal(err)
			}
			if _, err := p.Move(ctx, created[2], work.StateDone); err != nil {
				t.Fatal(err)
			}
			all, err := p.List(ctx, "")
			if err != nil {
				t.Fatal(err)
			}
			got := ids(all)
			if len(got) != len(created) {
				t.Fatalf("List(\"\") = %v, want every item %v", got, created)
			}
			for _, id := range created {
				found := false
				for _, g := range got {
					found = found || g == id
				}
				if !found {
					t.Errorf("List(\"\") = %v — item %s across states was dropped", got, id)
				}
			}
		})
	}
}

// A defect and a task differ only by type: created identically, they carry
// the same shape (state, title, spec) and only Type diverges.
//
// [scenario.work.roundtrip.defect-type]
// [scenario.work-item.defect-is-a-type]
func TestDefectDiffersFromTaskOnlyByType(t *testing.T) {
	ctx := context.Background()
	for _, cp := range contractProviders() {
		t.Run(cp.name, func(t *testing.T) {
			p, _ := cp.new(t)
			task, err := p.Create(ctx, work.CreateRequest{Title: "Crash on empty id", Spec: "story.x"})
			if err != nil {
				t.Fatal(err)
			}
			defect, err := p.Create(ctx, work.CreateRequest{Title: "Crash on empty id", Type: work.TypeDefect, Spec: "story.x"})
			if err != nil {
				t.Fatal(err)
			}
			if task.Type != "" {
				t.Errorf("task type = %q, want the zero type", task.Type)
			}
			if defect.Type != work.TypeDefect {
				t.Errorf("defect type = %q, want %q", defect.Type, work.TypeDefect)
			}
			// Identical except for Type (ids and provider links are
			// per-item by nature).
			a, b := task, defect
			a.ID, b.ID, a.URL, b.URL = "", "", "", ""
			a.Type, b.Type = "", ""
			if !reflect.DeepEqual(a, b) {
				t.Errorf("defect and task differ beyond type:\ntask   = %+v\ndefect = %+v", a, b)
			}
			// Both move through the same states.
			for _, it := range []work.Item{task, defect} {
				claimed, err := p.Claim(ctx, it.ID)
				if err != nil || claimed.State != work.StateInProgress {
					t.Errorf("claim %s = %+v, %v — defects and tasks share the state machine", it.ID, claimed, err)
				}
			}
		})
	}
}

// [scenario.work.roundtrip.spec-pointer]
// [scenario.work-item.spec-pointer]
func TestSpecPointerIsRecordedOnEveryProvider(t *testing.T) {
	ctx := context.Background()
	for _, cp := range contractProviders() {
		t.Run(cp.name, func(t *testing.T) {
			p, fake := cp.new(t)
			it, err := p.Create(ctx, work.CreateRequest{Title: "Write the parity docs", Spec: "story.engine.parity"})
			if err != nil {
				t.Fatal(err)
			}
			if it.Spec != "story.engine.parity" {
				t.Fatalf("created spec pointer = %q, want story.engine.parity", it.Spec)
			}
			if fake != nil {
				// github-projects records the pointer durably on the issue.
				if len(fake.created) != 1 || !strings.Contains(fake.created[0].Body, "story.engine.parity") {
					t.Errorf("spec pointer not recorded on the issue: %+v", fake.created)
				}
				return
			}
			// markdown: the pointer survives a full storage round trip.
			listed, err := p.List(ctx, "")
			if err != nil || len(listed) != 1 || listed[0].Spec != "story.engine.parity" {
				t.Errorf("spec pointer lost in round trip: %+v, %v", listed, err)
			}
		})
	}
}
