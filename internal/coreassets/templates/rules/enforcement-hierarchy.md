# Enforcement Hierarchy

> The standard to apply when deciding _where a convention lives_. Read it when
> you reach for a new rule — it is reference, not a per-session checklist.

A rule an agent must _remember_ is the weakest kind of rule. Under load, prose
gets skipped — the more conventions a `SKILL.md` carries, the more reliably some
get ignored. The durable conventions in a SpecKit repo are the ones a machine
enforces, not the ones an agent is asked to keep in mind. So when you reach for a
new rule, reach down this hierarchy first.

## The tiers — strongest to weakest

- **Tier 0 — The engine gate** (`specify gate`). Deterministic; the agent cannot
  forget or skip it. `gate firewall` blocks a scenario-tagged test edited away
  from its spec; `gate generated` refuses edits to engine-owned output
  (`.speckit/lock/`, codegen); `gate scope` rejects a commit whose scope isn't a
  defined spec/feature ID, target, or harness area. Wired as git hooks and as the
  CI required check. If a rule can be a gate check, it should be.
- **Tier 1 — Commands & the engine.** The `/speckit.*` authoring commands and the
  `specify` engine (`scan` / `verify` / `drift` / `cover` / `parity`) plus each
  target's own configured commands (its test command in `.speckit/specs.json`,
  the project's fmt/lint tooling). Agent-invoked, but the
  _behavior_ is codified, not recalled. Drift, coverage, parity, and verification
  belong here — and already live here: the engine computes them from the repo, it
  doesn't ask the agent to.
- **Tier 2 — Templates** (`.speckit/templates/` and the feature-folder
  templates). Shape the work so the correct thing is the path of least resistance
  — the frontmatter, the `// SPEC:` reverse pointer, the scenario tags all come
  pre-wired.
- **Tier 3 — Prose** (`SKILL.md` files, these `rules/` files). The rule the agent
  must read and remember. Necessary for judgment that can't be mechanized, but the
  tier most likely to be missed.

## The rule

- **Before adding a prose rule, ask whether the gate or the engine could enforce
  it deterministically.** If yes, build that instead of writing the prose.
- **Where a mechanism already enforces a rule, don't also state it in prose.** The
  duplicate prose rots — it drifts from the mechanism and competes for attention.
  Delete it.
- **Promote to a mechanism only when the check is cheap, deterministic, and
  target-agnostic.** Subjective judgment — good naming, the right abstraction,
  whether a deviation still makes sense — stays prose. A linter cannot make that
  call, and pretending it can produces noise.

## Worked example — the sync invariants

The join invariants in `specs/CONVENTIONS.md` are exactly the kind of rule that
belongs in a mechanism, not prose: _every declared scenario has a bound,
passing test_; _no test is bound to an undeclared scenario_; _a spec's lock
matches its current content hash_. All three are cheap, deterministic, and
target-agnostic — so the engine owns them. `specify verify` computes the
scenario↔test join and fails on any unjoinable scenario or dangling binding;
`specify drift` flags a spec whose text changed since it was last verified;
`specify parity --gate` fails unless every scenario conforms. None of it is prose
an agent has to remember. **That is the bar:** when a convention is cheap,
deterministic, and target-agnostic, mechanize it; otherwise leave it here.
