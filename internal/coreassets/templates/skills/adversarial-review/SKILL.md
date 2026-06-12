---
name: adversarial-review
description: Use as the final refutational pass before declaring a spec done — or whenever code looks correct and the suite is green but nobody has tried to break it. Surfaces defects on edges the spec never enumerated, tests that would still pass if the behavior were wrong, and aliasing / mutation / concurrency / resource bugs that confirmatory review approves.
---

# Adversarial Review

Confirmatory review checks the code against the spec's explicit clauses and the quality rules. This pass is **refutational**: its job is not to confirm the code is right; it is to assume the code is wrong and find the input, caller action, or missing test that proves it.

**Core principle:** a green suite and a clause-by-clause match prove the code does what someone thought to check. The adversary's whole value is in what nobody thought to check. (Lifts the adversarial-refinement idea from VSDD: a refutational stage, fresh context, refute-by-default, convergence by exhaustion.)

## When to use

- The final stage before declaring any spec done on any target — after the code matches the spec and `specify verify <target>` is green.
- Any time code "looks correct, tests pass" but no one has tried to break it.
- Standalone, against an existing implementation you distrust.

**Do NOT use it** to replace the basic spec-match check (a refutational pass on code that doesn't even match the spec wastes attention), or on trivial edits (a renamed constant — nothing to break).

## The stance

**Refute by default.** Open assuming the code is broken and the tests are theater. Your output is the evidence that breaks it, or — only after you have genuinely tried and failed — the admission that you couldn't.

- No "overall this looks solid, but…". No preamble, no goodwill, no credit for clean formatting.
- Every finding is a concrete defect: the exact location, the exact input or caller action that triggers it, and observed-vs-required behavior.
- You are not here to be agreeable. You are here to be correct about what is wrong.

## Fresh context, different model

- **Fresh context window.** The adversary carries no memory of building the code — no accumulated goodwill, no "I know what I meant here." Dispatch it as a subagent or run it in a clean session.
- **Different model from the implementer where possible.** Cognitive diversity catches shared blind spots.

## What to attack — five lenses

Spend the most effort on the first; it's the one confirmatory review can't cover.

1. **Un-enumerated edges.** The spec lists what must hold; you hunt what it forgot to forbid. For every input: null, empty, whitespace-only, max size, negative, zero, duplicate, unicode, already-sorted, reverse-sorted, concurrent. For every return value: is it an alias of internal state a caller can mutate? For every resource: released on every path, including the throwing one? For every `async`: what races?
2. **Test quality.** Would this test still pass if the behavior were subtly wrong? Hunt tautologies, assertions on implementation detail, over-mocking that tests the mock, fixtures that dodge the hard case, and **the missing test** — the scenario the spec implies but the suite never exercises. (`specify verify` already fails on an *unbound* scenario; you hunt the under-asserted one.)
3. **Spec fidelity under interpretation.** The code may satisfy the letter of a clause while violating its intent. Name the ambiguity and the interpretation the code chose.
4. **Security surface.** Unvalidated boundary input, injection vectors, authorization assumed rather than checked.
5. **Spec gaps the implementation reveals.** Implemented behavior no clause covers — either scope creep to cut, or a real behavior the spec should capture. Route the latter back to the spec, not into silent acceptance.

## Verify before you report — no fabricated flaws

A fabricated defect corrupts the review as surely as a missed one. **Every claimed defect carries a concrete reproduction you actually traced.** Before you write "this breaks on input X," run X through the code step by step and confirm the bad result. If you can't produce the triggering input, you have a suspicion, not a defect — mark it as such; never promote it.

## Convergence — the stopping signal

Hallucination-based termination. You are done when, and only when, you are **reduced to inventing problems that aren't there.**

- Real defect found → report it; it feeds back; you go again on the fixed code.
- Findings have decayed to wording nitpicks → **converged.** Say so plainly.
- You catch yourself manufacturing a flaw to have something to say → that's the exit signal, not a finding. Stop and report convergence.

"I tried to break it along all five lenses and could not" is a clean, valuable output. Don't pad.

## Output format

```
ADVERSARIAL REVIEW — <spec-id> on <target>

DEFECTS (must fix):
  1. <location> — <what's wrong> — repro: <exact input/action> — got <X>, spec requires <Y>

SUSPICIONS (unverified, needs a check):
  - <observation and the input that might trigger it>

SPEC GAPS (route to spec, not to the implementer):
  - <implemented behavior no clause covers, or clause the code reinterpreted>

VERDICT: BROKEN (n defects) | CONVERGED (no real defect found across all five lenses)
```

**DEFECTS** → fix, then re-run this stage. **SUSPICIONS** → verify or drop before acting; never fix blind. **SPEC GAPS** → surface to the user / route to the spec; don't let the implementer silently encode an answer. Loop until VERDICT is CONVERGED.

## Red flags — you reverted to confirmation

You wrote "looks good" anywhere · you only checked the clauses the spec listed · you trusted the green suite as proof · you never constructed a single input designed to break the code · you reported a defect you didn't actually trace. Any of these → restart from refute-by-default.

## Related skills

- `test-driven-development` — the discipline whose tests this stage tries to defeat.
- `systematic-debugging` — once a defect is confirmed, find its root cause before the fix.
- `verification-before-completion` — the gate this stage feeds; a spec isn't done until the adversary converges.
