package main

import (
	"testing"

	"github.com/markmals/speckit/internal/config"
)

func TestSyncPlan(t *testing.T) {
	d := config.Deploy{
		Kind:    "cloudflare-workers-ssr",
		CI:      map[string]string{"CLOUDFLARE_API_TOKEN": "op://Private/CF/token"},
		Runtime: map[string]string{"DATABASE_URL": "op://Private/db/url", "API_KEY": "op://Private/api/key"},
	}
	plan := syncPlan(d)
	if len(plan) != 3 {
		t.Fatalf("plan size = %d, want 3", len(plan))
	}
	// ci entries come first and target GitHub Actions.
	if plan[0].Dest != destGitHubActions || plan[0].Env != "CLOUDFLARE_API_TOKEN" {
		t.Errorf("plan[0] = %+v", plan[0])
	}
	// runtime entries follow, sorted, targeting the platform store.
	if plan[1].Env != "API_KEY" || plan[1].Dest != destPlatformRuntime {
		t.Errorf("plan[1] = %+v (want sorted runtime first)", plan[1])
	}
	if plan[2].Env != "DATABASE_URL" || plan[2].Dest != destPlatformRuntime {
		t.Errorf("plan[2] = %+v", plan[2])
	}
	// the plan carries references, never values.
	for _, op := range plan {
		if !config.IsOpRef(op.Ref) {
			t.Errorf("plan op %q ref is not an op:// reference: %q", op.Env, op.Ref)
		}
	}
}

func TestSyncPlanEmpty(t *testing.T) {
	if plan := syncPlan(config.Deploy{Kind: "github-pages-spa"}); len(plan) != 0 {
		t.Errorf("expected empty plan for a no-secret deploy, got %+v", plan)
	}
}
