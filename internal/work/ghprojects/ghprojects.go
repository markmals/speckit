// Package ghprojects adapts `specify work` onto a GitHub Projects v2
// board: items are issues, states are columns of a single-select status
// field. Every verb resolves and validates the project, field, and
// destination column BEFORE any mutation, so a bad column can never leave
// an issue half-claimed.
package ghprojects

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/markmals/speckit/internal/github"
	"github.com/markmals/speckit/internal/work"
)

// Client is the slice of the GitHub client the provider drives —
// injectable so the call sequence is testable without a network.
type Client interface {
	ResolveProject(ctx context.Context, owner string, number int) (github.Project, error)
	ListItems(ctx context.Context, projectID, statusField string) ([]github.Item, error)
	AddItem(ctx context.Context, projectID, contentNodeID string) (string, error)
	SetSingleSelect(ctx context.Context, projectID, itemID, fieldID, optionID string) error
	GetIssue(ctx context.Context, repo github.Repo, number int) (github.Issue, error)
	CreateIssue(ctx context.Context, repo github.Repo, in github.CreateIssueInput) (github.Issue, error)
	AssignIssue(ctx context.Context, repo github.Repo, number int, assignees []string) error
	Viewer(ctx context.Context) (string, error)
}

var _ Client = (*github.Client)(nil)

// Options addresses the board. Zero-valued fields take defaults: Owner the
// repo owner, StatusField "Status", Columns DefaultColumns (per-key —
// a partial map overrides only the states it names).
type Options struct {
	Project     int
	Owner       string
	StatusField string
	Columns     map[string]string // canonical state → column name
}

// DefaultColumns is the stock canonical-state → column map.
func DefaultColumns() map[string]string {
	return map[string]string{
		work.StateReady:      "Ready",
		work.StateInProgress: "In Progress",
		work.StateBlocked:    "On Hold",
		work.StateDone:       "Closed",
	}
}

// Provider drives one board in one repo.
type Provider struct {
	client Client
	repo   github.Repo
	opts   Options
}

var _ work.Provider = (*Provider)(nil)

// New validates the options and applies defaults.
func New(client Client, repo github.Repo, opts Options) (*Provider, error) {
	if opts.Project <= 0 {
		return nil, fmt.Errorf("github-projects: a positive project (board) number is required")
	}
	if opts.Owner == "" {
		opts.Owner = repo.Owner
	}
	if opts.StatusField == "" {
		opts.StatusField = "Status"
	}
	cols := DefaultColumns()
	for state, col := range opts.Columns {
		if !work.IsCanonical(state) {
			return nil, fmt.Errorf("github-projects: no state %q to map to column %q (states: %s)", state, col, strings.Join(work.CanonicalStates, ", "))
		}
		cols[state] = col
	}
	opts.Columns = cols
	return &Provider{client: client, repo: repo, opts: opts}, nil
}

func (p *Provider) Name() string { return "github-projects" }

// Ready lists the actionable column, keeping OPEN items only — one check
// that excludes closed issues and closed/merged PRs alike.
func (p *Provider) Ready(ctx context.Context) ([]work.Item, error) {
	return p.byColumn(ctx, p.opts.Columns[work.StateReady], true)
}

// List returns the whole board for "", else one state's column. The state
// vocabulary is fixed: only the four mapped states resolve.
func (p *Provider) List(ctx context.Context, state string) ([]work.Item, error) {
	if state == "" {
		return p.byColumn(ctx, "", false)
	}
	col, err := p.column(state)
	if err != nil {
		return nil, err
	}
	return p.byColumn(ctx, col, false)
}

func (p *Provider) byColumn(ctx context.Context, col string, openOnly bool) ([]work.Item, error) {
	proj, err := p.client.ResolveProject(ctx, p.opts.Owner, p.opts.Project)
	if err != nil {
		return nil, err
	}
	items, err := p.client.ListItems(ctx, proj.ID, p.opts.StatusField)
	if err != nil {
		return nil, err
	}
	var out []work.Item
	for _, it := range items {
		if col != "" && !strings.EqualFold(it.Status, col) {
			continue
		}
		if openOnly && !strings.EqualFold(it.State, "OPEN") {
			continue
		}
		out = append(out, p.item(it))
	}
	return out, nil
}

// Create opens an issue and lands it in the ready column. The column is
// resolved before the issue exists, so a bad board can never leave an
// orphan issue off the board.
func (p *Provider) Create(ctx context.Context, req work.CreateRequest) (work.Item, error) {
	proj, err := p.client.ResolveProject(ctx, p.opts.Owner, p.opts.Project)
	if err != nil {
		return work.Item{}, err
	}
	readyCol := p.opts.Columns[work.StateReady]
	field, opt, err := resolveColumn(proj, p.opts.StatusField, readyCol)
	if err != nil {
		return work.Item{}, err
	}
	in := github.CreateIssueInput{Title: req.Title}
	// The shapes the defect-intake form stamps: a defect is the Bug issue
	// type plus the portable `bug` label; everything else is a Task.
	typ := ""
	if req.Type == work.TypeDefect {
		in.Type = "Bug"
		in.Labels = []string{"bug"}
		typ = work.TypeDefect
	} else {
		in.Type = "Task"
	}
	if req.Spec != "" {
		in.Body = "Spec: " + req.Spec
	}
	iss, err := p.client.CreateIssue(ctx, p.repo, in)
	if err != nil {
		return work.Item{}, err
	}
	if err := p.moveCard(ctx, proj, field, opt, iss.NodeID); err != nil {
		return work.Item{}, fmt.Errorf("opened #%d but failed to land it in %q: %w", iss.Number, readyCol, err)
	}
	return work.Item{ID: strconv.Itoa(iss.Number), Title: iss.Title, State: work.StateReady, Type: typ, Spec: req.Spec, URL: iss.HTMLURL}, nil
}

// Claim assigns the viewer and moves the card to the in-progress column,
// refusing when the issue is already assigned to someone else (assignment
// has no native compare-and-swap; re-claiming your own is fine).
func (p *Provider) Claim(ctx context.Context, id string) (work.Item, error) {
	number, err := issueNumber(id)
	if err != nil {
		return work.Item{}, fmt.Errorf("claim: %w", err)
	}
	col := p.opts.Columns[work.StateInProgress]
	proj, err := p.client.ResolveProject(ctx, p.opts.Owner, p.opts.Project)
	if err != nil {
		return work.Item{}, err
	}
	field, opt, err := resolveColumn(proj, p.opts.StatusField, col)
	if err != nil {
		return work.Item{}, err
	}
	iss, err := p.client.GetIssue(ctx, p.repo, number)
	if err != nil {
		return work.Item{}, err
	}
	login, err := p.client.Viewer(ctx)
	if err != nil {
		return work.Item{}, err
	}
	if other := assignedToOther(iss, login); other != "" {
		return work.Item{}, fmt.Errorf("#%d is already assigned to @%s — not claiming (reassign on GitHub to override)", number, other)
	}
	if err := p.client.AssignIssue(ctx, p.repo, number, []string{login}); err != nil {
		return work.Item{}, err
	}
	if err := p.moveCard(ctx, proj, field, opt, iss.NodeID); err != nil {
		return work.Item{}, fmt.Errorf("assigned #%d to @%s but failed to move the card to %q: %w", number, login, col, err)
	}
	return work.Item{ID: strconv.Itoa(number), Title: iss.Title, State: work.StateInProgress, URL: iss.HTMLURL}, nil
}

// Move puts an issue's card in a state's column.
func (p *Provider) Move(ctx context.Context, id, state string) (work.Item, error) {
	number, err := issueNumber(id)
	if err != nil {
		return work.Item{}, fmt.Errorf("move: %w", err)
	}
	col, err := p.column(state)
	if err != nil {
		return work.Item{}, err
	}
	proj, err := p.client.ResolveProject(ctx, p.opts.Owner, p.opts.Project)
	if err != nil {
		return work.Item{}, err
	}
	field, opt, err := resolveColumn(proj, p.opts.StatusField, col)
	if err != nil {
		return work.Item{}, err
	}
	iss, err := p.client.GetIssue(ctx, p.repo, number)
	if err != nil {
		return work.Item{}, err
	}
	if err := p.moveCard(ctx, proj, field, opt, iss.NodeID); err != nil {
		return work.Item{}, err
	}
	return work.Item{ID: strconv.Itoa(number), Title: iss.Title, State: state, URL: iss.HTMLURL}, nil
}

// column maps a canonical state to its board column; anything else is
// outside this provider's fixed vocabulary.
func (p *Provider) column(state string) (string, error) {
	col, ok := p.opts.Columns[state]
	if !ok {
		return "", fmt.Errorf("unknown state %q (the board maps %s)", state, strings.Join(work.CanonicalStates, ", "))
	}
	return col, nil
}

// state maps a column name back to its canonical state; a column outside
// the map (e.g. Backlog) passes through as-is rather than being coerced.
func (p *Provider) state(column string) string {
	for st, col := range p.opts.Columns {
		if strings.EqualFold(col, column) {
			return st
		}
	}
	return column
}

func (p *Provider) item(it github.Item) work.Item {
	return work.Item{ID: strconv.Itoa(it.Number), Title: it.Title, State: p.state(it.Status), URL: it.URL}
}

// moveCard ensures the issue is on the board (AddItem is idempotent) and
// sets its status to the resolved column.
func (p *Provider) moveCard(ctx context.Context, proj github.Project, field github.Field, opt github.FieldOption, nodeID string) error {
	itemID, err := p.client.AddItem(ctx, proj.ID, nodeID)
	if err != nil {
		return err
	}
	return p.client.SetSingleSelect(ctx, proj.ID, itemID, field.ID, opt.ID)
}

// resolveColumn looks up the status field and the destination column on a
// resolved project — pure and side-effect-free, so every verb can validate
// before mutating anything.
func resolveColumn(proj github.Project, statusField, column string) (github.Field, github.FieldOption, error) {
	field, ok := proj.Field(statusField)
	if !ok {
		return github.Field{}, github.FieldOption{}, fmt.Errorf("project %q has no field %q", proj.Title, statusField)
	}
	opt, ok := field.Option(column)
	if !ok {
		return github.Field{}, github.FieldOption{}, fmt.Errorf("field %q has no column %q (have: %s)", field.Name, column, optionNames(field))
	}
	return field, opt, nil
}

// assignedToOther returns the login of an assignee that isn't the viewer,
// or "" if unassigned or only self-assigned.
func assignedToOther(iss github.Issue, login string) string {
	for _, a := range iss.Assignees {
		if a.Login != login {
			return a.Login
		}
	}
	return ""
}

func issueNumber(id string) (int, error) {
	n, err := strconv.Atoi(strings.TrimPrefix(id, "#"))
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%q is not an issue number", id)
	}
	return n, nil
}

func optionNames(f github.Field) string {
	names := make([]string, len(f.Options))
	for i, o := range f.Options {
		names[i] = o.Name
	}
	return strings.Join(names, ", ")
}
