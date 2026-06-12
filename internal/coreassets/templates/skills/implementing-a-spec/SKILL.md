---
name: implementing-a-spec
description: The default "how to write code" workflow — the discipline behind /speckit.implement <spec-id> <target>. Dispatches a fresh subagent per spec that writes failing tests, makes them pass, attaches a `// SPEC: <id>` reverse pointer, then runs spec-compliance review, code-quality review, an adversarial refutational pass, and finally `specify verify` to lock the spec green.
---

# Implementing a Spec

Each spec ID is a unit of work. To implement one on a target, dispatch a fresh subagent that writes failing tests (bound to the spec's scenario sub-IDs), makes them pass, attaches the `// SPEC: <id>` reverse pointer, then submits for layered review — spec compliance, code quality, then an adversarial pass that assumes the code is broken (`adversarial-review`). When it converges, `specify verify <target>` runs the suite and writes the lock (D7), making the spec officially green.

**Default workspace:** the user's current branch. No worktree/branch ceremony unless the user asks.

## When to use

- `/speckit.implement <spec-id> <target>`, or "implement <spec-id> on <target>".
- You're about to write substantive new code that realizes a spec.

**Do NOT use for:** trivial edits (do them inline — a failed trivial dispatch costs more than the work); bug fixes (`systematic-debugging` first for root cause, then this skill applies the fix); spec authoring (`brainstorming-feature`).

## Core principle

**Fresh subagent per spec + layered review (confirm, then refute) + lock on green = high quality, fast iteration.**

The controller (you) reads the spec(s) and any sibling-target reference, curates the exact context the implementer needs, tracks per-spec progress, and dispatches/reviews. The implementer writes failing tests → minimum code → reverse pointer → self-review. The reviewers check compliance, quality, then try to break it.

## Process

```
If /speckit.tasks produced features/<NNNN>/tasks/<target>.md, follow its phases
(P1 story first = MVP). Otherwise take the spec IDs in dependency order.

For each spec ID:
  1. Read spec + depends-on chain + a sibling target's reference impl (if any)
  2. Identify existing reverse pointers, tests, and gaps on this target
  3. Construct full context for the implementer (don't make them re-read)
  4. Dispatch implementer subagent → DONE / DONE_WITH_CONCERNS / NEEDS_CONTEXT / BLOCKED
  5. Dispatch spec-compliance reviewer  → fix + re-review until ✅
  6. Dispatch code-quality reviewer     → fix + re-review until ✅
  7. Dispatch adversarial reviewer (adversarial-review; fresh context, different model)
       BROKEN → fix each defect, re-run from step 5.  CONVERGED → continue.
  8. Run `specify verify <target>` → on green it locks the spec. Next spec.

When all specs done:
  9. `specify drift <target>` should be clean. Commit at natural boundaries.
```

### Dispatching the implementer

Provide the **full** spec text (paste it — don't say "read specs/foo.md"), the full text of every `depends-on` spec, any sibling-target reference implementation (`rg "SPEC: <id>"`), and the target's conventions/test-setup. Instruct:

1. **Write failing tests first**, bound to the spec's scenario sub-IDs using the target's native affordance per `specs/CONVENTIONS.md` — Swift traits, MSTest `[TestProperty]`, kotlin `@Tag`, or a `// [scenario.<id>]` source comment. **Never pollute the test description with the ID.**
2. Run the tests and **confirm they fail for the right reason.**
3. Implement the **minimum** code to pass.
4. Attach `// SPEC: <id>` to the smallest unit that fully realizes the spec (a deviation gets `// SPEC: <id> (deviates: <reason>)`).
5. **Self-review** against the spec; fix gaps. Report status. **Do not commit** — the controller commits once reviews pass.

Handle status: DONE → review. DONE_WITH_CONCERNS → address correctness/scope concerns first. NEEDS_CONTEXT → paste the missing context, re-dispatch (never "go look it up"). BLOCKED → diagnose (more context? a stronger model? split the task? wrong spec → stop and surface to the user). Never silently retry the same dispatch.

### The three reviews — non-negotiable, in order

- **Spec compliance:** every clause satisfied; `// SPEC: <id>` present; a test for **every** scenario with the right binding; nothing extra (no scope creep). ✅ or a list of gaps.
- **Code quality:** idiomatic for the target; names match the spec's vocabulary; no duplication; error handling only at boundaries. Apply the **signal-to-noise gate** before reporting — drop nits the formatter/linter owns, restatements, and one-site-only fixes; for each surviving finding, name three other places it would bite or it's a nitpick. A short list of real issues beats a long list that buries them.
- **Adversarial:** the code now satisfies the spec and reads cleanly — exactly when confirmatory review stops looking and real defects hide. Run `adversarial-review` in a **fresh context on a different model**. Handle the verdict: BROKEN → implementer fixes each defect, re-run compliance/quality if substantial, then re-run the adversary; SUSPICIONS → verify before acting; SPEC GAPS → surface to the user and route to a spec edit, never a silent code change; CONVERGED → done.

Never start quality review before compliance is ✅; never run the adversary before both confirmatory reviews are ✅ (refuting code that doesn't match the spec yet spends the adversary on the wrong layer).

### Lock and verify

After the adversary converges, run `specify verify <target>`. It runs the suite, joins each scenario to its bound test, and on green writes `.speckit/lock/<target>/<spec-id>` — the content-hash acknowledgment that this spec version was verified green (D7). A red `specify verify` is not done; return to `systematic-debugging`. Use `verification-before-completion` before claiming the work complete.

## Constraints

Never dispatch parallel implementers on the same files · never let the implementer commit · never skip a review or reorder them · never let self-review replace actual review · the adversary runs on a different model than the implementer — cognitive diversity is the point.

## Commit

Once reviews pass and `specify verify` is green, commit under the scoped-commits convention (`specify gate scope` enforces it). Natural boundaries per spec: `test: add scenarios for <spec-id> on <target>` then `feat: implement <spec-id> on <target>` (combine if the diff is small). Commit each spec independently — never bundle two specs. Review fixups fold into the pair if unpushed, else land as a follow-up `fix:`/`refactor:`.

## Skip dispatch entirely when

The change is trivial enough that dispatch is pure overhead — renaming a constant, a one-line typo, a single added test case, a comment. Just do it inline.

## Related skills

- `brainstorming-feature` — produces the specs this implements.
- `test-driven-development` — the RED→GREEN→lock discipline the implementer follows inside a dispatch.
- `adversarial-review` — the third review stage.
- `verification-before-completion` — the gate before claiming a spec is done.
- `systematic-debugging` — when an implementer is stuck on a confusing failure.
