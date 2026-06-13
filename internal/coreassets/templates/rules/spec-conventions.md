# Shared Spec Rules

> Projected by `specify init` into the agent's rules dir and loaded every
> session. Keep it short — full conventions live in `specs/CONVENTIONS.md`.

## The compact

- **Specs in `specs/` and `features/<NNNN>/` are the source of truth.**
  Implementations on every target must satisfy them.
- **Reverse pointers are mandatory.** Every class, function, or module that
  realizes a spec carries `// SPEC: <id>`. Tests are tagged with the scenario IDs
  they verify (the `[scenario.<id>]` prefix / `.scenario(...)` trait) — that tag
  is the join `specify verify` reads.
- **Use `// SPEC: <id> (deviates: <reason>)` when a target must differ.** Use
  `// SPEC: manual` for genuinely target-specific code with no cross-target
  analog. A bare deviation a test still fails is a `suspect` cell in `parity` —
  the marker never suppresses a failing test.
- **The spec defines what; the test proves it; the implementation satisfies it.**
  None is the source of truth alone.
- **Web is the reference target.** When implementing a spec on any other target
  (Apple, Android, a CLI), read the web realization for context — but the spec is
  authoritative.

## Before writing implementation code

1. Read the spec file. Confirm the ID, depends-on chain, and behavior.
2. Read the web reference implementation if one exists.
3. Read the target's existing patterns for similar specs (look for other
   `// SPEC:` annotations in the same area).
4. Write the failing tests first, tagged with the scenario IDs.
5. Implement the minimum to pass the tests.
6. Verify with `specify verify <target>` — it joins each scenario to its test and
   locks what passes.

## Before changing a spec

1. Search for the ID in the codebase: `rg 'SPEC: <id>'`.
2. List the affected targets and tests (`specify cover <id>` shows per-target
   status).
3. Update the spec.
4. Re-verify each affected target (`specify verify <target>`); reconcile the
   implementation and tests until green. Don't auto-rewrite tests to pass — that
   trips the `gate firewall`.

## Before changing implementation that has a spec

1. Decide: is this a bug fix the spec already requires, or a behavior change?
2. If behavior change: update the spec first, then bring each target back to green.
3. If bug fix: fix it, run `specify verify`, and check no other target has the
   same bug (`specify cover <id>`).

## Where to read more

- `specs/CONVENTIONS.md` — full conventions: kind taxonomy, frontmatter schema,
  stable IDs, reverse pointers, the scenario join, the deviation marker, drift.
- `specs/models/` — the domain models the engine itself is specified against.
