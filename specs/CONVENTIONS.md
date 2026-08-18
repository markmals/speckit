---
id: conventions
kind: conventions
---

# Spec Conventions

This document defines the structure of specs in SpecKit. Every spec, reverse pointer, and drift check assumes these rules. The `internal/specmodel` package is the mechanized form of this file; if you change anything here, audit `specmodel` and every existing spec for consistency.

> **TL;DR:** Markdown files with YAML frontmatter, stable dotted IDs, one logical thing per file, `// SPEC: <id>` comments in the code that implements them, and a content-hash lock that records what was last verified green.

This file is derived from the Workbench conventions and amended per the fork plan — most importantly **D7** (drift is a content-hash acknowledgment lock, not an mtime comparison).

## Why specs at all

Native, idiomatic implementations on every target mean the _behavior_ must converge even though the _code_ won't. Specs are the only artifact shared across targets — they are the contract. Everything else — each target's implementation — is a regeneration target. Specs describe **what** must hold; tests prove it; implementations satisfy it. None of the three is the source of truth on its own.

SpecKit is its own first user: the engine's behavior is specified here under `features/` and `specs/models/`, and its tests carry the same reverse pointers any product would.

## File and directory layout

```
specs/                          ← cross-cutting (used by ≥ 2 features or tool-wide)
├── CONVENTIONS.md              ← this file
├── ARCHITECTURE.md             ← singular per product (optional)
├── DESIGN_SYSTEM.md            ← singular per product (optional)
├── models/<id>.md              ← cross-cutting domain models
└── view-models/<id>.md         ← cross-cutting view models (rare)

features/<NNNN>-<slug>/         ← feature-scoped (only this feature uses it)
├── NARRATIVE.md                ← singular per feature
├── README.md                   ← singular per feature; describes the folder
├── stories/<id>.md             ← one user story per file
├── use-cases/<id>.md           ← one concrete use case per file
├── user-flow/<id>.md           ← one interaction sequence per file
├── models/<id>.md              ← one domain model per file
├── view-models/<id>.md         ← one view model per file
└── errors/<id>.md              ← one error catalog entry per file
```

### One logical thing per file

If a kind has multiple instances in a feature, it gets a **directory** of `<id>.md` files. If a kind has exactly one instance (the narrative), it stays a **file**. Directory names are kebab-case (`view-models/`, not `view_models/`).

### Cross-cutting vs feature-scoped

A spec lives in `features/<n>/` until a _second_ feature depends on it, then it is **promoted** to `specs/<kind>/<id>.md` — the file moves, **the ID does not change**, and reverse pointers stay valid through the move.

## Frontmatter schema

```yaml
---
id: <stable-dotted-id>          # required; must match the filename stem
kind: <one of the kinds below>  # required
depends-on: [<id>, <id>]        # optional; specs this one references
status: draft | accepted        # optional; default = accepted
---
```

Singular cross-cutting files use the short form (`id: conventions`, `kind: conventions`). `depends-on` is a flat, non-transitive list, primarily so a human or agent can grep for "what depends on `domain.specmodel`".

## Kind taxonomy

The closed set of allowed `kind:` values. Adding a kind is a deliberate change to this file **and** to `internal/specmodel` — never ad hoc.

| Kind            | Directory       | ID prefix                      | Notes                                          |
| --------------- | --------------- | ------------------------------ | ---------------------------------------------- |
| `narrative`     | (singular file) | `narrative.<feature-slug>`     | One per feature.                               |
| `story`         | `stories/`      | `story.<feature>.<capability>` | Gherkin lives inline.                          |
| `use-case`      | `use-cases/`    | `usecase.<feature>.<scenario>` | Concrete walkthrough.                          |
| `flow`          | `user-flow/`    | `flow.<feature>.<action>`      | Step-by-step interaction sequence.             |
| `domain`        | `models/`       | `domain.<entity>`              | Data shapes, invariants, validation rules. May carry scenarios as `- [scenario.id] …` acceptance bullets (a contract's acceptance criteria), joined like a story's. |
| `view-model`    | `view-models/`  | `vm.<feature>.<view>`          | State, actions, transitions, derived values.   |
| `command`       | `commands/`     | `command.<tool>.<verb>`        | One CLI command's behavior: flags, the work it does, output projection, exit code. |
| `error`         | `errors/`       | `error.<domain>.<kind>`        | User-observable failure + recovery.            |
| `protocol`      | `protocol/`     | `protocol.<producer>.<op>`     | Cross-workspace wire contract (one producer + ≥2 consumers). Carries scenarios, joined to tests like a story's (D12). |
| `architecture`  | (singular file) | `architecture`                 | Cross-cutting; one per product.                |
| `design-system` | (singular file) | `design-system`                | Cross-cutting; one per product.                |
| `conventions`   | (this file)     | `conventions`                  | Cross-cutting; one per product.                |

## Stable IDs

IDs are dotted, lowercase, hierarchical, and stable. The first segment is the kind prefix; the rest narrow to an instance.

**Good:** `domain.specmodel`, `story.engine.scan`, `vm.items.list`, `error.item.duplicate`
**Bad:** `Specmodel`, `engine/scan`, `vm-items-list`, `viewmodel.items.list` (use `vm.`)

- IDs are immutable once an implementation references them. Renaming is a deliberate migration: update the spec ID, every `// SPEC:` reference, and every test tag in one commit.
- IDs do not change on promotion from `features/` to `specs/`.
- IDs do not encode target — they describe abstract behavior.

### Filename = ID stem

The filename matches the trailing segment of the ID, dots preserved within the stem: `domain.specmodel` → `models/specmodel.md`; `story.engine.scan` → `stories/engine.scan.md`; `vm.items.list` → `view-models/items.list.md`.

The scanner also accepts the **full dotted ID** as the stem (`stories/story.engine.scan.md`), so a library may name files by either convention — or, like an adopted external project, a mix. Gherkin scenario headings are recognized at any level (`## Scenario …` or a nested `### Scenario N: …` under `## Acceptance Criteria`), and a spec may instead declare scenarios as inline `- [scenario.id] …` acceptance bullets — both forms produce scenarios the engine joins, on any non-singular kind (see [Stories and scenarios](#stories-and-scenarios)).

## Reverse pointers

Every implementation unit that realizes a spec carries its ID in a comment, attached to the smallest unit that fully realizes the spec (usually a class or top-level function). Do not annotate every helper.

```go
// SPEC: domain.specmodel
type Frontmatter struct { /* ... */ }
```

```ts
// SPEC: vm.items.list
export const itemsListQueryOptions = queryOptions({ /* ... */ })
```

```swift
// SPEC: vm.items.list
@Observable final class ItemsListViewModel { /* ... */ }
```

### Tests carry the same IDs — the scenario join (D12)

The engine joins a scenario to the test that proves it. The binding is **declared in source**; the **outcome is read from the report and matched by test identity** (suite/class + test name). The report does _not_ need to carry the scenario ID — verified in spike 0001, where Swift Testing's xunit and event-stream output drop custom traits and display names entirely. A scenario with no bound test, or a bound test the spec doesn't declare, is a **hard error** (`scan`/`verify`), never a silent zero-match.

**Use each framework's native metadata affordance where it has one; a source comment where it doesn't — never pollute the human-readable test description.** The scanner reads the binding from source; the runner's report supplies only pass/fail keyed by test identity, so no framework needs to surface the tag in its output.

| Framework | Affordance — binds without touching the description |
| --- | --- |
| **Swift Testing** | Custom traits: `@Suite(.spec("<id>"))` + `@Test(.scenario("<sub-id>"))` (shipped as `SpecTraits.swift`), with **raw-identifier** function names for the human description. The dotted ID lives in the trait. |
| **MSTest** (C#) | Attributes: `[TestProperty("spec", "<id>")]` and `[TestProperty("scenario", "<sub-id>")]` (or `[TestCategory]`) — metadata, not the method name. |
| **kotlin.test** (JUnit-target) | Annotations: `@Tag("spec:<id>")` and `@Tag("scenario:<sub-id>")` — metadata, not the `@DisplayName`. |
| **Vitest** | No native trait — a `// [scenario.<sub-id>]` comment directly above the `it(...)`, keeping the title clean. (A `[scenario.…]`-prefixed title is also accepted.) |
| **cargo-nextest** (Rust) | No native trait — a `// [scenario.<sub-id>]` comment above the `#[test]` fn (fn names can't hold dots/brackets). |
| **go test** | No native trait — a `// [scenario.<sub-id>]` comment above the test func / `t.Run`. |

In every case the per-format verify adapter normalizes the report to pass/fail by test identity, and the join overlays the source-declared scenario binding. Because the binding is source-side, encoding the scenario into test _names_ — and the fragile mangling that needs — is never required.

### Binding forms are language-scoped

A framework's affordance is only read from files of that framework's language: the `it("[scenario.id] …")` title form only from JS/TS sources, the `.scenario(…)` trait only from Swift sources, and the `// [scenario.id]` leading comment from any of them. A code sample embedded in another language's string literal is therefore never a binding — a project whose own tests carry sample binding syntax as fixture data declares nothing.

## Stories and scenarios

A story file contains frontmatter, an `As a / I want / So that` block, and an `# Acceptance Criteria` section of Gherkin scenarios. Each scenario carries a stable sub-ID:

```md
## Scenario 1: Creating an item with valid information

<!-- id: scenario.item.create.happy-path -->

- Given a signed-in user
- When the user creates an item with valid information
- Then the item appears in the user's list
```

Sub-IDs follow `scenario.<feature>.<capability>.<short-name>` and are what tests reference in their `[scenario.id]` tag.

Scenarios are not story-exclusive. **Every non-singular (per-file behavioral) kind may carry scenarios** the engine joins — `story`, `use-case`, `flow`, `domain`, `view-model`, `command`, `error`, and `protocol` — in either the `## Scenario` heading form above or the inline `- [scenario.id] …` acceptance-bullet form. The singular cross-cutting kinds (`narrative`, `architecture`, `design-system`, `conventions`) do not: their files are prose, so the engine never parses scenarios from them — which is why the illustrative `scenario.item.create.happy-path` in this file is never read as a real declaration.

## Marking unspecified content

Do not silently guess. Mark gaps inline with `[NEEDS CLARIFICATION: <question>]`. A spec cannot be implemented (`/speckit.implement`) while markers remain; `/speckit.clarify` surfaces and resolves them. Use it when a behavior/constraint/value is unstated or two interpretations are equally plausible — not for implementation details (those are `(deviates:)` comments) or out-of-scope placeholders.

## Deviation marker (D11)

When a target must diverge — constraint, idiom, or deliberate UX choice — annotate it so the pointer stays live:

```swift
// SPEC: vm.items.list (deviates: this target uses pull-to-refresh; the reference target uses a button)
```

```ts
// SPEC: manual — target-specific code with no spec
```

`(deviates: <reason>)` keeps drift detection flagging spec changes. Per **D11** a deviation is a **human attestation the engine cannot verify**: `specify parity` surfaces every marker in a stale-deviation audit and treats a deviation cell as "needs sign-off," never as green. `// SPEC: manual` opts out entirely, used sparingly.

**Scenario-scoped deviations.** Parity is computed per scenario, so a story-level marker is too coarse when only one scenario diverges. Target the scenario sub-ID directly:

```swift
// SPEC: scenario.items.list.empty (deviates: this target shows a system empty view)
```

The engine crosses deviation-presence with the joined test outcome on **independent axes** (spike 0001): a marker over a _failing_ test is classified `suspect`, never `declared-deviation` — a marker can never suppress a red test.

## Drift detection (D7 — acknowledgment lock, not mtime)

A spec and its implementation on a target are **in sync** when:

1. Every spec has at least one reverse pointer on that target (or is intentionally not yet implemented).
2. The target's lock shard for the spec records the **content hash of the spec version last verified green**, and that hash matches the spec's current content.
3. Tests tagged with the spec's scenarios exist on the target and passed at that verified-green hash.

The lock lives at `.speckit/lock/<target>/<spec-id>` — sharded per spec so parallel worktree agents never merge-conflict. `specify lock` is the **only** writer, invoked by `specify verify` on green; the path is covered by the generated-file gate. `specify drift` reports any spec whose current hash differs from its locked hash (or has no lock) — hash-mismatch-or-missing, with no reliance on filesystem mtimes (which git does not preserve).

## Reconciliation

When one target's implementation diverges from the spec (usually a direct bug fix), `specify reconcile <target>` reads the target's impl + tests, diffs against the spec, and proposes updates to the spec and the other targets. Reconciliation is **not automatic** — the agent proposes; a human approves.

## What is NOT a spec

Reference material an agent may read but must not place under `specs/` or a feature's spec subdirectories: wireframes/Figma URLs, prototype code, meeting notes/RFCs (use `docs/`), analytics/telemetry, and target-local cosmetic defects (those go in the target's own defect log). The test: **could a different target realize this differently and still be correct?** If no, it's implementation, not spec.
