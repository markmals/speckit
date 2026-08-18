# Adding a report format

The engine reads a target's test outcomes from a **report file** in one of the
formats named by `config.Formats` — today `junit`, `swift`, and `gotest`.
SpecKit deliberately does not model test *runners*, only their reports: any
runner that can write one of these formats works today, and teaching the engine
a genuinely new format is a small, contained change. This page names the real
extension points.

## How a report is consumed

`specify verify <target>` runs the target's `command` (if any), reads the file
at `report`, and parses it into normalized results. Each result is a
`reports.Result`:

```go
// internal/reports/reports.go
type Result struct {
    Suite string `json:"suite"`
    Name  string `json:"name"`
    Pass  bool   `json:"pass"`
}
```

`Name` is the **join identity** — the test's name as the runner reports it.
The scenario binding is declared in *source* (a Swift `.scenario("…")` trait, a
test title leading with `[scenario.id]`, or a leading `// [scenario.id]`
comment), and the engine joins a binding to a result when the result's `Name`
equals — or ends with — the binding's identity (`identityMatch` in
`internal/engine/verify.go`). A report never carries the scenario ID itself;
that's what keeps the binding in source, where the firewall can diff it.

## The three extension points

Adding a format touches exactly three places:

1. **A parser in `internal/reports`** — a function
   `Parse<Format>(data []byte) ([]Result, error)`, in its own file, next to the
   existing three:
   - `junit.go` — `ParseJUnit`: JUnit-family XML; a `<testcase>` with a
     `<failure>`/`<error>` child failed, a `<skipped>` child is not-passing.
   - `events.go` — `ParseSwiftEvents`: Swift Testing's
     `--event-stream-output-path` NDJSON; `test` records carry the display name,
     `issueRecorded` events mark failures.
   - `gotest.go` — `ParseGoTest`: `go test -json` NDJSON; top-level `func Test…`
     functions only (subtests roll up into their parent), skips omitted,
     interleaved non-JSON lines tolerated.

   The parser owns the format's judgment calls — what counts as a failure, what
   is omitted, how noise is tolerated. Keep it deterministic and offline: bytes
   in, results out.

2. **The dispatch in `internal/engine/verify_run.go`** — `joinTarget` selects
   the parser with a `switch cfg.Format` over the format name; add a case.

3. **The allow-list in `internal/config`** — `config.Formats`
   (`internal/config/config.go`) is what `scan` and `target add` validate a
   target's `format` against; add the name.

Then document the format in [config.md](config.md)'s `format` key and add
a parser test in `internal/reports` with a small captured fixture (see
`internal/reports/testdata/`).

## What does *not* change

The join, the lock, drift, cover, parity, and the gates are all downstream of
`[]reports.Result` — they never see the raw report. A new format needs no
engine change beyond the one `switch` case, and no change to the binding
scanner unless the format implies a new *binding* form (a new way of declaring
`[scenario.id]` in source), which is a separate, orthogonal extension
(`bindingsInContent` in `internal/engine/verify.go`).
