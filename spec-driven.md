# Specification-Driven Development (SDD)

## The power inversion

For decades, code was king. Specifications served code — scaffolding we built and
discarded once the "real work" of coding began. We wrote PRDs to guide
development and design docs to inform implementation, but these were always
subordinate to the code itself. Code was truth; everything else was, at best,
good intentions. As the asset (code) and its implementation are one and the same,
the spec rarely kept pace.

Spec-Driven Development inverts this. Specifications don't serve code — code
serves specifications. The spec is the primary artifact; code is one expression
of it, in a particular language and framework. This is possible now because the
cost of generating and regenerating code has collapsed. When writing a thousand
lines is cheap and rewriting them is cheaper, the scarce, durable thing is no
longer the code — it's a precise, complete, unambiguous statement of the behavior
the code must satisfy, and a way to prove the code still satisfies it.

That last clause is where SpecKit lives. Plenty of tools help you *write* specs.
SpecKit's job is to make the spec *load-bearing*: to keep every implementation
honest against it, and to say loudly — deterministically, in your terminal and in
CI — the moment any implementation stops telling the truth.

## The SpecKit model

SpecKit is built for one common, painful shape of software: **one behavior,
implemented natively many times.** A product ships on the web in React, on Apple
in Swift, on Android in Kotlin. There's no shared runtime and no cross-platform
framework — only the same intended behavior, expressed several ways.

- **One spec library, on `main`.** The behavior is written once as a spec library:
  feature folders under `features/<NNNN>-<slug>/` — a NARRATIVE, user stories with
  acceptance-criteria scenarios, domain models, error catalogs. Every story,
  scenario, and model carries a **stable dotted ID** (`story.welcome.greet`,
  `scenario.welcome.greet.hello`, `domain.specmodel`). These IDs are the spine
  everything else hangs from.

- **N native targets.** Each implementation is a **target** — a stack (web, apple,
  android, a Go CLI, …) declared in `.speckit/specs.json` with the command that
  runs its tests and the report it produces. The same specs; one target at a time;
  the first target is a worked example the rest mirror.

- **The `specify` binary is the engine.** It reads the spec library, checks it's
  internally well-formed, runs each target's real test suite, and joins the results
  back to the scenarios they were meant to prove. It is present at runtime, in the
  repo and in CI — not a one-time generator you run and discard.

## The load-bearing mechanism: the join

A green test run is not the same as a satisfied spec. SpecKit refuses to conflate
them. The connective tissue is a **source-bound join** between scenarios and tests.

You bind a test to the scenario it proves, in source, using the test framework's
own affordances — a Vitest title prefix, a Swift Testing trait, a Kotlin tag:

```ts
it("[scenario.welcome.greet.hello] greets a user by name", () => { … })
```

And you point code back at the spec it implements with a reverse pointer:

```ts
// SPEC: story.welcome.greet
export function greeting(name: string): string { … }
```

`specify verify <target>` then runs the target's tests, reads each test's scenario
binding *from source*, and joins it to the report outcome by test identity. From
that join it produces per-scenario pass/fail — and it is strict on purpose:

- A scenario with **no** bound test is not silently ignored; it's a hard failure.
- A test bound to a scenario **no story declares** is a dangling reference — a hard
  failure, not a passing test.
- "Green" means *the right scenarios were proven at this exact spec content* — not
  "the tests passed."

When a target genuinely must behave differently, you say so in code:
`// SPEC: <scenario-id> (deviates: <reason>)`. The engine surfaces that as a
**declared deviation** for a human to sign off — but if the test is actually
failing, it shows as **suspect**. Marking something intentional can never hide a
real failure.

## The workflow in practice

1. **Author the spec, code-free.** Write the feature's stories, scenarios, and
   models. `specify scan` lints the whole library — malformed or duplicate IDs,
   broken cross-references, scenarios missing IDs — before any code exists.

2. **Scaffold the target.** `specify target add <name> --stack <stack>` lays down a
   runnable starter on the recommended stack, registers the target, projects the
   stack's skill pack, and installs dependencies — resolving each package version
   by *running the package manager*, not hardcoding it. The starter arrives green
   on `verify`: one example spec, one bound test, one reverse pointer, so you
   extend a working loop instead of wiring one.

3. **Plan and implement, tests first.** The `/speckit.plan`, `/speckit.tasks`, and
   `/speckit.implement` prompts drive the agent to plan natively and write the
   failing, scenario-bound tests before the code that passes them. Plans and task
   lists are *disposable working artifacts*, not committed sources of truth — the
   spec library is.

4. **Verify and lock.** `specify verify <target>` runs the join. On full success it
   writes a **lock** — `.speckit/lock/<target>/<spec-id>.json`, a content-hash
   acknowledgment that this spec was proven green on this target at this exact
   content. The lock is what makes later questions answerable deterministically,
   without trusting mtimes or memory.

5. **Stay honest over time.**
   - `specify drift <target>` — which specs changed since they were last verified.
   - `specify cover <spec-id>` — where one spec stands across every target.
   - `specify parity <target>` — the full per-scenario picture: conforming,
     declared-deviation, drifted, suspect, or missing.

6. **Enforce at the boundary.** `specify gate` runs in git hooks and CI: it blocks a
   change that edits a scenario-tagged test without touching that scenario's spec
   (the firewall), blocks edits to files SpecKit owns (locks, codegen), and checks
   commit-subject scopes. Honesty stops being a matter of discipline and becomes a
   property of the repository.

## Why trunk-based, not branch-per-feature

SpecKit keeps the spec library on `main` as the single source of truth. It does
**not** mint a numbered branch per feature or generate a tree of `plan.md` /
`research.md` / `contracts/` files as committed artifacts. Targets advance on
trunk, each verified independently; divergence between targets is captured
exactly where it happens — in the lock and parity state under `.speckit/`, and in
the `(deviates:)` reverse pointers — never in a parallel forest of documents that
drifts from the code.

## Principles

- **The spec is the source of truth.** Code is a materialized view of it. When they
  disagree, the engine makes the disagreement loud and specific.
- **Everything the engine reports must be earned.** A green run is a *proof* —
  scenarios joined to passing tests at a known spec content — or it is nothing.
- **No silent gaps.** An unjoinable scenario, a dangling test, a drifted lock: each
  is surfaced by name. The engine's whole value is that it refuses to lie on your
  behalf.
- **Direction is human; execution is fast.** People set the behavior and approve
  deviations. The agent does the typing and closes the loop — implement, test,
  verify, fix, repeat.

A project may also keep a lightweight **constitution** (`/speckit.constitution`) —
a short set of non-negotiable principles the agent must honor — but its content is
yours to define, not a fixed creed inherited from any one stack.

## The transformation

SDD is not a better way to write documents. It's a different center of gravity for
how software is built: the spec is the asset, the implementations are projections
of it, and a present-at-runtime engine guarantees the projections stay faithful or
fail visibly. Writing the code got cheap. Knowing — provably, continuously — that
the code still does what you said it should is the part that didn't, and that is
what SpecKit is for.
