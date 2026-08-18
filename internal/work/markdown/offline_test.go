package markdown

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/work"
)

// Every verb works with no network and no external binary. The dependency
// half is structural: this package's transitive (non-test) imports contain
// no exec, no network stack, and no GitHub client. The behavior half is a
// full five-verb round trip on a bare temp dir.
//
// [scenario.work.markdown.offline]
func TestEveryVerbWorksOffline(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	for _, dep := range strings.Fields(string(out)) {
		switch {
		case dep == "os/exec" || dep == "net" || strings.HasPrefix(dep, "net/"),
			strings.Contains(dep, "internal/github"):
			t.Errorf("offline provider depends on %s", dep)
		}
	}

	// All five verbs, no network, no binary, nothing but the temp dir.
	p := New(t.TempDir(), "WORK.md")
	ctx := context.Background()
	if items, err := p.Ready(ctx); err != nil || len(items) != 0 {
		t.Fatalf("ready = %+v, %v", items, err)
	}
	it, err := p.Create(ctx, work.CreateRequest{Title: "Offline item"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := p.Claim(ctx, it.ID); err != nil {
		t.Fatalf("claim: %v", err)
	}
	if _, err := p.Move(ctx, it.ID, work.StateDone); err != nil {
		t.Fatalf("move: %v", err)
	}
	items, err := p.List(ctx, "")
	if err != nil || len(items) != 1 || items[0].State != work.StateDone {
		t.Fatalf("list = %+v, %v", items, err)
	}
}
