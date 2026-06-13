package github

import (
	"context"
	"fmt"
)

// Ruleset is the subset of a repo ruleset SpecKit reads (to update in place rather
// than duplicate on re-run).
type Ruleset struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// GateRulesetOptions parameterizes the SpecKit branch-protection ruleset: which CI
// contexts must pass, how many approvals a PR needs, and the ruleset name.
type GateRulesetOptions struct {
	Name            string
	Contexts        []string
	RequiredReviews int
}

const defaultRulesetName = "speckit-gate"

// DefaultGateContexts are the required checks the scaffolded ci.yml exposes: the
// static `quality` job and the reusable spec gate, which surfaces as `verify / verify`.
var DefaultGateContexts = []string{"quality", "verify / verify"}

// gateRulesetPayload builds the rulesets API payload: a PR requirement, the
// required status checks, and a non-fast-forward (force-push) block on the default
// branch. Pure, so it's unit-testable. Mirrors docs/ci-gating.md.
func gateRulesetPayload(opts GateRulesetOptions) map[string]any {
	name := opts.Name
	if name == "" {
		name = defaultRulesetName
	}
	contexts := opts.Contexts
	if len(contexts) == 0 {
		contexts = DefaultGateContexts
	}
	checks := make([]map[string]any, len(contexts))
	for i, ctx := range contexts {
		checks[i] = map[string]any{"context": ctx}
	}
	return map[string]any{
		"name":        name,
		"target":      "branch",
		"enforcement": "active",
		"conditions": map[string]any{
			"ref_name": map[string]any{"include": []string{"~DEFAULT_BRANCH"}, "exclude": []string{}},
		},
		"rules": []map[string]any{
			{"type": "pull_request", "parameters": map[string]any{
				"required_approving_review_count":   opts.RequiredReviews,
				"dismiss_stale_reviews_on_push":     false,
				"require_code_owner_review":         false,
				"require_last_push_approval":        false,
				"required_review_thread_resolution": false,
			}},
			{"type": "required_status_checks", "parameters": map[string]any{
				"strict_required_status_checks_policy": true,
				"required_status_checks":               checks,
			}},
			{"type": "non_fast_forward"},
		},
	}
}

// ListRulesets returns the repo's OWN branch rulesets (includes_parents=false, so
// an org/enterprise ruleset of the same name isn't mistaken for a repo one and then
// PUT to the repo-scoped endpoint). Paginated so an existing gate ruleset past page
// one isn't missed (which would wrongly create a duplicate).
func (c *Client) ListRulesets(ctx context.Context, repo Repo) ([]Ruleset, error) {
	const perPage = 100
	var all []Ruleset
	for page := 1; ; page++ {
		path := fmt.Sprintf("/repos/%s/%s/rulesets?includes_parents=false&per_page=%d&page=%d", repo.Owner, repo.Name, perPage, page)
		var batch []Ruleset
		if err := c.REST(ctx, "GET", path, nil, &batch); err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < perPage {
			break
		}
	}
	return all, nil
}

// ProvisionGateRuleset creates the SpecKit gate ruleset, or updates it in place
// when one of the same name already exists (so the command is re-runnable). It
// returns the ruleset id and whether it was updated (vs created).
func (c *Client) ProvisionGateRuleset(ctx context.Context, repo Repo, opts GateRulesetOptions) (id int, updated bool, err error) {
	name := opts.Name
	if name == "" {
		name = defaultRulesetName
	}
	existing, err := c.ListRulesets(ctx, repo)
	if err != nil {
		return 0, false, err
	}
	payload := gateRulesetPayload(opts)
	var out Ruleset
	for _, r := range existing {
		if r.Name == name {
			path := fmt.Sprintf("/repos/%s/%s/rulesets/%d", repo.Owner, repo.Name, r.ID)
			if err := c.REST(ctx, "PUT", path, payload, &out); err != nil {
				return 0, false, err
			}
			return out.ID, true, nil
		}
	}
	path := fmt.Sprintf("/repos/%s/%s/rulesets", repo.Owner, repo.Name)
	if err := c.REST(ctx, "POST", path, payload, &out); err != nil {
		return 0, false, err
	}
	return out.ID, false, nil
}
