package github

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// testClient points a Client at a test server for both REST and GraphQL.
func testClient(srv *httptest.Server) *Client {
	return &Client{
		http:       srv.Client(),
		token:      "test-token",
		restBase:   srv.URL,
		graphqlURL: srv.URL + "/graphql",
	}
}

func TestEndpoints(t *testing.T) {
	rest, gql := endpoints("")
	if rest != "https://api.github.com" || gql != "https://api.github.com/graphql" {
		t.Errorf("default endpoints = %q / %q", rest, gql)
	}
	rest, gql = endpoints("github.example.com")
	if rest != "https://github.example.com/api/v3" || gql != "https://github.example.com/api/graphql" {
		t.Errorf("enterprise endpoints = %q / %q", rest, gql)
	}
}

func TestTokenFromEnv(t *testing.T) {
	t.Setenv("GH_TOKEN", "from-env")
	tok, err := Token()
	if err != nil || tok != "from-env" {
		t.Fatalf("Token() = %q, %v", tok, err)
	}
}

func TestCreateIssueSendsAuthAndBody(t *testing.T) {
	var gotAuth, gotVersion, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotVersion = r.Header.Get("X-GitHub-Api-Version")
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"number":42,"html_url":"https://github.com/o/r/issues/42","node_id":"I_42"}`))
	}))
	defer srv.Close()

	iss, err := testClient(srv).CreateIssue(context.Background(), Repo{"o", "r"}, CreateIssueInput{
		Title: "boom", Body: "repro", Labels: []string{"bug"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if iss.Number != 42 || iss.NodeID != "I_42" {
		t.Errorf("issue = %+v", iss)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if gotVersion != "2022-11-28" {
		t.Errorf("api-version header = %q", gotVersion)
	}
	if gotPath != "/repos/o/r/issues" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(gotBody, `"title":"boom"`) || !strings.Contains(gotBody, `"bug"`) {
		t.Errorf("body = %q", gotBody)
	}
}

func TestListIssuesFiltersPullRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("labels"); got != "bug" {
			t.Errorf("labels query = %q", got)
		}
		_, _ = w.Write([]byte(`[
			{"number":1,"title":"real issue","labels":[{"name":"bug"}]},
			{"number":2,"title":"a PR","pull_request":{}}
		]`))
	}))
	defer srv.Close()

	issues, err := testClient(srv).ListIssues(context.Background(), Repo{"o", "r"}, ListOptions{State: "open", Labels: []string{"bug"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || issues[0].Number != 1 {
		t.Fatalf("expected only the non-PR issue, got %+v", issues)
	}
	if got := issues[0].LabelNames(); len(got) != 1 || got[0] != "bug" {
		t.Errorf("labels = %v", got)
	}
}

func TestCloseIssue(t *testing.T) {
	var method, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	if err := testClient(srv).CloseIssue(context.Background(), Repo{"o", "r"}, 7); err != nil {
		t.Fatal(err)
	}
	if method != "PATCH" || !strings.Contains(body, `"state":"closed"`) || !strings.Contains(body, `"state_reason":"completed"`) {
		t.Errorf("close = %s %s", method, body)
	}
}

func TestAPIErrorSurfacesMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer srv.Close()

	err := testClient(srv).REST(context.Background(), "GET", "/repos/o/r/issues/999", nil, nil)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Status != 404 || apiErr.Message != "Not Found" {
		t.Errorf("apiErr = %+v", apiErr)
	}
}

func TestResolveProjectParsesFieldsAndOptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"repositoryOwner":{"projectV2":{
			"id":"PVT_1","title":"Work","number":3,
			"fields":{"nodes":[
				{"id":"F_title","name":"Title"},
				{"id":"F_status","name":"Status","options":[
					{"id":"opt_todo","name":"Todo"},
					{"id":"opt_doing","name":"In Progress"}
				]}
			]}
		}}}}`))
	}))
	defer srv.Close()

	p, err := testClient(srv).ResolveProject(context.Background(), "octocat", 3)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "PVT_1" || p.Number != 3 {
		t.Errorf("project = %+v", p)
	}
	status, ok := p.Field("status") // case-insensitive
	if !ok {
		t.Fatal("Status field not found")
	}
	opt, ok := status.Option("In Progress")
	if !ok || opt.ID != "opt_doing" {
		t.Errorf("In Progress option = %+v (ok=%v)", opt, ok)
	}
}

func TestResolveProjectMissingIsClearError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"repositoryOwner":{"projectV2":null}}}`))
	}))
	defer srv.Close()

	_, err := testClient(srv).ResolveProject(context.Background(), "octocat", 99)
	if err == nil || !strings.Contains(err.Error(), "read:project") {
		t.Errorf("expected a read:project hint, got %v", err)
	}
}

func TestGraphQLSurfacesErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errors":[{"message":"Could not resolve to a node"}]}`))
	}))
	defer srv.Close()

	err := testClient(srv).GraphQL(context.Background(), "query{}", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "Could not resolve to a node") {
		t.Errorf("expected graphql error surfaced, got %v", err)
	}
}

func TestAddItemAndSetSingleSelect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var p struct {
			Query string `json:"query"`
		}
		_ = json.Unmarshal(b, &p)
		switch {
		case strings.Contains(p.Query, "addProjectV2ItemById"):
			_, _ = w.Write([]byte(`{"data":{"addProjectV2ItemById":{"item":{"id":"ITEM_1"}}}}`))
		default:
			_, _ = w.Write([]byte(`{"data":{"updateProjectV2ItemFieldValue":{"projectV2Item":{"id":"ITEM_1"}}}}`))
		}
	}))
	defer srv.Close()

	c := testClient(srv)
	item, err := c.AddItem(context.Background(), "PVT_1", "I_42")
	if err != nil || item != "ITEM_1" {
		t.Fatalf("AddItem = %q, %v", item, err)
	}
	if err := c.SetSingleSelect(context.Background(), "PVT_1", "ITEM_1", "F_status", "opt_doing"); err != nil {
		t.Fatalf("SetSingleSelect: %v", err)
	}
}
