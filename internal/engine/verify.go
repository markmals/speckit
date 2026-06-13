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
	swiftBindRe  = regexp.MustCompile("@Test\\(\\.scenario\\(\"(scenario\\.[a-z0-9.\\-]+)\"\\)\\)\\s*func `([^`]+)`")
	vitestBindRe = regexp.MustCompile(`it\("(\[(scenario\.[a-z0-9.\-]+)\][^"]*)"`)
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
	add(swiftBindRe, 1, 2)  // @Test(.scenario("…")) func `…`
	add(vitestBindRe, 2, 1) // it("[scenario.…] …"
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
