package markdown

import (
	"reflect"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/work"
)

const sample = "# Work\n" +
	"\n" +
	"Tracked by hand until now.\n" +
	"\n" +
	"## Ready\n" +
	"\n" +
	"- [ ] `wk-3` Implement handshake negotiation · spec: protocol.hive.handshake\n" +
	"\n" +
	"## In progress\n" +
	"\n" +
	"- [ ] `wk-1` Event log append path · spec: domain.event\n" +
	"\n" +
	"## Done\n" +
	"\n" +
	"- [x] `wk-2` Register the server target\n"

func TestParseSample(t *testing.T) {
	b := parse(sample)
	if b.preamble != "# Work\n\nTracked by hand until now." {
		t.Errorf("preamble = %q", b.preamble)
	}
	ready := b.list(work.StateReady)
	if len(ready) != 1 || ready[0].ID != "wk-3" || ready[0].Spec != "protocol.hive.handshake" {
		t.Errorf("ready = %+v", ready)
	}
	if ready[0].Title != "Implement handshake negotiation" {
		t.Errorf("title = %q", ready[0].Title)
	}
	done := b.list(work.StateDone)
	if len(done) != 1 || done[0].ID != "wk-2" || done[0].State != work.StateDone {
		t.Errorf("done = %+v", done)
	}
}

// parse → render → parse is the identity: the re-render is byte-identical
// and the parsed boards (preamble included) are structurally equal.
//
// [scenario.work.markdown.roundtrips]
func TestRenderParseStability(t *testing.T) {
	once := render(parse(sample))
	twice := render(parse(once))
	if once != twice {
		t.Errorf("render not stable:\n--- once ---\n%s\n--- twice ---\n%s", once, twice)
	}
	if !reflect.DeepEqual(parse(once), parse(twice)) {
		t.Errorf("parse → render → parse is not the identity:\n%#v\nvs\n%#v", parse(once), parse(twice))
	}
	// The original hand-written preamble is preserved through the trip.
	if !strings.Contains(once, "Tracked by hand until now.") {
		t.Errorf("preamble lost in round trip:\n%s", once)
	}
	// The empty canonical section is still rendered.
	if !strings.Contains(once, "## Blocked\n") {
		t.Errorf("empty canonical section dropped:\n%s", once)
	}
	// Done items keep their checked box.
	if !strings.Contains(once, "- [x] `wk-2` Register the server target") {
		t.Errorf("done item lost its [x]:\n%s", once)
	}
}

func TestParseItemFields(t *testing.T) {
	it, ok := parseItem("- [ ] `wk-9` Fix flake · in CI · spec: story.x · type: defect", "ready")
	if !ok {
		t.Fatal("item did not parse")
	}
	if it.Title != "Fix flake · in CI" { // " · " inside the title survives
		t.Errorf("title = %q", it.Title)
	}
	if it.Spec != "story.x" || it.Type != work.TypeDefect {
		t.Errorf("spec/type = %q/%q", it.Spec, it.Type)
	}
	if _, ok := parseItem("- plain bullet, no id", "ready"); ok {
		t.Error("a line without a backticked id must not parse as an item")
	}
	if it, ok := parseItem("- [ ] `wk-4` Plain task · type: task", "ready"); !ok || it.Type != "" {
		t.Errorf("type: task must normalize to the zero type, got %+v (ok=%v)", it, ok)
	}
}

func TestRenderOrdersCanonicalThenFirstSeen(t *testing.T) {
	src := "## Someday\n\n- [ ] `wk-5` Later\n\n## Done\n\n- [x] `wk-1` Old\n\n## Emptied\n\n## Ready\n\n- [ ] `wk-6` Now\n"
	out := render(parse(src))
	ready := strings.Index(out, "## Ready")
	done := strings.Index(out, "## Done")
	someday := strings.Index(out, "## Someday")
	if !(ready < done && done < someday) {
		t.Errorf("section order wrong:\n%s", out)
	}
	// An empty non-canonical section is dropped.
	if strings.Contains(out, "## Emptied") {
		t.Errorf("empty non-canonical section survived:\n%s", out)
	}
	// A file with no preamble gains the default one.
	if !strings.HasPrefix(out, "# Work\n") {
		t.Errorf("default preamble missing:\n%s", out)
	}
}

// [scenario.work.markdown.stable-short-ids]
func TestNextIDSkipsGaps(t *testing.T) {
	b := parse("## Ready\n\n- [ ] `wk-2` A\n- [ ] `wk-7` B\n\n## Done\n\n- [x] `wk-4` C\n")
	if got := b.nextID(); got != "wk-8" {
		t.Errorf("nextID = %q, want wk-8 (max+1, never a reused gap)", got)
	}
	if got := parse("").nextID(); got != "wk-1" {
		t.Errorf("nextID on empty board = %q", got)
	}
}

// Any `## Heading` IS a state named by its slug, and an item's state comes
// solely from the section it sits under — the checkbox is presentation.
//
// [scenario.work.markdown.sections-are-states]
func TestSectionsAreStates(t *testing.T) {
	src := "## Waiting for QA\n\n- [ ] `wk-1` Flaky spec\n\n## Done\n\n- [ ] `wk-2` Unchecked but done\n"
	b := parse(src)
	waiting := b.list("waiting-for-qa")
	if len(waiting) != 1 || waiting[0].ID != "wk-1" || waiting[0].State != "waiting-for-qa" {
		t.Errorf("arbitrary heading did not become its slug state: %+v", waiting)
	}
	// An unchecked box under ## Done is still done: section decides state.
	done := b.list(work.StateDone)
	if len(done) != 1 || done[0].ID != "wk-2" || done[0].State != work.StateDone {
		t.Errorf("state not decided solely by section: %+v", done)
	}
	if len(b.list(work.StateReady)) != 0 {
		t.Errorf("items leaked into a section they don't sit under")
	}
}

// [scenario.work.markdown.done-is-checked]
func TestDoneRendersCheckedEverythingElseUnchecked(t *testing.T) {
	b := parse("")
	b.add(work.StateReady, work.Item{ID: "wk-1", Title: "R"})
	b.add(work.StateInProgress, work.Item{ID: "wk-2", Title: "P"})
	b.add(work.StateBlocked, work.Item{ID: "wk-3", Title: "B"})
	b.add(work.StateDone, work.Item{ID: "wk-4", Title: "D"})
	b.add("someday", work.Item{ID: "wk-5", Title: "S"})
	out := render(b)
	for line, want := range map[string]string{
		"`wk-1` R": "- [ ]", "`wk-2` P": "- [ ]", "`wk-3` B": "- [ ]",
		"`wk-4` D": "- [x]", "`wk-5` S": "- [ ]",
	} {
		if !strings.Contains(out, want+" "+line) {
			t.Errorf("expected %q to render as %q:\n%s", line, want, out)
		}
	}
	if strings.Count(out, "- [x]") != 1 {
		t.Errorf("only the done item may render checked:\n%s", out)
	}
}
