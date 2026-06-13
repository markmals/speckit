package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGateRulesetPayloadDefaults(t *testing.T) {
	p := gateRulesetPayload(GateRulesetOptions{})
	if p["name"] != defaultRulesetName {
		t.Errorf("name = %v", p["name"])
	}
	// Round-trip through JSON to assert the shape the API receives.
	b, _ := json.Marshal(p)
	s := string(b)
	for _, want := range []string{
		`"target":"branch"`,
		`"enforcement":"active"`,
		`"~DEFAULT_BRANCH"`,
		`"context":"quality"`,
		`"context":"verify / verify"`,
		`"type":"non_fast_forward"`,
		`"type":"pull_request"`,
		`"type":"required_status_checks"`,
		`"strict_required_status_checks_policy":true`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("payload missing %s\n%s", want, s)
		}
	}
}

func TestGateRulesetCustomContexts(t *testing.T) {
	p := gateRulesetPayload(GateRulesetOptions{Contexts: []string{"only-this"}, RequiredReviews: 2})
	b, _ := json.Marshal(p)
	s := string(b)
	if !strings.Contains(s, `"context":"only-this"`) || strings.Contains(s, `"context":"quality"`) {
		t.Errorf("custom contexts not honored:\n%s", s)
	}
	if !strings.Contains(s, `"required_approving_review_count":2`) {
		t.Errorf("reviews not honored:\n%s", s)
	}
}

func TestProvisionRulesetCreatesWhenAbsent(t *testing.T) {
	var method string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		switch r.Method {
		case "GET":
			_, _ = w.Write([]byte(`[{"id":1,"name":"other"}]`)) // no speckit-gate
		case "POST":
			_, _ = w.Write([]byte(`{"id":99,"name":"speckit-gate"}`))
		}
	}))
	defer srv.Close()

	id, updated, err := testClient(srv).ProvisionGateRuleset(context.Background(), Repo{"o", "r"}, GateRulesetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if id != 99 || updated {
		t.Errorf("id=%d updated=%v (want 99, false)", id, updated)
	}
	if method != "POST" {
		t.Errorf("final method = %s, want POST", method)
	}
}

func TestProvisionRulesetUpdatesWhenPresent(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		switch r.Method {
		case "GET":
			_, _ = w.Write([]byte(`[{"id":7,"name":"speckit-gate"}]`)) // already exists
		case "PUT":
			_, _ = w.Write([]byte(`{"id":7,"name":"speckit-gate"}`))
		}
	}))
	defer srv.Close()

	id, updated, err := testClient(srv).ProvisionGateRuleset(context.Background(), Repo{"o", "r"}, GateRulesetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if id != 7 || !updated {
		t.Errorf("id=%d updated=%v (want 7, true)", id, updated)
	}
	if method != "PUT" || path != "/repos/o/r/rulesets/7" {
		t.Errorf("final = %s %s, want PUT /repos/o/r/rulesets/7", method, path)
	}
}
