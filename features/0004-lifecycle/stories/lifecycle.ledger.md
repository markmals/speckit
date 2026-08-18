---
id: story.lifecycle.ledger
kind: story
depends-on: [domain.ledger, story.lifecycle.apply]
---

# Story: Record the run ledger

As a maintainer curating the distro,
I want every `apply` to append a structured run record,
So that framework-curation claims get receipts (`specify bench` reads the ledger) instead of vibes.

# Acceptance Criteria

## Scenario 1: One append-only record per apply

<!-- id: scenario.lifecycle.ledger.append-only -->

- Given an `apply` run
- Then exactly one JSONL record is appended under `.speckit/`, and no prior record is rewritten (`domain.ledger` G1/G2)

## Scenario 2: Records carry iteration detail

<!-- id: scenario.lifecycle.ledger.iterations -->

- Given an `apply` that took several iterations to reach green
- Then the record carries per-iteration scenario results, attempt count, wall time, and the `fail_first_observed` flag

## Scenario 3: Token cost is optional, not zero

<!-- id: scenario.lifecycle.ledger.tokens-nullable -->

- Given an `apply` where model token usage was not measured
- Then the record's `tokens` is null (absent-not-zero, `domain.ledger` G4)

## Scenario 4: The ledger feeds bench

<!-- id: scenario.lifecycle.ledger.feeds-bench -->

- Given a ledger with records across candidate implementations for the same spec set
- When `specify bench` runs (post-v1)
- Then it derives its comparison table from these records, not from a separate measurement run
