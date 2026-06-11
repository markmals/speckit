// Throwaway spike: join real Vitest junit + Swift Testing output to spec
// scenarios, then compute the parity matrix. Demonstrates the D12 join (two
// strategies) and the D11 lying-deviation case.
//
//   - web   (Vitest): the report CARRIES the scenario id in the testcase name.
//   - apple (Swift Testing + custom traits + raw identifiers): the report does
//     NOT carry the id; it is read from the `.scenario("…")` trait in SOURCE and
//     joined to the report outcome by test identity (suite + raw-identifier name).
//
// Run from join/: `go run . ..`  (arg = spike root; default "..").
package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	scenarioRe = regexp.MustCompile(`scenario\.todo\.toggle\.[a-z\-]+`)
	declRe     = regexp.MustCompile(`<!--\s*id:\s*(scenario\.[a-z.\-]+)\s*-->`)
	deviateRe  = regexp.MustCompile(`// SPEC: (scenario\.[a-z.\-]+) \(deviates: ([^)]*)\)`)
	// Source-binding: a `.scenario("id")` trait immediately above a raw-identifier func.
	bindRe = regexp.MustCompile("@Test\\(\\.scenario\\(\"(scenario\\.[a-z.\\-]+)\"\\)\\)\\s*func `([^`]+)`")
)

func main() {
	root := ".."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	declared := parseSpec(filepath.Join(root, "spec/todo.toggle.md"))
	web, webErrs := parseWebJUnit(filepath.Join(root, "web/report.junit.xml"), declared)
	apple, appleErrs := parseAppleSourceBound(root, declared)
	deviations := parseDeviations(filepath.Join(root, "apple/Sources/Todo/Todo.swift"))

	scenarios := make([]string, 0, len(declared))
	for s := range declared {
		scenarios = append(scenarios, s)
	}
	sort.Strings(scenarios)

	fmt.Println("== D12 — scenario↔test join ==")
	joinFail := false
	for _, e := range webErrs {
		fmt.Printf("  ✗ JOIN ERROR (web, report-carried id): %s\n", e)
		joinFail = true
	}
	for _, e := range appleErrs {
		fmt.Printf("  ✗ JOIN ERROR (apple, source-bound): %s\n", e)
		joinFail = true
	}
	if !joinFail {
		fmt.Println("  (no join errors)")
	}

	fmt.Println("\n== Parity matrix ==")
	fmt.Printf("  %-26s  %-22s  %-22s\n", "scenario", "web", "apple")
	suspect := false
	for _, s := range scenarios {
		wc := webCell(web, s)
		ac := appleCell(apple, deviations, s)
		if ac.state == "SUSPECT" {
			suspect = true
		}
		fmt.Printf("  %-26s  %-22s  %-22s\n", strings.TrimPrefix(s, "scenario.todo.toggle."), render(wc), render(ac))
		if ac.detail != "" {
			fmt.Printf("  %-26s  %-22s  └─ %s\n", "", "", ac.detail)
		}
	}

	fmt.Println("\n== Gate verdict ==")
	gate := !joinFail && !suspect
	if joinFail {
		fmt.Println("  ✗ a dangling / unbound test reference means the join is dishonest (D12).")
	}
	if suspect {
		fmt.Println("  ✗ a (deviates:) marker is shadowing a FAILING test — the engine cannot")
		fmt.Println("    machine-verify that this divergence is intentional (D11).")
	}
	if gate {
		fmt.Println("  ✓ would gate green")
		os.Exit(0)
	}
	fmt.Println("\n  → parity cannot auto-green this PR; it needs a human sign-off.")
	os.Exit(1)
}

func parseSpec(path string) map[string]bool {
	b, err := os.ReadFile(path)
	must(err)
	out := map[string]bool{}
	for _, m := range declRe.FindAllStringSubmatch(string(b), -1) {
		out[m[1]] = true
	}
	return out
}

type junit struct {
	Cases []struct {
		Name    string    `xml:"name,attr"`
		Failure *struct{} `xml:"failure"`
	} `xml:"testsuite>testcase"`
}

// web: the scenario id is carried in the junit testcase name.
func parseWebJUnit(path string, declared map[string]bool) (map[string]bool, []string) {
	b, err := os.ReadFile(path)
	must(err)
	var j junit
	must(xml.Unmarshal(b, &j))
	pass := map[string]bool{}
	var errs []string
	for _, c := range j.Cases {
		s := scenarioRe.FindString(c.Name)
		if s == "" {
			errs = append(errs, "testcase has no scenario tag: "+c.Name)
			continue
		}
		if !declared[s] {
			errs = append(errs, "test references undeclared scenario "+s+" (renamed/typo)")
			continue
		}
		pass[s] = c.Failure == nil
	}
	return pass, errs
}

// apple: scenario id read from the `.scenario` trait in SOURCE, joined to the
// event-stream outcome by the test's raw-identifier name (its displayName).
func parseAppleSourceBound(root string, declared map[string]bool) (map[string]bool, []string) {
	bindings := parseSwiftBindings(filepath.Join(root, "apple/Tests")) // raw-id desc -> scenario id
	outcomes := parseAppleOutcomes(filepath.Join(root, "apple/events.ndjson")) // raw-id desc -> pass
	pass := map[string]bool{}
	var errs []string
	for desc, ok := range outcomes {
		s, bound := bindings[desc]
		if !bound {
			errs = append(errs, "test \""+desc+"\" has no .scenario(...) trait binding it to a scenario")
			continue
		}
		if !declared[s] {
			errs = append(errs, "test \""+desc+"\" binds to undeclared scenario "+s)
			continue
		}
		pass[s] = ok
	}
	return pass, errs
}

func parseSwiftBindings(dir string) map[string]string {
	out := map[string]string{}
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".swift") {
			return nil
		}
		b, _ := os.ReadFile(p)
		for _, m := range bindRe.FindAllStringSubmatch(string(b), -1) {
			out[m[2]] = m[1] // desc -> scenario id
		}
		return nil
	})
	return out
}

func parseAppleOutcomes(path string) map[string]bool {
	f, err := os.Open(path)
	must(err)
	defer f.Close()
	idToDesc := map[string]string{}
	failed := map[string]bool{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		var r struct {
			Kind    string `json:"kind"`
			Payload struct {
				Kind        string `json:"kind"`
				ID          string `json:"id"`
				TestID      string `json:"testID"`
				DisplayName string `json:"displayName"`
			} `json:"payload"`
		}
		if json.Unmarshal(sc.Bytes(), &r) != nil {
			continue
		}
		switch r.Kind {
		case "test":
			if r.Payload.DisplayName != "" {
				idToDesc[r.Payload.ID] = r.Payload.DisplayName
			}
		case "event":
			if r.Payload.Kind == "issueRecorded" {
				failed[r.Payload.TestID] = true
			}
		}
	}
	pass := map[string]bool{}
	for id, desc := range idToDesc {
		pass[desc] = !failed[id]
	}
	return pass
}

func parseDeviations(path string) map[string]string {
	b, err := os.ReadFile(path)
	must(err)
	out := map[string]string{}
	for _, m := range deviateRe.FindAllStringSubmatch(string(b), -1) {
		out[m[1]] = m[2]
	}
	return out
}

type cell struct{ state, detail string }

func webCell(pass map[string]bool, s string) cell {
	p, ok := pass[s]
	if !ok {
		return cell{state: "missing"}
	}
	if p {
		return cell{state: "conforming"}
	}
	return cell{state: "drifted"}
}

func appleCell(pass map[string]bool, dev map[string]string, s string) cell {
	reason, hasDev := dev[s]
	p, ok := pass[s]
	if !ok {
		return cell{state: "missing"}
	}
	switch {
	case hasDev && !p:
		return cell{state: "SUSPECT", detail: "marker claims “" + reason + "” but the test FAILED"}
	case hasDev && p:
		return cell{state: "declared-deviation", detail: "needs sign-off: “" + reason + "”"}
	case p:
		return cell{state: "conforming"}
	default:
		return cell{state: "drifted"}
	}
}

func render(c cell) string {
	switch c.state {
	case "conforming":
		return "✓ conforming"
	case "declared-deviation":
		return "~ declared-deviation"
	case "drifted":
		return "✗ drifted"
	case "missing":
		return "· missing"
	case "SUSPECT":
		return "⚠ SUSPECT"
	}
	return c.state
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "spike:", err)
		os.Exit(2)
	}
}
