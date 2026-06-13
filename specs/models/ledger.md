---
id: domain.ledger
kind: domain
depends-on: [domain.specmodel]
---

# Domain: Run ledger

The append-only record of regeneration attempts — the raw material for `specify bench` (framework-curation receipts) and for understanding agent throughput per spec/target.

## Shape

Append-only JSONL under `.speckit/`; one record per `apply` attempt:

- `spec`, `target` — what was applied where.
- `attempts` — iterations to reach green (or give up).
- `iterations` — per-iteration scenario results (which scenarios were red→green when).
- `wall_time` — elapsed.
- `tokens` — model tokens where available (nullable).
- `fail_first_observed` — whether a failing test was observed before the implementation passed it (TDD honesty signal).

## Invariants

- **G1 — Append-only.** Records are only appended; existing records are never rewritten.
- **G2 — One per attempt.** Each `apply` run appends exactly one record.
- **G3 — Self-describing.** A record carries enough to reconstruct the attempt's outcome without external state (spec, target, results, timings).
- **G4 — Nullable cost.** `tokens` is optional; absence means "not measured," not zero.
