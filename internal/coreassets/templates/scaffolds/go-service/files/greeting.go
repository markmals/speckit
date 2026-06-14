package main

import "net/http"

// newMux wires the service's routes.
func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /greeting/{name}", greetingHandler)
	return mux
}

// greeting returns the service's greeting for a name, defaulting to "world".
//
// SPEC: story.greeting.greet
func greeting(name string) string {
	if name == "" {
		name = "world"
	}
	return "Hello, " + name + "!"
}

// greetingHandler serves GET /greeting/{name}.
//
// SPEC: story.greeting.greet
func greetingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(greeting(r.PathValue("name"))))
}
