// Package github is a small REST + GraphQL client for Issues and Projects v2 —
// the backend of the github-projects work provider. It inherits gh's auth
// (`gh auth token`, or the GH_TOKEN / GITHUB_TOKEN env) so there is zero
// separate token plumbing, and contains its own GraphQL client so it depends on
// no external gh extension.
//
// Determinism line: this package owns all of SpecKit's network and GitHub state.
// It is imported ONLY by cmd/specify and the work providers — never by
// internal/engine, internal/specmodel, internal/reports, or internal/config. The
// engine stays offline and repo-local; work items are ephemeral coordination,
// never a verify/lock input. A board or issue call failing must never block a
// local `verify`.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Repo is an owner/name pair.
type Repo struct {
	Owner string
	Name  string
}

func (r Repo) String() string { return r.Owner + "/" + r.Name }

// Client talks to GitHub's REST and GraphQL APIs with a token inherited from gh.
type Client struct {
	http       *http.Client
	token      string
	restBase   string // e.g. https://api.github.com
	graphqlURL string // e.g. https://api.github.com/graphql
}

// New builds a Client, resolving the token from the environment or gh and the API
// endpoints from GH_HOST (defaulting to github.com). It returns a clear,
// actionable error when no token is available.
func New() (*Client, error) {
	tok, err := Token()
	if err != nil {
		return nil, err
	}
	restBase, graphqlURL := endpoints(os.Getenv("GH_HOST"))
	return &Client{
		http:       &http.Client{Timeout: 30 * time.Second},
		token:      tok,
		restBase:   restBase,
		graphqlURL: graphqlURL,
	}, nil
}

// endpoints maps a GH_HOST to the REST and GraphQL base URLs. The empty host (or
// github.com) is the public API; anything else is treated as GitHub Enterprise.
func endpoints(host string) (restBase, graphqlURL string) {
	if host == "" || host == "github.com" {
		return "https://api.github.com", "https://api.github.com/graphql"
	}
	return "https://" + host + "/api/v3", "https://" + host + "/api/graphql"
}

// Token resolves a GitHub token without ever persisting it: GH_TOKEN, then
// GITHUB_TOKEN, then `gh auth token` (so the binary inherits gh's auth when run as
// `gh specify …` or alongside an authenticated gh).
func Token() (string, error) {
	for _, env := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			return v, nil
		}
	}
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return "", fmt.Errorf("no GitHub token: set GH_TOKEN or run `gh auth login` (gh auth token: %w)", err)
	}
	tok := strings.TrimSpace(string(out))
	if tok == "" {
		return "", fmt.Errorf("no GitHub token: `gh auth token` returned empty — run `gh auth login`")
	}
	return tok, nil
}

// CurrentRepo resolves the repo to operate on: the GH_REPO env override first
// (so callers — and `gh` extensions — can target explicitly), else `gh repo
// view`. Note `gh repo view` resolves a fork's PARENT, so on a fork checkout set
// GH_REPO or pass --repo to hit your own fork rather than upstream.
func CurrentRepo() (Repo, error) {
	if v := strings.TrimSpace(os.Getenv("GH_REPO")); v != "" {
		return ParseRepo(v)
	}
	out, err := exec.Command("gh", "repo", "view", "--json", "owner,name").Output()
	if err != nil {
		return Repo{}, fmt.Errorf("not in a GitHub repo (set --repo / GH_REPO, or run from a repo): %w", err)
	}
	var v struct {
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return Repo{}, fmt.Errorf("parse `gh repo view`: %w", err)
	}
	if v.Owner.Login == "" || v.Name == "" {
		return Repo{}, fmt.Errorf("could not resolve owner/name from gh")
	}
	return Repo{Owner: v.Owner.Login, Name: v.Name}, nil
}

// ParseRepo parses an OWNER/REPO (or HOST/OWNER/REPO — the host is ignored here;
// endpoints come from GH_HOST) string into a Repo.
func ParseRepo(s string) (Repo, error) {
	parts := strings.Split(strings.TrimSpace(s), "/")
	switch len(parts) {
	case 3:
		parts = parts[1:] // drop host
		fallthrough
	case 2:
		if parts[0] == "" || parts[1] == "" {
			break
		}
		return Repo{Owner: parts[0], Name: parts[1]}, nil
	}
	return Repo{}, fmt.Errorf("invalid repo %q (want OWNER/REPO)", s)
}

// REST performs a REST call: method + path (e.g. "/repos/o/r/issues"), an optional
// JSON body, and an optional out to decode the 2xx response into. A non-2xx status
// is returned as an *APIError carrying GitHub's message.
func (c *Client) REST(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.restBase+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{Status: resp.StatusCode, Body: string(data), Message: ghMessage(data)}
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

// GraphQL executes a GraphQL query/mutation, decoding the `data` field into out.
// GraphQL is the only API for Projects v2 — Pillar 3 rides on this.
func (c *Client) GraphQL(ctx context.Context, query string, vars map[string]any, out any) error {
	payload := map[string]any{"query": query}
	if vars != nil {
		payload["variables"] = vars
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.graphqlURL, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{Status: resp.StatusCode, Body: string(data), Message: ghMessage(data)}
	}
	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}
	if len(envelope.Errors) > 0 {
		msgs := make([]string, len(envelope.Errors))
		for i, e := range envelope.Errors {
			msgs[i] = e.Message
		}
		return fmt.Errorf("graphql: %s", strings.Join(msgs, "; "))
	}
	if out != nil && len(envelope.Data) > 0 {
		return json.Unmarshal(envelope.Data, out)
	}
	return nil
}

// Viewer returns the authenticated user's login — used for the atomic self-claim
// (assign self + move the card in one step).
func (c *Client) Viewer(ctx context.Context) (string, error) {
	var resp struct {
		Viewer struct {
			Login string `json:"login"`
		} `json:"viewer"`
	}
	if err := c.GraphQL(ctx, `query{viewer{login}}`, nil, &resp); err != nil {
		return "", err
	}
	return resp.Viewer.Login, nil
}

// APIError is a non-2xx GitHub response.
type APIError struct {
	Status  int
	Message string
	Body    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("github API %d: %s", e.Status, e.Message)
	}
	return fmt.Sprintf("github API %d", e.Status)
}

// ghMessage best-effort extracts GitHub's error `message` field.
func ghMessage(data []byte) string {
	var v struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(data, &v)
	return v.Message
}
