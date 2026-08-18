---
name: verification-before-completion
description: Use before claiming any work is complete, fixed, or passing. Requires running the verifying command in this turn and reading its output before stating success. Evidence before assertions, always.
---

# Verification Before Completion

Claiming work is complete without verification is dishonesty, not efficiency. Evidence before claims, always.

## The Iron Law

```
NO COMPLETION CLAIMS WITHOUT FRESH VERIFICATION EVIDENCE
```

If you haven't run the verification command **in this turn**, you cannot claim the result.

## The gate

```
1. IDENTIFY  — what command proves the claim?
2. RUN       — execute the full command, fresh, no shortcuts
3. READ      — full output, exit code, failure count
4. VERIFY    — does the output confirm the claim?
                - No  → state actual status with evidence
                - Yes → state claim with evidence
5. CLAIM     — only now
```

Skip any step = lying.

## What proves what

| Claim | Required evidence | Insufficient |
| --- | --- | --- |
| Tests pass | Test command output: 0 failures, this turn | "Last run was clean", "Should pass now" |
| Build succeeds | Build command exit 0, this turn | Linter passed, types passed |
| Spec library well-formed | `specify scan` exits 0, this turn | "I didn't change the frontmatter" |
| Spec verified on a target | `specify verify <target>` green for the spec | "Code looks right", "tests pass locally" |
| Nothing drifted | `specify drift <target>` clean | "I only touched one file" |
| Parity clean | `specify parity <target>` all-conforming | "the deviation is intentional" (that's `suspect` until signed off) |
| Bug fixed | The failing reproduction now passes | Code changed, "I think it's fixed" |
| Regression test works | Red → green → revert → red → restore → green | Test passes once on the fixed code |
| Subagent finished | `git status` / `git diff` shows the changes | The subagent's own success report |

For a regression test, the full proof: write the test → run it with the fix (PASS) → revert the fix → run it (FAIL) → restore → run it (PASS). Without the FAIL step, you don't know the test catches the bug.

## Red flags — stop

"Should", "probably", "seems to", "I'm pretty sure" · "Great!/Perfect!/Done!" before running anything · about to commit or open a PR without re-running · trusting a subagent's report without checking the diff · claiming success from a partial check ("typecheck passed, build will pass") · tired and wanting it to be over · "just this once". Any of these = run the verification first.

## How to phrase results

```
✅ Ran `specify verify app` — story.items.list and story.item.create green, lock written.
```

Not "Looks good now!" / "Tests should be passing." / "I think we're done."

State bad results just as cleanly:

```
`specify verify app` is not green:
  FAIL       story.item.create   ([scenario.item.create.duplicate] — no error thrown)
  unjoinable story.items.list     (scenario.items.list.empty has no test)
Investigating now.
```

## When to apply

**Always before:** saying "done / complete / fixed / passing / ready / shipped" or anything positive about the state of the work; committing (or asking the user to); moving to the next task; reporting subagent success up to the user; closing an apply session. The rule covers exact phrases, paraphrases, synonyms, and **anything that implies success.**

## Why this matters

A false-positive completion claim makes the user review work that isn't done, builds the next work on a broken foundation, and erodes trust until they double-check everything. Running the command takes seconds; skipping it can cost hours.

## Bottom line

Run the command. Read the output. Then claim the result. Non-negotiable.

## Related skills

- `test-driven-development` — produces the verified-green state this gate checks.
- `adversarial-review` — a spec isn't done until the adversary converges; this gate enforces that.
