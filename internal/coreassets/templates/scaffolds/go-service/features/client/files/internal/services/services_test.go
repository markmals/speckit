package services

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeServer spins up an httptest.Server serving canned JSON keyed by request
// path; any unmocked path returns 501, so an unexpected upstream call fails the
// test loudly instead of hitting the network. This is the harness for unit-testing
// a client against a real *http transport without real I/O. (Untagged — out of
// scenario scope under the target's `bindings: scoped`, but it runs + must pass.)
func fakeServer(t *testing.T, routes map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotImplemented)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestClientGetThing(t *testing.T) {
	srv := fakeServer(t, map[string]string{"/things/42": `{"id":"42","name":"Towel"}`})
	got, err := New(srv.URL, "test-key").GetThing(context.Background(), "42")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Towel" {
		t.Errorf("Name = %q, want Towel", got.Name)
	}
}

func TestClientNon2xxIsAPIError(t *testing.T) {
	srv := fakeServer(t, map[string]string{}) // every path 501
	_, err := New(srv.URL, "").GetThing(context.Background(), "missing")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %T %v", err, err)
	}
	if apiErr.Status != http.StatusNotImplemented {
		t.Errorf("Status = %d, want 501", apiErr.Status)
	}
}
