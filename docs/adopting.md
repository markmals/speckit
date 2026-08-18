# Adopting SpecKit into an existing project

SpecKit adopts into a project that already exists. It generates no code,
installs no dependencies, and chooses no platform — you bring the project, the
tests, and the test runner; SpecKit brings the spec library and the
verification loop.

The whole adoption is four commands:

```sh
# 1. Project the agent integration into the current directory. --force is
#    required in a non-empty directory: the guard exists to protect a tree you
#    already own, and adoption always runs against one.
specify init --here --integration claude --force    # or codex, copilot, generic

# 2. Register your existing code as a target (see the worked examples below)
specify target add web --dir apps/web --format junit \
  --report apps/web/report.junit.xml --source apps/web/src \
  --command "npm --prefix apps/web test" --bindings scoped

# 3. Confirm the spec library and the config are well-formed
specify scan

# 4. Run the target's tests and join them to scenarios
specify verify web
```

`init --here` projects the `/speckit.*` agent commands, skills, and rules into
the current directory (it never clobbers accumulated agent memory). Because an
existing project is by definition non-empty, pass `--force`; without it `init`
refuses rather than writing into a directory it did not create. `target add`
writes one entry into `.speckit/specs.json` — nothing else; no files are
generated into the target's directory. At no point is a platform chosen.

## What a target registration says

- `--dir` — where the target lives, relative to the project root.
- `--command` — the shell command that runs the tests and produces the report.
  Omit it when the report already exists before `verify` runs.
- `--format` + `--report` — how and where the engine reads the results:
  `junit`, `swift`, or `gotest` (see the examples).
- `--source` — the directory scanned for scenario bindings; repeat the flag
  when bound tests span several directories.
- `--bindings scoped` — see below.
- `--product <label>` — an optional grouping label for `cover`/`parity`
  rollups.
- `--reference` — see below.

Full schema (including hand-editing the JSON directly) in
[config.md](config.md).

## Worked examples, one per report format

### `junit` — any runner with a JUnit XML reporter

Most JS/TS runners (and Gradle, and many others) can write JUnit-family XML.
Point the project's test script at a JUnit reporter so it writes the report
path, then:

```sh
specify target add web --dir apps/web --format junit \
  --report apps/web/report.junit.xml \
  --source apps/web/src \
  --command "npm --prefix apps/web test" \
  --bindings scoped
```

Bindings are read from source: a test title leading with the scenario ID
(`it("[scenario.projects.create.basic] creates a project", …)`) or a
`// [scenario.projects.create.basic]` comment on the line above the test.

### `swift` — Swift Testing's event stream

Swift Testing writes an NDJSON event stream the engine parses directly (the
xunit output drops the binding, so the event stream is the outcome source):

```sh
specify target add ios --dir apps/ios --format swift \
  --report apps/ios/.build/tests.ndjson \
  --source apps/ios/Tests \
  --command "swift test --package-path apps/ios --event-stream-output-path apps/ios/.build/tests.ndjson --event-stream-version 0" \
  --bindings scoped
```

The binding is the `.scenario("scenario.projects.create.basic")` trait on a
`@Test`; the join identity is the test's display name.

### `gotest` — `go test -json`

```sh
specify target add api --dir cmd/api --format gotest \
  --report .speckit/api.gotest.json \
  --source cmd/api --source internal \
  --command "go test -json ./cmd/... ./internal/... > .speckit/api.gotest.json" \
  --bindings scoped
```

The binding is a leading `// [scenario.projects.create.basic]` comment above
the `func Test…`; the join identity is the function name as `go test` reports
it. Note the repeated `--source`: a Go target whose bound tests span `cmd/` and
`internal/` lists both, and the engine joins the bindings into one target.

## `--bindings scoped` — the adoption default in practice

Under the default `strict` mode, **every** test in `--source` must bind a
scenario — an untagged test is a violation. That's the right discipline for a
suite grown inside SpecKit, but an adopting project almost always has a body of
pre-existing plain unit tests that prove nothing spec-shaped and shouldn't have
to. `--bindings scoped` treats untagged tests as out of scope: the suite runs
whole, the engine verifies the scenarios that *are* bound, and a failing bound
test or a dangling binding (naming a scenario that doesn't exist) is still a
hard error. Start `scoped`; tighten to `strict` if and when the whole suite is
scenario-bound.

## `--reference` — multi-target repos

When several targets implement the same specs, one of them is usually the
worked example the others match when a spec is ambiguous. Register that one
with `--reference`:

```sh
specify target add web … --reference
```

That sets `reference_target` in `.speckit/specs.json`. It's informational — the
engine privileges no target — but projected agent guidance reads it instead of
assuming a platform. When unset, no target is privileged.

## Then: bind as you verify

Each test names the scenario it proves, in source — that's the binding `verify`
joins on (the scanner reads `.swift`, `.ts`, `.tsx`, `.js`, `.mjs`, and `.go`
sources). If your tests already carry scenario tags, `verify` is green on day
one; otherwise add tags as you verify each spec. A scenario with no test, or a
test naming a scenario that doesn't exist, fails `verify` and is named — that's
the point.
