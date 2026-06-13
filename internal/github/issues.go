package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Issue is the subset of a GitHub issue SpecKit cares about.
type Issue struct {
	ID        int     `json:"id"`     // database id — for sub-issue edges (sub_issue_id)
	Number    int     `json:"number"` // the human #N
	Title     string  `json:"title"`
	Body      string  `json:"body"`
	State     string  `json:"state"`
	HTMLURL   string  `json:"html_url"`
	NodeID    string  `json:"node_id"` // for project AddItem / GraphQL
	Labels    []Label `json:"labels"`
	Assignees []User  `json:"assignees"`
	// PullRequest is non-nil when this "issue" is actually a PR — the REST issues
	// list conflates them, so callers filter on IsPR().
	PullRequest *struct{} `json:"pull_request,omitempty"`
}

// Label is an issue label (only the name is used).
type Label struct {
	Name string `json:"name"`
}

// User is the subset of a GitHub user SpecKit reads.
type User struct {
	Login string `json:"login"`
}

// IsPR reports whether this issue is really a pull request.
func (i Issue) IsPR() bool { return i.PullRequest != nil }

// LabelNames returns the issue's label names.
func (i Issue) LabelNames() []string {
	names := make([]string, len(i.Labels))
	for j, l := range i.Labels {
		names[j] = l.Name
	}
	return names
}

// CreateIssueInput is the payload for CreateIssue. Type sets the org Issue Type
// (Bug/Feature/Task/Epic) where the org has them; Labels are the portable
// fallback that works on personal repos too.
type CreateIssueInput struct {
	Title     string   `json:"title"`
	Body      string   `json:"body,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	Assignees []string `json:"assignees,omitempty"`
	Type      string   `json:"type,omitempty"`
}

// CreateIssue opens an issue. This supersedes the agent-driven `gh issue create`
// in the taskstoissues prompt with a typed, in-binary call.
func (c *Client) CreateIssue(ctx context.Context, repo Repo, in CreateIssueInput) (Issue, error) {
	var out Issue
	path := fmt.Sprintf("/repos/%s/%s/issues", repo.Owner, repo.Name)
	if err := c.REST(ctx, "POST", path, in, &out); err != nil {
		return Issue{}, err
	}
	return out, nil
}

// ListOptions filters ListIssues. State is open|closed|all (default open); Labels
// narrows to issues carrying ALL of them.
type ListOptions struct {
	State  string
	Labels []string
}

// ListIssues lists a repo's issues (pull requests filtered out), following
// pagination so a repo with more than one page isn't silently truncated.
func (c *Client) ListIssues(ctx context.Context, repo Repo, opts ListOptions) ([]Issue, error) {
	const perPage = 100
	var issues []Issue
	for page := 1; ; page++ {
		q := url.Values{}
		if opts.State != "" {
			q.Set("state", opts.State)
		}
		if len(opts.Labels) > 0 {
			q.Set("labels", strings.Join(opts.Labels, ","))
		}
		q.Set("per_page", strconv.Itoa(perPage))
		q.Set("page", strconv.Itoa(page))
		path := fmt.Sprintf("/repos/%s/%s/issues?%s", repo.Owner, repo.Name, q.Encode())

		var raw []Issue
		if err := c.REST(ctx, "GET", path, nil, &raw); err != nil {
			return nil, err
		}
		for _, i := range raw {
			if i.IsPR() {
				continue
			}
			issues = append(issues, i)
		}
		if len(raw) < perPage {
			break
		}
	}
	return issues, nil
}

// CloseIssue closes an issue as completed — the close-on-green step (the lock is
// the proof; the issue was just intake).
func (c *Client) CloseIssue(ctx context.Context, repo Repo, number int) error {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d", repo.Owner, repo.Name, number)
	body := map[string]string{"state": "closed", "state_reason": "completed"}
	return c.REST(ctx, "PATCH", path, body, nil)
}

// GetIssue fetches one issue (for its node id, database id, and current labels).
func (c *Client) GetIssue(ctx context.Context, repo Repo, number int) (Issue, error) {
	var out Issue
	path := fmt.Sprintf("/repos/%s/%s/issues/%d", repo.Owner, repo.Name, number)
	if err := c.REST(ctx, "GET", path, nil, &out); err != nil {
		return Issue{}, err
	}
	return out, nil
}

// AssignIssue adds assignees to an issue. The atomic-claim step assigns the viewer.
func (c *Client) AssignIssue(ctx context.Context, repo Repo, number int, assignees []string) error {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/assignees", repo.Owner, repo.Name, number)
	return c.REST(ctx, "POST", path, map[string][]string{"assignees": assignees}, nil)
}

// EnsureLabel creates a label if it doesn't already exist. It swallows ONLY the
// already-exists 422 (so `discover` can rely on the provenance label being
// present); any other 422 — an invalid color, an empty name, a rate-limit — is a
// real failure and surfaces, rather than being masked as idempotency.
func (c *Client) EnsureLabel(ctx context.Context, repo Repo, name, color, description string) error {
	path := fmt.Sprintf("/repos/%s/%s/labels", repo.Owner, repo.Name)
	body := map[string]string{"name": name, "color": color, "description": description}
	err := c.REST(ctx, "POST", path, body, nil)
	if apiErr, ok := err.(*APIError); ok && apiErr.Status == 422 && labelAlreadyExists(apiErr.Body) {
		return nil
	}
	return err
}

// labelAlreadyExists reports whether a 422 body is GitHub's "label already exists"
// (as opposed to a genuine validation failure).
func labelAlreadyExists(body string) bool {
	var v struct {
		Errors []struct {
			Code string `json:"code"`
		} `json:"errors"`
	}
	if json.Unmarshal([]byte(body), &v) != nil {
		return false
	}
	for _, e := range v.Errors {
		if e.Code == "already_exists" {
			return true
		}
	}
	return false
}

// AddSubIssue attaches child as a sub-issue of parent (epic → subtask hierarchy).
// child is the sub-issue's numeric id (not its number) — fetch it via the issue's
// id field. Used by the Projects/epics surface (Pillar 3).
func (c *Client) AddSubIssue(ctx context.Context, repo Repo, parent, childID int) error {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/sub_issues", repo.Owner, repo.Name, parent)
	return c.REST(ctx, "POST", path, map[string]int{"sub_issue_id": childID}, nil)
}
