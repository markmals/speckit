package engine

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/markmals/speckit/internal/reports"
	"github.com/markmals/speckit/internal/specmodel"
)

// Binding ties a test (by its report identity) to the scenario it proves. Read
// from source (D15), never from the report.
type Binding struct {
	Scenario specmodel.SpecID `json:"scenario"`
	Identity string           `json:"identity"`
	File     string           `json:"file,omitempty"` // source file the binding was read from
	Line     int              `json:"line,omitempty"` // 1-based line of the binding, for CI annotations
}

// VerifyResult is the outcome of joining test results to declared scenarios.
// Green requires every declared scenario to pass with no D12 violations.
type VerifyResult struct {
	Passed     []specmodel.SpecID `json:"passed"`
	Failed     []specmodel.SpecID `json:"failed"`
	Unjoinable []specmodel.SpecID `json:"unjoinable"` // declared scenario, no bound test (D12)
	Dangling   []Binding          `json:"dangling"`   // binding to an undeclared scenario (D12)
	Unbound    []reports.Result   `json:"unbound"`    // ran but no scenario binding (D12)
}

// Green reports whether every declared scenario passed with no join violations.
//
// SPEC: story.engine.verify (scenario.engine.verify.green-writes-lock)
func (v VerifyResult) Green() bool {
	return len(v.Failed) == 0 && len(v.Unjoinable) == 0 && len(v.Dangling) == 0 && len(v.Unbound) == 0
}

// Join matches test results to declared scenarios via source bindings (D15) and
// reports the D12 failure modes: a declared scenario with no bound test is
// unjoinable, a binding to an undeclared scenario is dangling, and a test that
// ran with no binding is unbound.
//
// SPEC: story.engine.verify
func Join(declared map[specmodel.SpecID]bool, results []reports.Result, bindings []Binding) VerifyResult {
	var v VerifyResult
	bound := make([]bool, len(results))
	outcomes := map[specmodel.SpecID][]bool{} // scenario -> pass flags from its bound tests

	for _, b := range bindings {
		for i, r := range results {
			if identityMatch(r, b.Identity) {
				bound[i] = true // the test has a binding (even if to a bad scenario)
				if declared[b.Scenario] {
					outcomes[b.Scenario] = append(outcomes[b.Scenario], r.Pass)
				}
			}
		}
		if !declared[b.Scenario] {
			v.Dangling = append(v.Dangling, b) // D12: binding to an undeclared scenario
		}
	}

	for i, r := range results {
		if !bound[i] {
			v.Unbound = append(v.Unbound, r)
		}
	}

	for s := range declared {
		passes, ok := outcomes[s]
		if !ok || len(passes) == 0 {
			v.Unjoinable = append(v.Unjoinable, s)
			continue
		}
		allPass := true
		for _, p := range passes {
			if !p {
				allPass = false
				break
			}
		}
		if allPass {
			v.Passed = append(v.Passed, s)
		} else {
			v.Failed = append(v.Failed, s)
		}
	}

	sortIDs(v.Passed)
	sortIDs(v.Failed)
	sortIDs(v.Unjoinable)
	return v
}

func identityMatch(r reports.Result, identity string) bool {
	return r.Name == identity || strings.HasSuffix(r.Name, identity)
}

func sortIDs(ids []specmodel.SpecID) {
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
}

var (
	// swiftTestRe matches a Swift Testing `@Test(<traits>) func `name`` block. The
	// trait list may hold several `.scenario(...)` (a test can pin more than one);
	// `([^()]|\([^()]*\))*` tolerates the one level of nesting those traits add, so a
	// multi-trait `@Test(.scenario("a"), .scenario("b"))` is captured whole.
	swiftTestRe     = regexp.MustCompile("@Test\\(((?:[^()]|\\([^()]*\\))*)\\)\\s*func `([^`]+)`")
	swiftScenarioRe = regexp.MustCompile(`\.scenario\("(scenario\.[a-z0-9.\-]+)"\)`)
	vitestBindRe    = regexp.MustCompile(`it\("(\[(scenario\.[a-z0-9.\-]+)\][^"]*)"`)

	// the language-agnostic leading-comment form: `// [scenario.id]` on a line of
	// its own above a test declaration (Go `func Test…`, or a JS/TS `it/test(…)`).
	scenarioTagRe = regexp.MustCompile(`//\s*\[(scenario\.[a-z0-9.\-]+)\]`)
	goTestFuncRe  = regexp.MustCompile(`^\s*func\s+(Test[A-Za-z0-9_]*)\s*\(`)
	jsTestRe      = regexp.MustCompile("^\\s*(?:it|test)(?:\\.\\w+)?\\s*\\(\\s*[\"'`]([^\"'`]+)[\"'`]")
)

// bindingsInContent extracts scenario↔test bindings from one source file's
// text, tagging each with the file and its 1-based line (for CI annotations).
func bindingsInContent(path, src string) []Binding {
	var bs []Binding
	// scenarioGroup/identityGroup are the submatch indices for each binding form.
	add := func(re *regexp.Regexp, scenarioGroup, identityGroup int) {
		for _, m := range re.FindAllStringSubmatchIndex(src, -1) {
			bs = append(bs, Binding{
				Scenario: specmodel.SpecID(src[m[2*scenarioGroup]:m[2*scenarioGroup+1]]),
				Identity: src[m[2*identityGroup]:m[2*identityGroup+1]],
				File:     filepath.ToSlash(path),
				Line:     1 + strings.Count(src[:m[0]], "\n"),
			})
		}
	}
	bs = append(bs, swiftBindings(path, src)...) // @Test(.scenario("…")[, .scenario("…")]) func `…`
	add(vitestBindRe, 2, 1)                      // it("[scenario.…] …"
	bs = append(bs, leadingCommentBindings(path, src)...)
	return bs
}

// swiftBindings reads the Swift Testing trait form: every `.scenario("id")` in a
// test's `@Test(...)` traits binds that scenario to the test's raw-identifier name
// — a single test may pin several scenarios. The join identity is the function's
// backtick name, which Swift Testing reports as the test's display name.
func swiftBindings(path, src string) []Binding {
	var bs []Binding
	for _, m := range swiftTestRe.FindAllStringSubmatchIndex(src, -1) {
		traits, name := src[m[2]:m[3]], src[m[4]:m[5]]
		line := 1 + strings.Count(src[:m[0]], "\n")
		for _, sm := range swiftScenarioRe.FindAllStringSubmatch(traits, -1) {
			bs = append(bs, Binding{
				Scenario: specmodel.SpecID(sm[1]),
				Identity: name,
				File:     filepath.ToSlash(path),
				Line:     line,
			})
		}
	}
	return bs
}

// leadingCommentBindings reads the language-agnostic leading-comment form: one
// or more `// [scenario.<id>]` comment lines immediately above a test
// declaration. Each pending tag binds to the NEXT test's report identity — a Go
// `func Test…` name (how `go test` reports it) or a JS/TS `it/test(…)` title
// (how Vitest reports it). Blank lines and continuation `//` comments between the
// tag and the test are tolerated (multi-line comment blocks); any other code
// line clears the pending tags so a stray tag never binds a distant test.
func leadingCommentBindings(path, src string) []Binding {
	goSource := filepath.Ext(path) == ".go"
	type pending struct {
		id   specmodel.SpecID
		line int
	}
	var pend []pending
	var bs []Binding
	for i, line := range strings.Split(src, "\n") {
		if m := scenarioTagRe.FindStringSubmatch(line); m != nil {
			pend = append(pend, pending{specmodel.SpecID(m[1]), i + 1})
			continue
		}
		if t := strings.TrimSpace(line); t == "" || strings.HasPrefix(t, "//") {
			continue // blank or continuation comment — keep the pending tags
		}
		if len(pend) == 0 {
			continue
		}
		var identity string
		if goSource {
			if m := goTestFuncRe.FindStringSubmatch(line); m != nil {
				identity = m[1]
			}
		} else if m := jsTestRe.FindStringSubmatch(line); m != nil {
			identity = m[1]
		}
		if identity != "" {
			for _, p := range pend {
				bs = append(bs, Binding{Scenario: p.id, Identity: identity, File: filepath.ToSlash(path), Line: p.line})
			}
		}
		pend = pend[:0] // a code line consumes (or, if not a test, discards) the tags
	}
	return bs
}

// ScanBindings reads scenario↔test bindings from a target's test source (D15):
// Swift Testing `.scenario(...)` traits on raw-identifier funcs, and Vitest
// it() titles that lead with [scenario.id]. The binding's Identity is the test
// name as it appears in the runner's report. Generated and vendored
// directories (node_modules, .gitignore'd trees) are skipped — see
// walkSourceFiles.
//
// SPEC: story.engine.verify (scenario.engine.verify.source-bound-join)
func ScanBindings(dir string) ([]Binding, error) {
	var bindings []Binding
	err := walkSourceFiles(dir, sourceExts, func(path string, content []byte) {
		bindings = append(bindings, bindingsInContent(path, string(content))...)
	})
	return bindings, err
}
