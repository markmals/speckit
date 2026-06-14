package reports

import "testing"

// SPEC: story.engine.verify (scenario.engine.verify.normalizes-reports)
func TestParseGoTest(t *testing.T) {
	stream := `{"Action":"run","Package":"x"}
{"Action":"pass","Package":"x","Test":"TestParse"}
{"Action":"run","Package":"x","Test":"TestConvert"}
{"Action":"pass","Package":"x","Test":"TestConvert/case_a"}
{"Action":"fail","Package":"x","Test":"TestConvert/case_b"}
{"Action":"fail","Package":"x","Test":"TestConvert"}
{"Action":"skip","Package":"x","Test":"TestSkipped"}
this is not json — tolerated
{"Action":"fail","Package":"x"}
`
	results, err := ParseGoTest([]byte(stream))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 top-level results (subtests rolled up, skip omitted, package events ignored), got %d: %+v", len(results), results)
	}
	got := map[string]bool{}
	for _, r := range results {
		got[r.Name] = r.Pass
	}
	if !got["TestParse"] {
		t.Error("TestParse should be recorded as passing")
	}
	if pass, ok := got["TestConvert"]; !ok || pass {
		t.Error("TestConvert should be recorded as failing (a subtest failed)")
	}
	if _, ok := got["TestConvert/case_a"]; ok {
		t.Error("subtests must not appear as separate results (rolled into the parent)")
	}
	if _, ok := got["TestSkipped"]; ok {
		t.Error("skipped tests must be omitted")
	}
}
