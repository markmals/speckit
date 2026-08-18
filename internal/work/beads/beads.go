// Package beads adapts `specify work` onto the Beads CLI (`bd`), mapping
// the canonical verbs onto bd's own primitives — ready, create, the atomic
// `update --claim`, status updates — rather than reimplementing them.
package beads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/markmals/speckit/internal/work"
)

// Canonical-state ↔ bd-status mapping, both directions:
//
//	ready       ↔ open
//	in-progress ↔ in_progress
//	blocked     ↔ blocked
//	done        ↔ closed
//
// bd statuses outside the table (deferred, pinned, hooked, custom) pass
// through as-is when read; Move and List reject states outside it.
var toBD = map[string]string{
	work.StateReady:      "open",
	work.StateInProgress: "in_progress",
	work.StateBlocked:    "blocked",
	work.StateDone:       "closed",
}

var fromBD = map[string]string{
	"open":        work.StateReady,
	"in_progress": work.StateInProgress,
	"blocked":     work.StateBlocked,
	"closed":      work.StateDone,
}

// runner executes bd with args and returns its stdout — injectable so the
// argv and JSON mapping are testable without bd installed.
type runner func(ctx context.Context, args ...string) ([]byte, error)

// Provider shells out to bd; the database is whatever `bd` discovers from
// the working directory (.beads/).
type Provider struct {
	run runner
}

var _ work.Provider = (*Provider)(nil)

// New verifies the Beads CLI is available.
func New() (*Provider, error) {
	if _, err := exec.LookPath("bd"); err != nil {
		return nil, fmt.Errorf("work provider \"beads\" needs the `bd` CLI on PATH — install it with `brew install beads` or `npm install -g @beads/bd` (github.com/steveyegge/beads): %w", err)
	}
	return &Provider{run: runBD}, nil
}

func (p *Provider) Name() string { return "beads" }

func (p *Provider) Ready(ctx context.Context) ([]work.Item, error) {
	return p.items(ctx, "ready", "--json", "--limit", "0")
}

func (p *Provider) Create(ctx context.Context, req work.CreateRequest) (work.Item, error) {
	args := []string{"create", req.Title, "--json"}
	if req.Type == work.TypeDefect {
		args = append(args, "--type", "bug")
	}
	if req.Spec != "" {
		args = append(args, "--spec-id", req.Spec)
	}
	out, err := p.run(ctx, args...)
	if err != nil {
		return work.Item{}, err
	}
	var iss bdIssue // create prints the one issue as an object
	if err := json.Unmarshal(out, &iss); err != nil {
		return work.Item{}, fmt.Errorf("bd create: parse output: %w", err)
	}
	return iss.item(), nil
}

// Claim uses bd's atomic compare-and-set claim: assignee to the caller,
// status to in_progress, idempotent when already claimed by the caller.
func (p *Provider) Claim(ctx context.Context, id string) (work.Item, error) {
	return p.update(ctx, id, "update", id, "--claim", "--json")
}

func (p *Provider) Move(ctx context.Context, id, state string) (work.Item, error) {
	status, ok := toBD[state]
	if !ok {
		return work.Item{}, unknownState("move", state)
	}
	return p.update(ctx, id, "update", id, "--status", status, "--json")
}

func (p *Provider) List(ctx context.Context, state string) ([]work.Item, error) {
	if state == "" {
		return p.items(ctx, "list", "--all", "--json", "--limit", "0")
	}
	status, ok := toBD[state]
	if !ok {
		return nil, unknownState("list", state)
	}
	return p.items(ctx, "list", "--status", status, "--json", "--limit", "0")
}

// update runs a bd mutation, which prints the updated issues as an array.
func (p *Provider) update(ctx context.Context, id string, args ...string) (work.Item, error) {
	out, err := p.run(ctx, args...)
	if err != nil {
		return work.Item{}, err
	}
	var issues []bdIssue
	if err := json.Unmarshal(out, &issues); err != nil {
		return work.Item{}, fmt.Errorf("bd %s: parse output: %w", args[0], err)
	}
	if len(issues) == 0 {
		return work.Item{}, fmt.Errorf("bd: no issue %q", id)
	}
	return issues[0].item(), nil
}

func (p *Provider) items(ctx context.Context, args ...string) ([]work.Item, error) {
	out, err := p.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	var issues []bdIssue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("bd %s: parse output: %w", args[0], err)
	}
	items := make([]work.Item, len(issues))
	for i, iss := range issues {
		items[i] = iss.item()
	}
	return items, nil
}

func unknownState(verb, state string) error {
	return fmt.Errorf("%s: unknown state %q (beads maps %s)", verb, state, strings.Join(work.CanonicalStates, ", "))
}

// bdIssue is the subset of bd's --json output the adapter reads.
type bdIssue struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	IssueType string `json:"issue_type"`
	SpecID    string `json:"spec_id"`
}

// item normalizes a bd issue. Types: bug ↔ defect, task ↔ "" (the zero
// type); other bd types (feature, epic, chore, …) pass through as-is.
func (i bdIssue) item() work.Item {
	it := work.Item{ID: i.ID, Title: i.Title, Spec: i.SpecID}
	if s, ok := fromBD[i.Status]; ok {
		it.State = s
	} else {
		it.State = i.Status
	}
	switch i.IssueType {
	case "bug":
		it.Type = work.TypeDefect
	case "task", "":
		it.Type = ""
	default:
		it.Type = i.IssueType
	}
	return it
}

func runBD(ctx context.Context, args ...string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, "bd", args...).Output()
	if err == nil {
		return out, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return out, fmt.Errorf("bd %s: %s", strings.Join(args, " "), bdErrorText(out, exit.Stderr))
	}
	return out, fmt.Errorf("bd %s: %w", strings.Join(args, " "), err)
}

// bdErrorText prefers bd's structured {"error": …} (written to stdout under
// --json), else stderr, else the bare exit.
func bdErrorText(stdout, stderr []byte) string {
	var v struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(stdout, &v) == nil && v.Error != "" {
		return v.Error
	}
	if s := strings.TrimSpace(string(stderr)); s != "" {
		return s
	}
	return "exited with an error"
}
