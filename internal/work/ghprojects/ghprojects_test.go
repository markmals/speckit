package ghprojects

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/github"
	"github.com/markmals/speckit/internal/work"
)

// fakeClient records the call sequence and serves a canned board: project
// PVT_1 with a Status field carrying the default columns, plus Backlog.
type fakeClient struct {
	calls     []string
	items     []github.Item
	issue     github.Issue
	created   github.CreateIssueInput
	addErr    error
	assignErr error
}

func board() github.Project {
	opts := []github.FieldOption{
		{ID: "opt_ready", Name: "Ready"},
		{ID: "opt_doing", Name: "In Progress"},
		{ID: "opt_hold", Name: "On Hold"},
		{ID: "opt_closed", Name: "Closed"},
		{ID: "opt_backlog", Name: "Backlog"},
	}
	return github.Project{ID: "PVT_1", Title: "Work", Number: 3, Fields: []github.Field{{ID: "F_status", Name: "Status", Options: opts}}}
}

func (f *fakeClient) ResolveProject(_ context.Context, owner string, number int) (github.Project, error) {
	f.calls = append(f.calls, fmt.Sprintf("ResolveProject(%s,%d)", owner, number))
	return board(), nil
}

func (f *fakeClient) ListItems(_ context.Context, projectID, statusField string) ([]github.Item, error) {
	f.calls = append(f.calls, fmt.Sprintf("ListItems(%s,%s)", projectID, statusField))
	return f.items, nil
}

func (f *fakeClient) AddItem(_ context.Context, projectID, contentNodeID string) (string, error) {
	f.calls = append(f.calls, fmt.Sprintf("AddItem(%s,%s)", projectID, contentNodeID))
	return "ITEM_1", f.addErr
}

func (f *fakeClient) SetSingleSelect(_ context.Context, projectID, itemID, fieldID, optionID string) error {
	f.calls = append(f.calls, fmt.Sprintf("SetSingleSelect(%s,%s,%s,%s)", projectID, itemID, fieldID, optionID))
	return nil
}

func (f *fakeClient) GetIssue(_ context.Context, _ github.Repo, number int) (github.Issue, error) {
	f.calls = append(f.calls, fmt.Sprintf("GetIssue(%d)", number))
	return f.issue, nil
}

func (f *fakeClient) CreateIssue(_ context.Context, _ github.Repo, in github.CreateIssueInput) (github.Issue, error) {
	f.calls = append(f.calls, "CreateIssue")
	f.created = in
	return github.Issue{Number: 42, Title: in.Title, NodeID: "I_42", HTMLURL: "https://github.com/o/r/issues/42"}, nil
}

func (f *fakeClient) AssignIssue(_ context.Context, _ github.Repo, number int, assignees []string) error {
	f.calls = append(f.calls, fmt.Sprintf("AssignIssue(%d,%s)", number, strings.Join(assignees, ",")))
	return f.assignErr
}

func (f *fakeClient) Viewer(context.Context) (string, error) {
	f.calls = append(f.calls, "Viewer")
	return "octocat", nil
}

func newProvider(t *testing.T, f *fakeClient) *Provider {
	t.Helper()
	p, err := New(f, github.Repo{Owner: "o", Name: "r"}, Options{Project: 3})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestNewDefaults(t *testing.T) {
	p := newProvider(t, &fakeClient{})
	if p.opts.Owner != "o" || p.opts.StatusField != "Status" {
		t.Errorf("defaults = %+v", p.opts)
	}
	if p.opts.Columns[work.StateBlocked] != "On Hold" {
		t.Errorf("columns = %v", p.opts.Columns)
	}
	if _, err := New(&fakeClient{}, github.Repo{Owner: "o"}, Options{}); err == nil {
		t.Error("New must require a project number")
	}
	if _, err := New(&fakeClient{}, github.Repo{Owner: "o"}, Options{Project: 3, Columns: map[string]string{"someday": "X"}}); err == nil {
		t.Error("New must reject a column override for a non-canonical state")
	}
}

func TestPartialColumnOverride(t *testing.T) {
	p, err := New(&fakeClient{}, github.Repo{Owner: "o"}, Options{Project: 3, Columns: map[string]string{work.StateReady: "Todo"}})
	if err != nil {
		t.Fatal(err)
	}
	if p.opts.Columns[work.StateReady] != "Todo" || p.opts.Columns[work.StateDone] != "Closed" {
		t.Errorf("columns = %v", p.opts.Columns)
	}
}

func TestReadyFiltersColumnAndOpenState(t *testing.T) {
	f := &fakeClient{items: []github.Item{
		{Number: 1, Title: "in ready, open", Status: "Ready", State: "OPEN", URL: "u1"},
		{Number: 2, Title: "in ready, closed", Status: "Ready", State: "CLOSED"},
		{Number: 3, Title: "elsewhere", Status: "Backlog", State: "OPEN"},
	}}
	items, err := newProvider(t, f).Ready(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "1" || items[0].State != work.StateReady || items[0].URL != "u1" {
		t.Errorf("ready = %+v", items)
	}
}

func TestListMapsColumnsAndPassesUnknownThrough(t *testing.T) {
	f := &fakeClient{items: []github.Item{
		{Number: 1, Title: "a", Status: "In Progress", State: "OPEN"},
		{Number: 2, Title: "b", Status: "Backlog", State: "OPEN"},
	}}
	items, err := newProvider(t, f).List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if items[0].State != work.StateInProgress {
		t.Errorf("mapped state = %q", items[0].State)
	}
	if items[1].State != "Backlog" { // unmapped column passes through as-is
		t.Errorf("unmapped column = %q", items[1].State)
	}

	filtered, err := newProvider(t, f).List(context.Background(), work.StateInProgress)
	if err != nil || len(filtered) != 1 || filtered[0].ID != "1" {
		t.Errorf("filtered = %+v, %v", filtered, err)
	}
}

func TestCreatePreflightsColumnBeforeOpeningTheIssue(t *testing.T) {
	f := &fakeClient{}
	it, err := newProvider(t, f).Create(context.Background(), work.CreateRequest{Title: "Crash", Type: work.TypeDefect, Spec: "story.x"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"ResolveProject(o,3)", "CreateIssue", "AddItem(PVT_1,I_42)", "SetSingleSelect(PVT_1,ITEM_1,F_status,opt_ready)"}
	if !slices.Equal(f.calls, want) {
		t.Errorf("calls = %v, want %v", f.calls, want)
	}
	if f.created.Type != "Bug" || !slices.Contains(f.created.Labels, "bug") {
		t.Errorf("defect issue shape = %+v", f.created)
	}
	if !strings.Contains(f.created.Body, "story.x") {
		t.Errorf("spec pointer missing from body: %q", f.created.Body)
	}
	if it.ID != "42" || it.State != work.StateReady || it.Type != work.TypeDefect || it.URL == "" {
		t.Errorf("created = %+v", it)
	}
}

func TestCreateTaskShape(t *testing.T) {
	f := &fakeClient{}
	it, err := newProvider(t, f).Create(context.Background(), work.CreateRequest{Title: "Chore", Type: work.TypeTask})
	if err != nil {
		t.Fatal(err)
	}
	if f.created.Type != "Task" || len(f.created.Labels) != 0 {
		t.Errorf("task issue shape = %+v", f.created)
	}
	if it.Type != "" { // "" == task
		t.Errorf("task type must normalize to zero, got %q", it.Type)
	}
}

func TestCreateWithBadColumnMutatesNothing(t *testing.T) {
	f := &fakeClient{}
	p, err := New(f, github.Repo{Owner: "o", Name: "r"}, Options{Project: 3, Columns: map[string]string{work.StateReady: "No Such Column"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Create(context.Background(), work.CreateRequest{Title: "x"}); err == nil {
		t.Fatal("bad ready column must fail create")
	}
	if slices.Contains(f.calls, "CreateIssue") {
		t.Errorf("issue opened despite failed preflight: %v", f.calls)
	}
}

func TestClaimPreflightsBeforeAssigning(t *testing.T) {
	f := &fakeClient{issue: github.Issue{Number: 7, Title: "T", NodeID: "I_7"}}
	it, err := newProvider(t, f).Claim(context.Background(), "7")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"ResolveProject(o,3)", "GetIssue(7)", "Viewer", "AssignIssue(7,octocat)", "AddItem(PVT_1,I_7)", "SetSingleSelect(PVT_1,ITEM_1,F_status,opt_doing)"}
	if !slices.Equal(f.calls, want) {
		t.Errorf("calls = %v, want %v", f.calls, want)
	}
	if it.State != work.StateInProgress || it.ID != "7" {
		t.Errorf("claimed = %+v", it)
	}
}

func TestClaimRefusesForeignAssignee(t *testing.T) {
	f := &fakeClient{issue: github.Issue{Number: 7, Assignees: []github.User{{Login: "someone-else"}}}}
	_, err := newProvider(t, f).Claim(context.Background(), "7")
	if err == nil || !strings.Contains(err.Error(), "someone-else") {
		t.Fatalf("claim of a foreign issue = %v", err)
	}
	for _, call := range f.calls {
		if strings.HasPrefix(call, "AssignIssue") || strings.HasPrefix(call, "AddItem") || strings.HasPrefix(call, "SetSingleSelect") {
			t.Errorf("mutation ran despite refusal: %v", f.calls)
		}
	}
}

func TestClaimBadColumnFailsBeforeAnyMutation(t *testing.T) {
	f := &fakeClient{issue: github.Issue{Number: 7, NodeID: "I_7"}}
	p, err := New(f, github.Repo{Owner: "o", Name: "r"}, Options{Project: 3, Columns: map[string]string{work.StateInProgress: "Bogus"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Claim(context.Background(), "7"); err == nil || !strings.Contains(err.Error(), "Bogus") {
		t.Fatalf("claim with bad column = %v", err)
	}
	// The bad column fails at preflight — the issue is never half-claimed.
	if slices.Contains(f.calls, "AssignIssue(7,octocat)") {
		t.Errorf("assigned despite failed preflight: %v", f.calls)
	}
}

func TestMoveSequence(t *testing.T) {
	f := &fakeClient{issue: github.Issue{Number: 9, Title: "T", NodeID: "I_9"}}
	it, err := newProvider(t, f).Move(context.Background(), "9", work.StateBlocked)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"ResolveProject(o,3)", "GetIssue(9)", "AddItem(PVT_1,I_9)", "SetSingleSelect(PVT_1,ITEM_1,F_status,opt_hold)"}
	if !slices.Equal(f.calls, want) {
		t.Errorf("calls = %v, want %v", f.calls, want)
	}
	if it.State != work.StateBlocked {
		t.Errorf("moved = %+v", it)
	}
}

func TestMoveRejectsUnknownStateBeforeAnyCall(t *testing.T) {
	f := &fakeClient{}
	_, err := newProvider(t, f).Move(context.Background(), "9", "someday")
	if err == nil || !strings.Contains(err.Error(), `"someday"`) {
		t.Fatalf("move unknown state = %v", err)
	}
	if len(f.calls) != 0 {
		t.Errorf("network calls ran for a rejected state: %v", f.calls)
	}
	if _, err := newProvider(t, f).Move(context.Background(), "not-a-number", work.StateDone); err == nil {
		t.Error("move must reject a non-numeric issue id")
	}
}

// overrideFake serves a board whose status lives in a custom "State" field
// (a decoy default-named "Status" field is present too), so the assertions
// below fail if an override is ignored.
type overrideFake struct {
	calls []string
	items []github.Item
	issue github.Issue
}

func overrideBoard() github.Project {
	return github.Project{ID: "PVT_1", Title: "Work", Number: 3, Fields: []github.Field{
		{ID: "F_status", Name: "Status", Options: []github.FieldOption{
			{ID: "opt_ready", Name: "Ready"},
			{ID: "opt_doing", Name: "In Progress"},
			{ID: "opt_hold", Name: "On Hold"},
			{ID: "opt_closed", Name: "Closed"},
		}},
		{ID: "F_state", Name: "State", Options: []github.FieldOption{
			{ID: "opt_todo", Name: "Todo"},
			{ID: "opt_doing2", Name: "Doing"},
			{ID: "opt_paused", Name: "Paused"},
			{ID: "opt_shipped", Name: "Shipped"},
		}},
	}}
}

func (f *overrideFake) ResolveProject(_ context.Context, owner string, number int) (github.Project, error) {
	f.calls = append(f.calls, fmt.Sprintf("ResolveProject(%s,%d)", owner, number))
	return overrideBoard(), nil
}

func (f *overrideFake) ListItems(_ context.Context, projectID, statusField string) ([]github.Item, error) {
	f.calls = append(f.calls, fmt.Sprintf("ListItems(%s,%s)", projectID, statusField))
	return f.items, nil
}

func (f *overrideFake) AddItem(_ context.Context, projectID, contentNodeID string) (string, error) {
	f.calls = append(f.calls, fmt.Sprintf("AddItem(%s,%s)", projectID, contentNodeID))
	return "ITEM_1", nil
}

func (f *overrideFake) SetSingleSelect(_ context.Context, projectID, itemID, fieldID, optionID string) error {
	f.calls = append(f.calls, fmt.Sprintf("SetSingleSelect(%s,%s,%s,%s)", projectID, itemID, fieldID, optionID))
	return nil
}

func (f *overrideFake) GetIssue(_ context.Context, _ github.Repo, number int) (github.Issue, error) {
	f.calls = append(f.calls, fmt.Sprintf("GetIssue(%d)", number))
	return f.issue, nil
}

func (f *overrideFake) CreateIssue(_ context.Context, _ github.Repo, in github.CreateIssueInput) (github.Issue, error) {
	f.calls = append(f.calls, "CreateIssue")
	return github.Issue{Number: 42, Title: in.Title, NodeID: "I_42", HTMLURL: "https://github.com/o/r/issues/42"}, nil
}

func (f *overrideFake) AssignIssue(_ context.Context, _ github.Repo, number int, assignees []string) error {
	f.calls = append(f.calls, fmt.Sprintf("AssignIssue(%d,%s)", number, strings.Join(assignees, ",")))
	return nil
}

func (f *overrideFake) Viewer(context.Context) (string, error) {
	f.calls = append(f.calls, "Viewer")
	return "octocat", nil
}

// The five verbs drive a Projects v2 board through the client seam, with
// the status field and every column name overridden — each mutation lands
// in the overridden field's options, reads filter by the overridden
// columns, and preflight (project + column resolution) precedes every
// mutation.
//
// [scenario.work.providers.github-projects-board]
func TestFiveVerbsDriveTheBoardWithOverriddenFieldAndColumns(t *testing.T) {
	ctx := context.Background()
	newP := func(f *overrideFake) *Provider {
		p, err := New(f, github.Repo{Owner: "o", Name: "r"}, Options{
			Project:     3,
			StatusField: "State",
			Columns: map[string]string{
				work.StateReady:      "Todo",
				work.StateInProgress: "Doing",
				work.StateBlocked:    "Paused",
				work.StateDone:       "Shipped",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		return p
	}

	// ready + list read through the overridden field and map the
	// overridden columns back to canonical states.
	f := &overrideFake{items: []github.Item{
		{Number: 1, Title: "todo", Status: "Todo", State: "OPEN"},
		{Number: 2, Title: "shipped", Status: "Shipped", State: "OPEN"},
	}}
	ready, err := newP(f).Ready(ctx)
	if err != nil || len(ready) != 1 || ready[0].ID != "1" || ready[0].State != work.StateReady {
		t.Fatalf("ready = %+v, %v", ready, err)
	}
	done, err := newP(f).List(ctx, work.StateDone)
	if err != nil || len(done) != 1 || done[0].ID != "2" || done[0].State != work.StateDone {
		t.Fatalf("list done = %+v, %v", done, err)
	}
	if want := "ListItems(PVT_1,State)"; !slices.Contains(f.calls, want) {
		t.Errorf("reads did not use the overridden status field: %v", f.calls)
	}

	// create preflights, then lands the card in the overridden ready column.
	f = &overrideFake{}
	if _, err := newP(f).Create(ctx, work.CreateRequest{Title: "New"}); err != nil {
		t.Fatal(err)
	}
	want := []string{"ResolveProject(o,3)", "CreateIssue", "AddItem(PVT_1,I_42)", "SetSingleSelect(PVT_1,ITEM_1,F_state,opt_todo)"}
	if !slices.Equal(f.calls, want) {
		t.Errorf("create calls = %v, want %v", f.calls, want)
	}

	// claim preflights before assigning, then lands in the overridden
	// in-progress column.
	f = &overrideFake{issue: github.Issue{Number: 7, Title: "T", NodeID: "I_7"}}
	if _, err := newP(f).Claim(ctx, "7"); err != nil {
		t.Fatal(err)
	}
	want = []string{"ResolveProject(o,3)", "GetIssue(7)", "Viewer", "AssignIssue(7,octocat)", "AddItem(PVT_1,I_7)", "SetSingleSelect(PVT_1,ITEM_1,F_state,opt_doing2)"}
	if !slices.Equal(f.calls, want) {
		t.Errorf("claim calls = %v, want %v", f.calls, want)
	}

	// move reaches the overridden blocked column, preflight first.
	f = &overrideFake{issue: github.Issue{Number: 9, Title: "T", NodeID: "I_9"}}
	if _, err := newP(f).Move(ctx, "9", work.StateBlocked); err != nil {
		t.Fatal(err)
	}
	want = []string{"ResolveProject(o,3)", "GetIssue(9)", "AddItem(PVT_1,I_9)", "SetSingleSelect(PVT_1,ITEM_1,F_state,opt_paused)"}
	if !slices.Equal(f.calls, want) {
		t.Errorf("move calls = %v, want %v", f.calls, want)
	}
}
