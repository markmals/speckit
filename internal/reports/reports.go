// Package reports normalizes the test output of each platform's runner into a
// common []Result keyed by test identity (D12). The engine's scenario join
// reads the scenario binding from source and joins it to these results by
// identity, so a report need not carry the scenario id.
//
// SPEC: story.engine.verify
package reports

// Result is a normalized per-test outcome. (Suite, Name) is the test's identity
// — the key the source-bound scenario join matches against.
type Result struct {
	Suite string `json:"suite"`
	Name  string `json:"name"`
	Pass  bool   `json:"pass"`
}
