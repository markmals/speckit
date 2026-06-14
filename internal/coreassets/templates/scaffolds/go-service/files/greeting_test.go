package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// [scenario.greeting.greet.hello]
func TestGreetingHandlerGreetsByName(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/greeting/Ada", nil)
	newMux().ServeHTTP(rec, req)
	if got := rec.Body.String(); got != "Hello, Ada!" {
		t.Errorf("body = %q, want %q", got, "Hello, Ada!")
	}
}

// [scenario.greeting.greet.defaults-to-world]
func TestGreetingDefaultsToWorld(t *testing.T) {
	if got := greeting(""); got != "Hello, world!" {
		t.Errorf("greeting(\"\") = %q, want %q", got, "Hello, world!")
	}
}
