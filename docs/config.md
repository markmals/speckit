# Configuration — `.speckit/specs.json`

`.speckit/specs.json` declares your project's **targets** — the implementations
the engine verifies — plus the optional reference target and the work-tracking
provider. It's plain JSON (strict — no comments or trailing commas).

The config is deliberately stack-agnostic: a target is described by where it
lives, how to run its tests, and how to read the resulting report. Nothing in
this file names a framework, runtime, or platform.

## Schema (version 2)

```json
{
  "version": 2,
  "agent": "claude",
  "reference_target": "web",
  "paths": { "specs": "specs", "features": "features" },
  "targets": {
    "web": {
      "dir": "apps/web",
      "command": "npm --prefix apps/web test",
      "format": "junit",
      "report": "apps/web/report.junit.xml",
      "source": "apps/web/src",
      "product": "consumer-app"
    },
    "ios": {
      "dir": "apps/ios",
      "command": "swift test --package-path apps/ios --event-stream-output-path apps/ios/.build/tests.ndjson --event-stream-version 0",
      "format": "swift",
      "report": "apps/ios/.build/tests.ndjson",
      "source": "apps/ios/Tests",
      "product": "consumer-app"
    },
    "daemon": {
      "dir": "cmd/daemon",
      "command": "go test -json ./cmd/... ./internal/... > .speckit/daemon.gotest.json",
      "format": "gotest",
      "report": ".speckit/daemon.gotest.json",
      "source": ["cmd/daemon", "internal"],
      "bindings": "scoped"
    }
  },
  "work": { "provider": "markdown", "file": "WORK.md" }
}
```

## Top-level keys

- **`version`** — the file's schema version. This build reads and writes `2`.
  An older file loads fine (with a one-line notice) and is normalized to `2`
  the next time SpecKit writes it.
- **`agent`** — who `init` projected for (`claude` · `codex` · `copilot` ·
  `generic`). `specify init --integration <agent>` records it.
- **`reference_target`** — the target whose behavior the other targets match
  when a spec is ambiguous across them. Purely informational: the engine
  privileges no target; projected agent guidance reads this key instead of
  hardcoding a platform. Set it with `specify target add <name> … --reference`.
  When unset, no target is privileged (a single-target project needs no key —
  the sole target is trivially the reference).
- **`paths`** — locates the spec library (optional; defaults `specs/` and
  `features/`).
- **`targets`** — the verifiable implementations; see below.
- **`work`** — the work-tracking provider block (optional; see
  [work-providers.md](work-providers.md)). Absent means the `markdown` provider
  on `WORK.md`. No engine command reads this block.

## Target keys

- **`dir`** — the target's root, relative to the project root. Informational —
  nothing is generated into it — but it records what the target *is*.
  `specify target add --dir <path>` writes it; a target that omits it is
  treated as rooted at the project root.
- **`command`** — the shell command that runs the target's tests and produces
  the report. Optional: leave it empty when the report already exists before
  `verify` runs.
- **`format`** — how the report is parsed: `junit` (JUnit-family XML from any
  runner with a JUnit reporter), `swift` (Swift Testing's
  `--event-stream-output-path` NDJSON), or `gotest` (the NDJSON
  `go test -json` writes; the join identity is the `func Test…` name).
- **`report`** — where the report lands, relative to the project root.
- **`source`** — the directory (or directories) scanned for scenario bindings.
  A single string scans one dir; a JSON array scans every listed dir and joins
  the bindings into one target (e.g. a Go service whose tests span `cmd/` and
  `internal/`). One or more non-empty paths are required.
- **`bindings`** — how an untagged test (one that binds no scenario) is treated:
  `strict` (the default — every test must prove a scenario, an untagged one is
  an unbound D12 violation) or `scoped` (untagged tests are out of scope, so a
  suite that mixes scenario tests with plain unit tests still verifies what it
  binds). A failing bound test and a dangling binding remain violations in both
  modes.
- **`product`** / **`products`** — an optional label (or list of labels)
  grouping targets into products; see below.

### Retired keys: `stack` and `deploy`

Earlier schema versions carried a per-target `stack` (which selected a scaffold
and a platform skill pack) and `deploy` (a deployment manifest). Both are
**retired and ignored**: an unmigrated config still loads and every command
still runs — SpecKit prints a one-line notice naming the ignored keys and drops
them on the next write. What a target is built with, and how it deploys, is the
adopting project's business, not the spec engine's.

## What the engine does with it

- `specify verify <target>` runs the target's `command` (if any), parses the
  `report` in `format`, scans `source` for source-declared scenario bindings
  (D15), joins results to declared scenarios, and on green writes the lock at
  `.speckit/lock/<target>/<spec-id>.json`.
- `specify drift <target>` · `cover <spec-id>` · `parity <target>` read that
  per-target lock.
- `specify scan` validates this file when it's present: every target needs a
  valid `format` (`junit` | `swift` | `gotest`), a `report`, and a `source`,
  and any `bindings` value must be `strict` or `scoped`. An absent `specs.json`
  is fine — engine commands that need a target just tell you to configure one.

A **target is the atomic unit**: a globally-unique name with its own lock.

## Products — today, a label

A **product** is an application; a **target** is one implementation of it (a
web app, an iOS app, a Go daemon — all targets of one product). A repo can hold
several products.

Today a product is expressed as an optional **label** on a target: `"product":
"consumer-app"`, or `"products": ["consumer-app", "admin-app"]` for a target
shared by more than one app. Products are derived from the labels; `cover` /
`parity` can group and roll up by them (a product is green when all its targets
are green for the specs they implement). There is no separate `products`
collection — the label carries the whole concept.

## Future options

These are **deliberately not built yet**. Both are purely additive — a new
optional top-level key, no migration — so deferring them costs nothing, and
shipping them before something consumes them would be guesswork.

### A first-class `products` collection

Promote products from a label to a top-level collection the day a multi-product
repo needs products to carry their own metadata (description, owner, release
channel) or a named rollup independent of any single target:

```json
"products": {
  "consumer-app": { "targets": ["web", "ios"], "description": "the customer app" },
  "admin-app":    { "targets": ["admin-web"] }
}
```

Targets stay flat and globally-unique (so a shared target has one lock, not one
per product); the collection just references them by name. The label form above
is the seed of this — migrating is mechanical.

### `contracts`

A **contract** is how a product's targets communicate through a service — an
API schema (e.g. OpenAPI) served by one target and consumed by others. A single
contract can be consumed by targets across *several* products, so contracts
would be a top-level collection referencing targets, not nested under any one
product:

```json
"contracts": {
  "auth-api": {
    "kind":       "openapi",
    "definition": "contracts/auth.openapi.yaml",
    "provider":   "auth-server",
    "consumers":  ["web", "ios", "admin-web"]
  }
}
```

**Why deferred:** in its first useful form a contract is only *validated*
(provider/consumers resolve to real targets, the definition file exists) — inert
otherwise. Its real payoff is **contract-drift**: flagging the consumer targets
when a provider changes the interface. That's a second acknowledgment/lock
mechanism (parallel to spec-drift) and arguably overlaps with codegen/diff
tooling. The schema should be shaped by that drift behavior once it's designed —
not guessed at now. Until then, document a product's service topology in its
`NARRATIVE.md` or an `ARCHITECTURE.md`, not the tooling config.
