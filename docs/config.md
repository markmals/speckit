# Configuration — `.speckit/specs.json`

`.speckit/specs.json` declares your project's **targets** — the native
implementations the engine verifies. It's plain JSON (strict — no comments or
trailing commas).

## Schema (today)

```json
{
  "version": 1,
  "agent": "claude",
  "paths": { "specs": "specs", "features": "features" },
  "targets": {
    "web": {
      "stack": "web",
      "command": "pnpm -C apps/web test --run",
      "format": "junit",
      "report": "apps/web/report.junit.xml",
      "source": "apps/web/src",
      "product": "consumer-app"
    },
    "ios": {
      "stack": "apple",
      "command": "xcodebuild test -scheme App -resultBundlePath …",
      "format": "swift",
      "report": "apps/ios/.build/tests.ndjson",
      "source": "apps/ios/Tests",
      "product": "consumer-app"
    }
  }
}
```

Field by field: `version` is this file's schema version; `agent` is who `init`
projected for; `paths` (optional, defaults shown) locates the spec library; each
**target** carries a `stack` (selects its platform pack — see below), the verify
wiring (`command` to run, `report` `format` ∈ `junit`/`swift`, `report` path,
`source` dir scanned for bindings), and an optional `product` label.

### What the engine does with it

- `specify verify <target>` runs the target's `command`, parses the `report` in
  `format`, scans `source` for source-declared scenario bindings (D15), joins
  results to declared scenarios, and on green writes the lock at
  `.speckit/lock/<target>/<spec-id>.json`.
- `specify drift <target>` · `cover <spec-id>` · `parity <target>` read that
  per-target lock.
- `specify scan` validates this file when it's present: every target needs a
  valid `format` (`junit` | `swift`), a `report`, and a `source`. An absent
  `specs.json` is fine — engine commands that need a target just tell you to
  configure one.

A **target is the atomic unit**: a globally-unique name with its own lock. The
`platform` vocabulary from earlier builds is gone — it's `target` everywhere now.

## Platform packs

A target's optional **`stack`** selects a **pack** of platform skills — the
stack-specific dev and verification guidance (React/TanStack, UIKit/SwiftUI,
Compose, WinUI, the CLI stacks, …). Unlike the process-discipline skills (which
`init` projects for every project), platform skills are stack-specific, so
they're projected **on demand**:

```sh
specify packs        # project the packs for every stack in specs.json
```

`packs` reads `specs.json`, takes the distinct `stack` values across your
targets, and projects each pack's skills into the agent's skills dir — using the
`agent` field to know where (`.claude/skills`, `.agents/skills`, `.github/skills`).
Re-run it after adding a target on a new stack.

| `stack` | pack skills projected |
| --- | --- |
| `web` | `web-development`, `web-verification` |
| `website` | `website-development` |
| `apple` | `ios-development`, `ios-simulator-control` |
| `android` | `android-development`, `android-emulator-control` |
| `go-cli` · `node-cli` | the matching `*-development` skill |

The GUI packs pair with the `visual-verifier` subagent, which drives a real
browser / simulator / emulator through a story's Gherkin scenarios.

## Products — today, a label

A **product** is an application; a **target** is one implementation of it (web,
iOS, a Convex server, a Go daemon — all targets of one product). A repo can hold
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

A **contract** is how a product's targets communicate through a service —
currently a Convex backend or an OpenAPI server (Node/Go). A single contract can
be consumed by targets across *several* products, so contracts would be a
top-level collection referencing targets, not nested under any one product:

```json
"contracts": {
  "auth-api": {
    "kind":       "openapi",                      // openapi | convex
    "definition": "contracts/auth.openapi.yaml",  // the interface schema
    "provider":   "auth-server",                  // the target that serves it
    "consumers":  ["web", "ios", "admin-web"]     // targets that call it — may cross products
  }
}
```

**Why deferred:** in its first useful form a contract is only *validated*
(provider/consumers resolve to real targets, the definition file exists) — inert
otherwise. Its real payoff is **contract-drift**: flagging the consumer targets
when a provider changes the interface. That's a second acknowledgment/lock
mechanism (parallel to spec-drift) and arguably overlaps with codegen/diff
tooling (OpenAPI diff, `openapi-codegen`). The schema should be shaped by that
drift behavior once it's designed — not guessed at now. Until then, document a
product's service topology in its `NARRATIVE.md` or an `ARCHITECTURE.md`, not the
tooling config.
