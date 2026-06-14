package specmodel

import "testing"

func TestParseFrontmatter(t *testing.T) {
	fm, ok := ParseFrontmatter("---\nid: story.x.y\nkind: story\ndepends-on: [a, b]\nstatus: draft\n---\nbody\n")
	if !ok {
		t.Fatal("expected ok")
	}
	if fm.ID != "story.x.y" || fm.Kind != KindStory || fm.Status != "draft" {
		t.Errorf("got %+v", fm)
	}
	if len(fm.DependsOn) != 2 || fm.DependsOn[0] != "a" || fm.DependsOn[1] != "b" {
		t.Errorf("depends-on: %v", fm.DependsOn)
	}
}

func TestParseFrontmatterNotASpec(t *testing.T) {
	if _, ok := ParseFrontmatter("# README\n\nno frontmatter here"); ok {
		t.Error("a file without frontmatter must not parse as a spec")
	}
	if _, ok := ParseFrontmatter("---\nkind: story\n---\nno id"); ok {
		t.Error("frontmatter without an id must not parse as a spec")
	}
}

func TestParseScenarios(t *testing.T) {
	md := "## Scenario 1: a\n\n<!-- id: scenario.x.y.a -->\n- Given\n\n## Scenario 2: b\n- Given\n"
	sc := parseScenarios(md)
	if len(sc) != 2 {
		t.Fatalf("expected 2 scenarios, got %d", len(sc))
	}
	if sc[0].SubID != "scenario.x.y.a" {
		t.Errorf("scenario 1 sub-id = %q", sc[0].SubID)
	}
	if sc[1].SubID != "" {
		t.Errorf("scenario 2 should have no sub-id, got %q", sc[1].SubID)
	}
}

// Scenarios are also recognized when nested as `### Scenario N:` under an
// `## Acceptance Criteria` heading (trove's form); the same-or-shallower break
// keeps scenario 1 from absorbing scenario 2's sub-id.
func TestParseScenariosNestedHeading(t *testing.T) {
	md := "## Acceptance Criteria\n\n### Scenario 1: convert\n\n<!-- id: scenario.x.y.convert -->\n- Given\n\n### Scenario 2: fail\n\n<!-- id: scenario.x.y.fail -->\n- Given\n"
	sc := parseScenarios(md)
	if len(sc) != 2 {
		t.Fatalf("expected 2 nested scenarios, got %d: %+v", len(sc), sc)
	}
	if sc[0].SubID != "scenario.x.y.convert" || sc[1].SubID != "scenario.x.y.fail" {
		t.Errorf("nested ### Scenario sub-ids not parsed: %+v", sc)
	}
}
