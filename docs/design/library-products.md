# Design — library products

**Status:** proposal, for review. Folds in the `~/Developer` sweep finding that
much of the real work is libraries/SDKs/CLIs/extensions, not end-user apps.

## The problem

SpecKit's authoring model assumes an **end-user app**: NARRATIVE → *human* user
stories → view-models / flows → UI. `writing-user-stories` even hard-codes "the
actor must be a real human — not a system, service, or API." For a **library**
that's backwards: the consumer *is* a developer / an API caller, and the spec is
the public surface + behavioral invariants, not a UI walkthrough.

The engine is unaffected — it joins a scenario to the test that proves it
regardless of who the actor is. So this is an **additive authoring change**, not
an engine change.

## `kind: app | library`

A product is one or the other. The kind selects the authoring path; it does
**not** change the engine, the lock, or the join.

- **Default from stack.** App stacks (`web`, `website`, `apple`, `android`,
  the CLI stacks) ⇒ `app`. Library stacks (`swift-package`, `swift-cli`,
  `npm-package`, `vscode-extension`) ⇒ `library`. An explicit `kind` overrides.
- **Open: where it's declared.** Cleanest is to derive it from the feature's
  target stacks, with an explicit override when a repo mixes both. (CLIs are the
  ambiguous case — a CLI has a human user, so `remctl` is app-flavored, but a
  developer-facing CLI/SDK is library-flavored. Default CLIs to `app`; override
  to `library` for SDK-shaped tools.)

## The library authoring path

When `kind: library`, the authoring skills branch:

- **`writing-user-stories`** — relax the human-actor rule: the actor is the **API
  / CLI consumer**. Scenarios describe **observable API behavior + invariants**,
  still Given/When/Then, still bound to a test:

  ```md
  **As a** developer using the reactivity library
  **I want** derived values to recompute when their dependencies change
  **So that** I don't manage update propagation by hand

  ## Scenario: a derived value tracks its dependency
  <!-- id: scenario.signal.derived.tracks -->
  - Given a signal `count` with value 1 and a derived `doubled = count * 2`
  - When `count` is set to 5
  - Then reading `doubled` yields 10
  ```

  The "no UI mechanics / no internal state" rules still hold — the observable is
  the API's *return value / emitted effect*, not its internals.

- **Applicable spec kinds.** `story` (API behaviors), `domain` (the types/data
  shapes the surface exposes), and `error` (failure modes) apply. The UI-only
  kinds — `view-model`, `flow` — do **not**. (Possible addition: an `api` /
  `contract` kind for declaring the public surface itself — types, signatures,
  guarantees — distinct from behavioral scenarios. Open question: add it, or keep
  the surface implicit in the stories + domain models.)

- **`brainstorming-feature`** — a library branch: explore the **API surface and
  its invariants** (what's public, what guarantees hold, what's out of scope),
  not personas and screens. Lean hard on **property tests** for the invariants
  (the TDD skill already calls for "for all" properties — libraries are where
  that pays off most).

- **`implementing-a-spec`** is unchanged — still RED → GREEN → review → `specify
  verify`/lock. The bound test is a unit/property test instead of a UI test.

## What changes (summary)

- A `kind` (derived-from-stack, overridable) on the product/feature.
- Library branches in `writing-user-stories` and `brainstorming-feature`.
- A note in `specs/CONVENTIONS.md`: which kinds apply per product kind; the
  relaxed actor rule for libraries.
- The library/CLI/extension **scaffolds** (in
  [stack-scaffolding.md](stack-scaffolding.md)).
- **No engine change.**

## Out of scope here

Versioning / semver of a library's public API (a contract-stability concern,
closer to the deferred `contracts` work than to this) — not modeled now.
