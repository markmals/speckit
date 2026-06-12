---
name: writing-user-stories
description: Use when writing or reviewing user stories and Gherkin acceptance criteria (Given/When/Then) for a SpecKit feature. Trigger when the output is a user-facing capability described from the user's perspective — not when designing APIs, schemas, or technical plans (those are /speckit.plan).
---

# Writing User Stories

A user story describes **one user-observable capability** in plain language, paired with **Gherkin acceptance criteria** that are externally testable. The story is the _what and why_; the criteria are the _contract_. In SpecKit each scenario carries a stable sub-ID that a test binds to (the join the engine verifies), so the criteria you write here are what `specify verify` later proves.

**Core principle:** _Imagine it's 1922._ Most software does something a person could do manually, just less efficiently. If your story or scenarios depend on a particular UI, framework, endpoint, or database, you've written implementation, not a user story — and you've coupled a cross-target spec to one target's mechanics.

## Story file shape (per `specs/CONVENTIONS.md`)

```md
---
id: story.<feature>.<capability>
kind: story
depends-on: [domain.<entity>, vm.<feature>.<view>]
---

**As a** [real user persona]
**I want** [capability the user performs]
**So that** [user-visible outcome or value]

**Independent test:** [how this story can be verified on its own]

# Acceptance Criteria

## Scenario 1: [Specific behavior in plain language]

<!-- id: scenario.<feature>.<capability>.<short-name> -->

- Given [initial state]
- When [single user action]
- Then [observable outcome]
```

Rules: the **user** is a real human actor (not "system"/"service"/"scheduler"); the **want** is what the user does, not how it's built; the **so that** is value the user perceives. Use consistent terminology — no synonyms for the same concept.

## One story = one capability

A story delivers one user-observable capability. If criteria branch into unrelated verbs — _share_ + _revoke_ + _audit_ + _rate-limit_ — it's too large; split it. Symptoms: more than ~6 scenarios, cross-cutting concerns (a11y, perf, security) bundled in, or an "Out of Scope"/"NFR" section smuggling extra behavior. Cross-cutting concerns get their own stories or shared standards.

## Avoid the "how"

Never reference frameworks, components, endpoints, routes, status codes, data models, tables, or named UI elements (buttons, dialogs, dropdowns). Those live in `/speckit.plan` output or design, applied evenly across targets.

## The three steps

- **Given = state, not actions.** The scene before the user interacts. `Given the user is signed in`, not `Given the user clicks export`. Use vivid named characters (`Dr. Bill` beats `User A`). Repeated givens across every scenario → a `## Background` (≤4 lines, one per story).
- **When = exactly one trigger.** One user action or event. Multiple `When`s = multiple scenarios. `When the user requests a data export`, not a four-step click chain.
- **Then = observable outcomes only.** What the user can see, receive, or experience. Never `Then an audit log entry is created` / `Then the row is inserted` / `Then a 403 is returned` / `Then the job is queued`. If you want to assert internal state, step back: what does the user actually observe?

Successive givens/thens read better with `And`/`But`. Never use `And` to hide a second `When`.

## Worked example

```md
**As a** signed-in customer
**I want** to download my transaction history
**So that** I can keep my own records and use the data in other tools.

# Acceptance Criteria

## Scenario 1: Exporting personal data as CSV
<!-- id: scenario.export.csv.happy-path -->
- Given the user is signed in
- And the user has recorded transactions
- When the user requests a data export
- Then the user receives a downloadable file containing all recorded transactions

## Scenario 2: Exporting with no records
<!-- id: scenario.export.csv.empty -->
- Given the user is signed in
- And the user has no recorded transactions
- When the user requests a data export
- Then the user receives a file containing only column headers
```

Two scenarios, one capability, no UI mechanics, no filenames/encodings/status-codes/queues. User perspective intact end-to-end.

## Mark gaps, don't guess

When a behavior, constraint, or value is unstated and two readings are equally plausible, insert `[NEEDS CLARIFICATION: <question>]` inline rather than inventing an answer. `/speckit.clarify` resolves these; a story with open markers can't be implemented.

## Red flags — stop and rewrite

| Red flag in a step | Fix |
| --- | --- |
| `click`, `tap`, `drag`, `select from dropdown`, `type into field` | Describe what the user is _trying to do_ |
| Named UI elements ("Export button", "Share dialog") | Describe the action abstractly |
| Multiple `When`s in one scenario | Split scenarios |
| `audit log`, `database`, `queue`, `cache`, `record is created` | Replace with what the user observes |
| HTTP codes, endpoints, payloads | Replace with the user-facing message/behavior |
| File encodings, BOMs, RFC numbers, p95 latency | Move to `/speckit.plan` or NFR standards |
| `OR` / branching inside a step | Split into separate scenarios |
| `Given the user clicks…` | Move to `When`, or rewrite as state |
| `Then the system…` / `Then the backend…` | Rewrite from the user's perspective |
| 10+ scenarios spanning multiple verbs | Split into multiple stories, one capability each |

## Related skills

- `brainstorming-feature` — the feature-authoring flow that calls this skill for each story.
- `test-driven-development` — turns each scenario's sub-ID into a failing test that `specify verify` joins.
