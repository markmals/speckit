package beads

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/work"
)

// fakeRunner replays canned stdout per call and records every argv.
type fakeRunner struct {
	calls [][]string
	out   []string
	fail  error
}

func (f *fakeRunner) run(_ context.Context, args ...string) ([]byte, error) {
	f.calls = append(f.calls, args)
	if f.fail != nil {
		return nil, f.fail
	}
	out := f.out[len(f.calls)-1]
	return []byte(out), nil
}

func provider(out ...string) (*Provider, *fakeRunner) {
	f := &fakeRunner{out: out}
	return &Provider{run: f.run}, f
}

func assertArgv(t *testing.T, got []string, want ...string) {
	t.Helper()
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("argv = %q, want %q", got, want)
	}
}

// The adapter delegates to bd's own primitives — its native ready predicate
// (which honors bd's typed dependencies), its atomic compare-and-set
// `update --claim`, its status updates — rather than reimplementing them:
// the exact argv per verb is the proof.
//
// [scenario.work.providers.beads-maps-native]
func TestRoundTrip(t *testing.T) {
	ctx := context.Background()
	p, f := provider(
		`{"id":"wk-a1","title":"Event log append path","status":"open","issue_type":"task","spec_id":"domain.event"}`,
		`[{"id":"wk-a1","title":"Event log append path","status":"open","issue_type":"task","spec_id":"domain.event"}]`,
		`[{"id":"wk-a1","title":"Event log append path","status":"in_progress","issue_type":"task","spec_id":"domain.event"}]`,
		`[{"id":"wk-a1","title":"Event log append path","status":"closed","issue_type":"task","spec_id":"domain.event"}]`,
		`[{"id":"wk-a1","title":"Event log append path","status":"closed","issue_type":"task","spec_id":"domain.event"}]`,
	)

	created, err := p.Create(ctx, work.CreateRequest{Title: "Event log append path", Spec: "domain.event"})
	if err != nil {
		t.Fatal(err)
	}
	if created.State != work.StateReady || created.Spec != "domain.event" || created.Type != "" {
		t.Errorf("created = %+v", created)
	}

	ready, err := p.Ready(ctx)
	if err != nil || len(ready) != 1 || ready[0].State != work.StateReady {
		t.Fatalf("ready = %+v, %v", ready, err)
	}

	claimed, err := p.Claim(ctx, "wk-a1")
	if err != nil || claimed.State != work.StateInProgress {
		t.Fatalf("claimed = %+v, %v", claimed, err)
	}

	moved, err := p.Move(ctx, "wk-a1", work.StateDone)
	if err != nil || moved.State != work.StateDone {
		t.Fatalf("moved = %+v, %v", moved, err)
	}

	listed, err := p.List(ctx, work.StateDone)
	if err != nil || len(listed) != 1 || listed[0].State != work.StateDone {
		t.Fatalf("listed = %+v, %v", listed, err)
	}

	assertArgv(t, f.calls[0], "create", "Event log append path", "--json", "--spec-id", "domain.event")
	assertArgv(t, f.calls[1], "ready", "--json", "--limit", "0")
	assertArgv(t, f.calls[2], "update", "wk-a1", "--claim", "--json")
	assertArgv(t, f.calls[3], "update", "wk-a1", "--status", "closed", "--json")
	assertArgv(t, f.calls[4], "list", "--status", "closed", "--json", "--limit", "0")
}

// [scenario.work.providers.beads-maps-native]
func TestCreateDefectMapsToBug(t *testing.T) {
	p, f := provider(`{"id":"wk-b2","title":"Crash","status":"open","issue_type":"bug"}`)
	it, err := p.Create(context.Background(), work.CreateRequest{Title: "Crash", Type: work.TypeDefect})
	if err != nil {
		t.Fatal(err)
	}
	assertArgv(t, f.calls[0], "create", "Crash", "--json", "--type", "bug")
	if it.Type != work.TypeDefect {
		t.Errorf("type = %q, want defect (bug ↔ defect)", it.Type)
	}
}

// [scenario.work.providers.beads-maps-native]
func TestListAllAndStatusPassThrough(t *testing.T) {
	p, f := provider(`[
		{"id":"wk-1","title":"A","status":"open","issue_type":"task"},
		{"id":"wk-2","title":"B","status":"deferred","issue_type":"epic"}
	]`)
	items, err := p.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	assertArgv(t, f.calls[0], "list", "--all", "--json", "--limit", "0")
	if items[0].State != work.StateReady {
		t.Errorf("open must map to ready, got %q", items[0].State)
	}
	if items[1].State != "deferred" || items[1].Type != "epic" {
		t.Errorf("unknown status/type must pass through as-is, got %+v", items[1])
	}
}

func TestUnknownStateRejected(t *testing.T) {
	p, f := provider()
	if _, err := p.Move(context.Background(), "wk-1", "someday"); err == nil || !strings.Contains(err.Error(), `"someday"`) {
		t.Errorf("move unknown state = %v", err)
	}
	if _, err := p.List(context.Background(), "someday"); err == nil || !strings.Contains(err.Error(), `"someday"`) {
		t.Errorf("list unknown state = %v", err)
	}
	if len(f.calls) != 0 {
		t.Errorf("bd must not run for a rejected state, ran %v", f.calls)
	}
}

func TestRunnerErrorSurfaces(t *testing.T) {
	p, _ := provider()
	p.run = func(context.Context, ...string) ([]byte, error) {
		return nil, fmt.Errorf("bd update wk-9: no issue found matching \"wk-9\"")
	}
	if _, err := p.Claim(context.Background(), "wk-9"); err == nil || !strings.Contains(err.Error(), "wk-9") {
		t.Errorf("claim error = %v", err)
	}
}

// [scenario.work.providers.beads-requires-bd]
func TestMissingBDNamesTheInstallStep(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // nothing on PATH → LookPath("bd") fails
	_, err := New()
	if err == nil {
		t.Fatal("New must fail without bd on PATH")
	}
	for _, want := range []string{"bd", "PATH", "brew install beads"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing-bd error %q does not name %q", err, want)
		}
	}
}

// The canonical-state ↔ bd-status table maps both directions for every
// canonical state: Move sends bd's own status vocabulary and the returned
// issue maps back to the canonical state it left as.
//
// [scenario.work.providers.beads-maps-native]
func TestStatusMappingBothDirections(t *testing.T) {
	ctx := context.Background()
	table := map[string]string{
		work.StateReady:      "open",
		work.StateInProgress: "in_progress",
		work.StateBlocked:    "blocked",
		work.StateDone:       "closed",
	}
	for state, status := range table {
		p, f := provider(fmt.Sprintf(`[{"id":"wk-1","title":"T","status":%q,"issue_type":"task"}]`, status))
		it, err := p.Move(ctx, "wk-1", state)
		if err != nil {
			t.Fatalf("move %s: %v", state, err)
		}
		assertArgv(t, f.calls[0], "update", "wk-1", "--status", status, "--json")
		if it.State != state {
			t.Errorf("bd status %q mapped back to %q, want %q", status, it.State, state)
		}
	}
}
