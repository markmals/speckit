---
name: go-cli-development
description: Use when writing or modifying the Go CLI target. Covers Cobra/Fang command surface + Bubble Tea (Elm/TEA-style state) + Bubbles + Lip Gloss + Huh + Glamour idioms, `database/sql` + glebarez/go-sqlite, and oapi-codegen (generated OpenAPI server/client) with `go test` / `go vet` / `go fmt`. Complementary to `implementing-a-spec` (process).
---

# Go CLI Development

This skill covers **how to write the CLI when its stack is Go**. The CLI is one target with a
choice of stack — Node, Rust, or Go — picked at setup; this is the **Go** stack. For the _workflow_
of implementing a spec, see `implementing-a-spec`. For _what to build_, see the spec.

The web app is the **reference implementation**. This is a **terminal client** — it mirrors web
behavior idiomatically in a TUI. Read the web realization for context; the spec is authoritative.

The TUI layer is the **[Charm](https://charm.sh/)** ecosystem — Bubble Tea · Bubbles · Lip Gloss ·
Huh · Glamour · Harmonica · Wish. The Rust CLI stack mirrors these with `charmed_rust`, so the two
stay at parity — see `rust-cli-development`.

## Stack at a glance

| Concern             | Choice                                                                          | First-party docs                                                                     |
| ------------------- | ------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| Language            | Go                                                                              | [go.dev](https://go.dev/)                                                            |
| CLI starter kit     | Fang (Charm wrapper over Cobra — styled help, errors, completions)              | [github.com/charmbracelet/fang](https://github.com/charmbracelet/fang)               |
| Arg parser          | Cobra                                                                           | [github.com/spf13/cobra](https://github.com/spf13/cobra)                             |
| State + view        | Bubble Tea (Elm/TEA-style model · message · update · view)                      | [github.com/charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea)     |
| Components          | Bubbles (lists, inputs, tables, spinners, viewports, file pickers)              | [github.com/charmbracelet/bubbles](https://github.com/charmbracelet/bubbles)         |
| Styling             | Lip Gloss                                                                       | [github.com/charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss)       |
| Forms / prompts     | Huh                                                                             | [github.com/charmbracelet/huh](https://github.com/charmbracelet/huh)                 |
| Markdown render     | Glamour                                                                         | [github.com/charmbracelet/glamour](https://github.com/charmbracelet/glamour)         |
| Animations          | Harmonica                                                                       | [github.com/charmbracelet/harmonica](https://github.com/charmbracelet/harmonica)     |
| SSH app framework   | Wish (+ Wishlist for an SSH directory)                                          | [github.com/charmbracelet/wish](https://github.com/charmbracelet/wish)               |
| Networking          | `net/http`                                                                      | [pkg.go.dev/net/http](https://pkg.go.dev/net/http)                                   |
| Serialization       | `encoding/json`                                                                 | [pkg.go.dev/encoding/json](https://pkg.go.dev/encoding/json)                         |
| Logging             | `log/slog`                                                                      | [pkg.go.dev/log/slog](https://pkg.go.dev/log/slog)                                   |
| Embedded assets     | `embed`                                                                         | [pkg.go.dev/embed](https://pkg.go.dev/embed)                                         |
| On-device database  | `database/sql` + glebarez/go-sqlite (pure-Go driver, no cgo)                    | [github.com/glebarez/go-sqlite](https://github.com/glebarez/go-sqlite)               |
| API server / client | oapi-codegen (generate both from the OpenAPI document — don't hand-roll either) | [github.com/oapi-codegen/oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) |
| Tests               | `testing`, `go test` (golden files in `testdata/`)                              | [pkg.go.dev/testing](https://pkg.go.dev/testing)                                     |
| Formatter           | `go fmt`                                                                        | [pkg.go.dev/cmd/gofmt](https://pkg.go.dev/cmd/gofmt)                                 |
| Static checks       | `go vet`                                                                        | [pkg.go.dev/cmd/vet](https://pkg.go.dev/cmd/vet)                                     |
| Package manager     | Go modules                                                                      | [go.dev/ref/mod](https://go.dev/ref/mod)                                             |
| Auth                | Clerk (token attached to the generated client)                                  | [clerk.com/docs](https://clerk.com/docs)                                             |

## The Client layer depends on the backend mode

How the CLI reaches its data is set by the project's backend (see `specs/ARCHITECTURE.md` →
"Backend modes"):

- **OpenAPI** — a typed client generated by **oapi-codegen** over `net/http`; never assemble requests
  by hand. In this mode the Go stack can also **host** the contract: an oapi-codegen-generated server
  over `net/http`.
- **Convex** — Convex's HTTP client over `net/http`.
- **No API** — no client at all; `database/sql` + **go-sqlite** is the source of truth.

In the remote modes, the local SQLite store is a local-first cache (not a second backend) and
identity flows through **Clerk**, whose token the client attaches.

## Idioms (read these before writing code)

### Cobra + Fang for the CLI surface

Define commands declaratively with Cobra; run the root through Fang for Charm-styled help, errors,
and completions. Keep command handlers thin — parse args, then hand off. No business logic in the
command layer.

```go
// SPEC: manual
func newRootCmd() *cobra.Command {
    root := &cobra.Command{Use: "app", Short: "..."}
    root.AddCommand(
        &cobra.Command{Use: "list", Short: "List items", RunE: runList},
        &cobra.Command{Use: "create <title>", Args: cobra.ExactArgs(1), RunE: runCreate},
    )
    return root
}

func main() {
    if err := fang.Execute(context.Background(), newRootCmd()); err != nil {
        os.Exit(1)
    }
}
```

### Bubble Tea: pure state + event loop, separate from rendering

Hold UI state in an **app-state model**. `Update` folds a message into new state; `View` renders a
string from that state via Lip Gloss / Bubbles. The **update layer is pure and testable** — no IO,
no terminal handles. This pure layer is the view-model analog and the **primary spec target**.
Bubble Tea owns the event pump and redraw scheduling; you write the model, the pure `Update`, and the
`View`.

```go
// SPEC: vm.items.list
type ItemsListModel struct {
    Items    []Item
    Selected int
}

type MoveMsg int

const (
    Up MoveMsg = iota
    Down
)

// Update is pure: a message in, new state out. No IO.
func (m ItemsListModel) Update(msg MoveMsg) ItemsListModel {
    switch msg {
    case Up:
        if m.Selected > 0 {
            m.Selected--
        }
    case Down:
        if m.Selected+1 < len(m.Items) {
            m.Selected++
        }
    }
    return m
}
```

- The reverse pointer lives on the **model** (the pure update layer), not the `View`.
- `View` reads state and renders with Lip Gloss / Bubbles; it holds no logic. Tag it `// SPEC: manual`
  — it has no cross-target contract.
- IO (`net/http` calls, `database/sql` reads, terminal events) lives at the edges and feeds messages
  into `Update`. The Bubble Tea `tea.Cmd` is how you schedule that IO without doing it inside `Update`.

### Tests at the logic layer

Standard Go test files (`*_test.go`) carrying `// SPEC: <id>`, with a `// [scenario.<id>]` comment
above each `func TestXxx` (Go test names can't hold dots or brackets, so the scenario sub-ID lives in
the comment that drift tooling greps). Table-driven where it reads well. The `specify verify <target>`
command runs the target's configured test command and joins scenarios to tests; format the test output
as `junit` so the engine can parse it.

```go
// SPEC: vm.items.list
package app

import "testing"

// [scenario.items.list.move-down]
func TestMovingDownAdvancesSelection(t *testing.T) {
    m := ItemsListModel{Items: []Item{{}, {}}, Selected: 0}
    m = m.Update(Down)
    if m.Selected != 1 {
        t.Fatalf("got selected=%d, want 1", m.Selected)
    }
}
```

The pure `Update` carries the behavioral coverage. To pin a rendered frame, write the `View()` output
to a **golden file** under `testdata/` and compare — gate regeneration behind a `-update` flag so the
golden is reviewed, never silently rewritten.

```go
// [scenario.items.list.renders-title]
func TestRendersSelectedTitle(t *testing.T) {
    m := ItemsListModel{Items: []Item{{Title: "First item"}}, Selected: 0}
    got := m.View()
    golden := filepath.Join("testdata", "items_list.golden")
    if *update {
        os.WriteFile(golden, []byte(got), 0o644)
    }
    want, _ := os.ReadFile(golden)
    if got != string(want) {
        t.Errorf("view mismatch:\n got: %q\nwant: %q", got, want)
    }
}
```

See `specs/CONVENTIONS.md` for the full spec ID format, scenario sub-ID scheme, and the
source-comment binding affordance (`// SPEC: <id>` reverse pointers).

### Error handling: return errors, don't panic

Functions that can fail return `error`; wrap with context using `fmt.Errorf("doing X: %w", err)` so
the chain is inspectable with `errors.Is` / `errors.As`. **No `panic` in shipped paths** — surface
the failure to the command layer, which prints it and exits non-zero (Fang renders it). Validate
external input at the boundary; trust internal contracts. Don't swallow an error into a silent
fallback that hides a real bug.

## File layout (within the target's source)

```
<source>/
├── main.go            ← entry point; wires Cobra/Fang → app → Bubble Tea program
├── cmd/               ← Cobra command definitions (thin; SPEC: manual)
├── app/               ← models + pure Update logic (the spec target)
├── ui/                ← Bubble Tea views: Lip Gloss styling + Bubbles widgets (dumb; SPEC: manual)
├── api/               ← oapi-codegen client/server wrapper (net/http transport)
├── store/             ← database/sql + go-sqlite, local-first cache
└── go.mod
```

## Verifying

- Run the target manually (e.g. `go run .`) for checks against a live terminal.
- `go test ./...` for the pure logic layer — this is where behavioral coverage lives.
- `specify verify <target>` runs the target's configured command, joins scenarios to tests, and
  reports pass/fail against the spec. Run it and read the output before claiming success.
- Golden files under `testdata/` to pin a rendered `View` string; regenerate only behind `-update`.
- `go vet ./...` and `go fmt ./...` (gofmt) must be clean before you declare done.

Run the verifying command **in this turn** and read its output before claiming success — see
`verification-before-completion`.

## When to invoke a more specific skill

- About to write tests? → `test-driven-development`
- About to claim work is done? → `verification-before-completion`
- Debugging something unexpected? → `systematic-debugging`
- Implementing a spec end-to-end? → `implementing-a-spec`

Related: the other two CLI stacks are `rust-cli-development` (the same Charm TUI in Rust via
charmed_rust) and `node-cli-development` (the Node/TS-Rest stack that owns the OpenAPI contract in
OpenAPI mode).

## Commit

Land focused, atomic commits as the work hits natural boundaries — typically per spec ID, per model +
its tests, or per cohesive refactor. Use the scoped-commits convention (`specify gate scope`).

Go-specific notes:

- **`go.mod` / `go.sum` bumps go alone.** A dependency change is its own commit
  (`chore: bump <module>`); don't bundle it with feature code.
- **Generated oapi-codegen output rides with the contract/codegen commit** that produced it — never
  commit it out of sync with the OpenAPI document.
- **Don't commit build outputs.** The compiled binary and any `dist/` are gitignored.
