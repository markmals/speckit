# Spike 0001 — Engine trust (throwaway)

**Phase 1 of the fork plan.** Disposable: this code exists only to answer a gate
question before the real engine (Phase 3) is designed. Nothing here is promoted
to `internal/`.

## Gate question

> Is mechanized parity + the scenario-to-test join trustworthy enough to gate a
> PR on, or does the deviation cell need a human in the loop?

## Method

One 3-scenario spec (`spec/todo.toggle.md`), implemented twice:

- **web** — TypeScript + Vitest, real junit XML (`web/report.junit.xml`)
- **apple** — Swift + Swift Testing, real xunit XML (`apple/report.xunit.xml`)

A throwaway Go tool (`join/`) parses the spec scenarios + both reports + the
apple source's `(deviates:)` markers, joins tests→scenarios (D12), and computes
the four-cell parity matrix (D11).

The implementations are rigged to exercise the hard cases:

| scenario | web | apple | what it probes |
| --- | --- | --- | --- |
| `complete`   | pass | pass | baseline conforming |
| `reactivate` | pass | pass + **honest** `(deviates:)` | declared-deviation that is real |
| `guard-empty`| pass | **fail** + **lying** `(deviates:)` | D11: a marker shadowing a red test |
| `complete-typo` (web test only) | pass | — | D12: a test tagged to a scenario the spec doesn't declare |

Findings land in `FINDINGS.md` and feed back into the D11/D12 specs.
