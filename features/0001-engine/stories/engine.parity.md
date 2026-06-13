---
id: story.engine.parity
kind: story
depends-on: [story.engine.verify, domain.specmodel]
---

# Story: Parity across targets

As a developer or agent maintaining N native implementations of one spec,
I want `specify parity` to show, per (target × scenario), an honest classification of conformance,
So that I can see at a glance what is genuinely satisfied, what deviates intentionally, and what is masquerading as fine.

Pressure-tested in spike 0001 (`spikes/0001-engine-trust/`) against real Vitest + Swift Testing output. The four-cell model from the plan gained a **fifth state, `suspect`**, when the spike showed a deviation marker can shadow a failing test.

# Acceptance Criteria

## Scenario 1: A passing scenario with no deviation is conforming

<!-- id: scenario.engine.parity.conforming -->

- Given a target whose joined test for a scenario passes and whose source carries no `(deviates:)` marker for it
- When the user runs `specify parity`
- Then the cell is `conforming`

## Scenario 2: A passing scenario with an honest deviation needs sign-off

<!-- id: scenario.engine.parity.declared-deviation -->

- Given a target whose joined test passes and whose source carries a `(deviates: <reason>)` marker for that scenario
- When the user runs `specify parity`
- Then the cell is `declared-deviation`, shown with its reason
- And it is treated as "needs sign-off," never as green (D11)

## Scenario 3: A deviation marker over a failing test is SUSPECT

<!-- id: scenario.engine.parity.suspect-lying-marker -->

- Given a target whose joined test for a scenario FAILS but whose source carries a `(deviates:)` marker for it
- When the user runs `specify parity`
- Then the cell is `suspect`, not `declared-deviation`
- And the engine reports that the marker cannot be machine-verified as intentional
- And `specify parity --gate` exits non-zero

## Scenario 4: A failing scenario with no marker is drifted

<!-- id: scenario.engine.parity.drifted -->

- Given a target whose joined test fails and whose source carries no deviation marker
- When the user runs `specify parity`
- Then the cell is `drifted`

## Scenario 5: A scenario with no test or pointer is missing

<!-- id: scenario.engine.parity.missing -->

- Given a target that neither tests nor carries a reverse pointer for a scenario's spec
- When the user runs `specify parity`
- Then the cell is `missing`, distinct from `drifted`

## Scenario 6: Deviation presence and test outcome are computed independently

<!-- id: scenario.engine.parity.independent-axes -->

- Given any target/scenario pair
- When the engine classifies its parity cell
- Then deviation-marker presence and joined-test outcome are evaluated as independent axes and then crossed
- So that a marker can never override or suppress a failing test result
