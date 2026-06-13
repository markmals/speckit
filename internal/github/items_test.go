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

func TestListItemsParsesStatusAndPaginates(t *testing.T) {
	var calls int
	var sawStatusField string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		b, _ := io.ReadAll(r.Body)
		var p struct {
			Variables struct {
				StatusField string `json:"statusField"`
				After       string `json:"after"`
			} `json:"variables"`
		}
		_ = json.Unmarshal(b, &p)
		sawStatusField = p.Variables.StatusField
		if p.Variables.After == "" {
			// page 1: one item in Todo, hasNextPage
			_, _ = w.Write([]byte(`{"data":{"node":{"items":{
				"pageInfo":{"hasNextPage":true,"endCursor":"CUR"},
				"nodes":[
					{"id":"IT_1","status":{"name":"Todo"},"content":{"__typename":"Issue","number":11,"title":"first","url":"u1","state":"OPEN"}},
					{"id":"IT_draft","content":{"__typename":"DraftIssue","title":"a draft"}}
				]
			}}}}`))
			return
		}
		// page 2: one item with no status, no more pages
		_, _ = w.Write([]byte(`{"data":{"node":{"items":{
			"pageInfo":{"hasNextPage":false,"endCursor":""},
			"nodes":[{"id":"IT_2","content":{"__typename":"Issue","number":12,"title":"second","url":"u2","state":"OPEN"}}]
		}}}}`))
	}))
	defer srv.Close()

	items, err := testClient(srv).ListItems(context.Background(), "PVT_1", "Status")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Errorf("expected 2 paged calls, got %d", calls)
	}
	if sawStatusField != "Status" {
		t.Errorf("statusField var = %q", sawStatusField)
	}
	if len(items) != 2 { // the DraftIssue is skipped, not a phantom #0
		t.Fatalf("items = %d, want 2 (draft must be skipped)", len(items))
	}
	for _, it := range items {
		if it.Number == 0 {
			t.Errorf("phantom issue #0 leaked (draft/redacted not filtered): %+v", it)
		}
	}
	if items[0].Number != 11 || items[0].Status != "Todo" || items[0].ItemID != "IT_1" {
		t.Errorf("items[0] = %+v", items[0])
	}
	if items[1].Number != 12 || items[1].Status != "" { // null status field tolerated
		t.Errorf("items[1] = %+v, want #12 with empty status", items[1])
	}
}

func TestViewer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"viewer":{"login":"octocat"}}}`))
	}))
	defer srv.Close()
	login, err := testClient(srv).Viewer(context.Background())
	if err != nil || login != "octocat" {
		t.Fatalf("Viewer = %q, %v", login, err)
	}
}

func TestEnsureLabelTreats422AsSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"Validation Failed","errors":[{"code":"already_exists"}]}`))
	}))
	defer srv.Close()
	if err := testClient(srv).EnsureLabel(context.Background(), Repo{"o", "r"}, "discovered-from", "5319e7", "x"); err != nil {
		t.Errorf("already-exists 422 should be success, got %v", err)
	}
}

func TestEnsureLabelSurfacesRealValidationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		// e.g. an invalid color — NOT already_exists; must surface, not be swallowed.
		_, _ = w.Write([]byte(`{"message":"Validation Failed","errors":[{"resource":"Label","field":"color","code":"invalid"}]}`))
	}))
	defer srv.Close()
	if err := testClient(srv).EnsureLabel(context.Background(), Repo{"o", "r"}, "x", "not-a-hex", "y"); err == nil {
		t.Error("a non-already-exists 422 must surface, not be swallowed as idempotency")
	}
}

func TestAssignIssue(t *testing.T) {
	var body, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	if err := testClient(srv).AssignIssue(context.Background(), Repo{"o", "r"}, 5, []string{"octocat"}); err != nil {
		t.Fatal(err)
	}
	if path != "/repos/o/r/issues/5/assignees" || !strings.Contains(body, "octocat") {
		t.Errorf("assign = %s %s", path, body)
	}
}
