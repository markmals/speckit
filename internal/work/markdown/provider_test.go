package markdown

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/work"
)

// [scenario.work.markdown.missing-file-is-empty]
func TestMissingFileIsAnEmptyBoard(t *testing.T) {
	p := New(t.TempDir(), "WORK.md")
	ctx := context.Background()
	for name, list := range map[string]func() ([]work.Item, error){
		"ready": func() ([]work.Item, error) { return p.Ready(ctx) },
		"list":  func() ([]work.Item, error) { return p.List(ctx, "") },
	} {
		items, err := list()
		if err != nil || len(items) != 0 {
			t.Errorf("%s on a missing file = %v, %v (want empty, nil)", name, items, err)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	root := t.TempDir()
	p := New(root, "WORK.md")
	ctx := context.Background()

	created, err := p.Create(ctx, work.CreateRequest{Title: "Event log append path", Spec: "domain.event"})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != "wk-1" || created.State != work.StateReady {
		t.Fatalf("created = %+v", created)
	}

	ready, err := p.Ready(ctx)
	if err != nil || len(ready) != 1 || ready[0].ID != "wk-1" {
		t.Fatalf("ready = %+v, %v", ready, err)
	}

	claimed, err := p.Claim(ctx, "wk-1")
	if err != nil || claimed.State != work.StateInProgress {
		t.Fatalf("claimed = %+v, %v", claimed, err)
	}
	if ready, _ := p.Ready(ctx); len(ready) != 0 {
		t.Errorf("claimed item still in ready: %+v", ready)
	}

	moved, err := p.Move(ctx, "wk-1", work.StateDone)
	if err != nil || moved.State != work.StateDone {
		t.Fatalf("moved = %+v, %v", moved, err)
	}

	all, err := p.List(ctx, "")
	if err != nil || len(all) != 1 {
		t.Fatalf("list all = %+v, %v", all, err)
	}
	got := all[0]
	if got.State != work.StateDone || got.Spec != "domain.event" || got.Title != "Event log append path" {
		t.Errorf("item after round trip = %+v", got)
	}

	src, err := os.ReadFile(filepath.Join(root, "WORK.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "- [x] `wk-1` Event log append path · spec: domain.event") {
		t.Errorf("done item not rendered [x] with its spec:\n%s", src)
	}
}

func TestDefectTypeSurvivesRoundTrip(t *testing.T) {
	p := New(t.TempDir(), "WORK.md")
	ctx := context.Background()
	if _, err := p.Create(ctx, work.CreateRequest{Title: "Crash on empty id", Type: work.TypeDefect, Spec: "story.x"}); err != nil {
		t.Fatal(err)
	}
	items, err := p.List(ctx, work.StateReady)
	if err != nil || len(items) != 1 {
		t.Fatalf("list = %+v, %v", items, err)
	}
	if items[0].Type != work.TypeDefect || items[0].Spec != "story.x" {
		t.Errorf("defect round trip = %+v", items[0])
	}
}

// [scenario.work.markdown.stable-short-ids]
func TestIDAllocationPastAGap(t *testing.T) {
	root := t.TempDir()
	// wk-1..wk-8 came and went; only wk-9 survived. The next ids continue
	// past the survivor — a removed item's id is never reallocated.
	if err := os.WriteFile(filepath.Join(root, "WORK.md"),
		[]byte("## Ready\n\n- [ ] `wk-9` Survivor\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := New(root, "WORK.md")
	it, err := p.Create(context.Background(), work.CreateRequest{Title: "Next"})
	if err != nil {
		t.Fatal(err)
	}
	if it.ID != "wk-10" {
		t.Errorf("id = %q, want wk-10 (never reuse a deleted id)", it.ID)
	}
	again, err := p.Create(context.Background(), work.CreateRequest{Title: "After"})
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != "wk-11" {
		t.Errorf("id = %q, want wk-11 (sequential next free number)", again.ID)
	}
}

func TestPreamblePreserved(t *testing.T) {
	root := t.TempDir()
	pre := "# Our backlog\n\nHand-written intro kept verbatim.\n"
	if err := os.WriteFile(filepath.Join(root, "WORK.md"),
		[]byte(pre+"\n## Ready\n\n- [ ] `wk-1` A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := New(root, "WORK.md")
	if _, err := p.Claim(context.Background(), "wk-1"); err != nil {
		t.Fatal(err)
	}
	src, _ := os.ReadFile(filepath.Join(root, "WORK.md"))
	if !strings.HasPrefix(string(src), "# Our backlog\n\nHand-written intro kept verbatim.\n") {
		t.Errorf("preamble not preserved:\n%s", src)
	}
}

// [scenario.work.markdown.sections-are-states]
func TestMoveToNonCanonicalStateCreatesItsSection(t *testing.T) {
	root := t.TempDir()
	p := New(root, "WORK.md")
	ctx := context.Background()
	if _, err := p.Create(ctx, work.CreateRequest{Title: "A"}); err != nil {
		t.Fatal(err)
	}
	it, err := p.Move(ctx, "wk-1", "waiting for review")
	if err != nil {
		t.Fatal(err)
	}
	if it.State != "waiting-for-review" {
		t.Errorf("state = %q", it.State)
	}
	src, _ := os.ReadFile(filepath.Join(root, "WORK.md"))
	if !strings.Contains(string(src), "## Waiting for review\n\n- [ ] `wk-1` A\n") {
		t.Errorf("non-canonical section missing:\n%s", src)
	}
	items, err := p.List(ctx, "waiting-for-review")
	if err != nil || len(items) != 1 {
		t.Errorf("list by non-canonical state = %+v, %v", items, err)
	}
}

func TestUnknownIDFailsNamingIt(t *testing.T) {
	p := New(t.TempDir(), "WORK.md")
	ctx := context.Background()
	if _, err := p.Claim(ctx, "wk-404"); err == nil || !strings.Contains(err.Error(), "wk-404") {
		t.Errorf("claim unknown id error = %v", err)
	}
	if _, err := p.Move(ctx, "wk-404", work.StateDone); err == nil || !strings.Contains(err.Error(), "wk-404") {
		t.Errorf("move unknown id error = %v", err)
	}
}

// [scenario.work.markdown.inline-spec-pointer]
func TestSpecPointerIsAnInlineSuffix(t *testing.T) {
	root := t.TempDir()
	p := New(root, "WORK.md")
	ctx := context.Background()
	if _, err := p.Create(ctx, work.CreateRequest{Title: "Write the parity docs", Spec: "story.engine.parity"}); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join(root, "WORK.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "- [ ] `wk-1` Write the parity docs · spec: story.engine.parity\n") {
		t.Errorf("spec pointer is not the documented inline suffix:\n%s", src)
	}
	// And the suffix survives the parse direction too.
	items, err := p.List(ctx, work.StateReady)
	if err != nil || len(items) != 1 || items[0].Spec != "story.engine.parity" {
		t.Errorf("inline spec pointer lost on parse: %+v, %v", items, err)
	}
}
