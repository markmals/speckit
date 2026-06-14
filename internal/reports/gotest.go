package reports

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
)

// ParseGoTest parses the NDJSON event stream `go test -json` writes. Each test
// emits a terminal action (pass / fail / skip); the join identity is the test
// name as go reports it — the `func Test…` name, which is what the leading
// `// [scenario.id]` binding form attaches to.
//
// Only top-level test functions are recorded: a subtest's name carries a "/"
// and is rolled up into its parent's pass/fail, so it never appears as a
// separate (unbound) result. Skipped tests are omitted — a skip neither proves
// nor fails a scenario. Package-level and `output` events are ignored, and any
// non-JSON line (interleaved build output) is tolerated.
func ParseGoTest(data []byte) ([]Result, error) {
	type key struct{ pkg, test string }
	pass := map[key]bool{}
	var order []key

	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		var e struct {
			Action  string `json:"Action"`
			Package string `json:"Package"`
			Test    string `json:"Test"`
		}
		if json.Unmarshal(sc.Bytes(), &e) != nil {
			continue // tolerate interleaved non-JSON lines
		}
		if e.Test == "" || strings.Contains(e.Test, "/") {
			continue // package-level event, or a subtest (rolled into its parent)
		}
		switch e.Action {
		case "pass", "fail":
			k := key{e.Package, e.Test}
			if _, seen := pass[k]; !seen {
				order = append(order, k)
			}
			pass[k] = e.Action == "pass"
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	results := make([]Result, 0, len(order))
	for _, k := range order {
		results = append(results, Result{Suite: k.pkg, Name: k.test, Pass: pass[k]})
	}
	return results, nil
}
