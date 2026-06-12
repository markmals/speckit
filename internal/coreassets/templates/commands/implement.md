---
description: Implement a feature or spec on a platform — failing tests first, layered review, adversarial pass, then verify-and-lock.
---

# /speckit.implement — Implement on a platform

Bring a platform's code and tests into conformance with the spec, one spec ID at a time, ending each in a green `specify verify` that locks it. This is the default "how to write code" workflow.

**Arguments:** `<feature-or-spec-id> <platform>` — e.g. `/speckit.implement 0001-managing-items web` (whole feature) or `/speckit.implement story.item.create web` (one spec).

## What to do

Run the **`implementing-a-spec`** skill. It carries the discipline; the short version:

1. **Checklist gate.** If `features/<NNNN>/checklists/` exists, show each checklist's pass/fail. **Block on any incomplete checklist** — the user must explicitly choose to proceed.
2. **Order the work.** If `features/<NNNN>/tasks/<platform>.md` exists, follow its phases (P1 story first). Otherwise take the spec IDs in dependency order.
3. **Per spec ID** — dispatch a fresh implementer subagent that:
   - writes **failing tests first**, bound to the scenario sub-IDs via the platform's native affordance (`specs/CONVENTIONS.md`), and confirms they fail for the right reason (`test-driven-development`);
   - writes the **minimum** code to pass;
   - attaches `// SPEC: <id>` (or `// SPEC: <id> (deviates: <reason>)`) to the smallest realizing unit.
4. **Review, in order:** spec-compliance → code-quality → **adversarial** (`adversarial-review`, fresh context, different model). Don't reorder; don't run the adversary until both confirmatory reviews are ✅. Loop on BROKEN; route SPEC GAPS back to the spec, never a silent code change.
5. **Verify and lock.** Run `specify verify <platform>` — on green it joins each scenario to its test and writes `.speckit/lock/<platform>/<spec-id>` (D7). A red verify is not done → `systematic-debugging`.

When all specs are in, `specify drift <platform>` should be clean. Use `verification-before-completion` before claiming done.

## Discipline

Spec is authoritative — if it's wrong, stop and tell the user; they edit it, then re-invoke. Never invent or rename spec IDs. Surface unimplemented dependencies and offer to apply them first. Platform divergences are explicit `(deviates: <reason>)`, never silent.

## Commit

Per spec, once reviews pass and `specify verify` is green: `test: …` then `feat: implement <spec-id> on <platform>` (combine if small). One spec per commit — never bundle two. Scoped per `specify gate scope`.
