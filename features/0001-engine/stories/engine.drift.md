---
id: story.engine.drift
kind: story
depends-on: [story.engine.verify]
---

# Story: Detect drift against the lock

As a developer or agent maintaining N implementations of one spec,
I want `specify drift <platform>` to tell me which specs changed since they were last verified green,
So that reconciliation is a deterministic command, not a guess — and independent of filesystem mtimes (D7).

# Acceptance Criteria

## Scenario 1: An edited spec drifts red

<!-- id: scenario.engine.drift.edited-spec-red -->

- Given a spec previously verified green on a platform (its lock shard exists)
- When the spec file's content is edited so its hash changes
- And the user runs `specify drift <platform>`
- Then the spec is reported as drifted (hash mismatch)
- And the command exits non-zero

## Scenario 2: Re-verifying clears the drift

<!-- id: scenario.engine.drift.reverify-clears -->

- Given a spec reported as drifted on a platform
- When the platform is brought back in line and `specify verify <platform>` passes
- And the user runs `specify drift <platform>`
- Then the spec is no longer reported as drifted

## Scenario 3: A spec never verified is reported as missing

<!-- id: scenario.engine.drift.never-verified-missing -->

- Given a spec that has no lock shard on a platform
- When the user runs `specify drift <platform>`
- Then the spec is reported as missing (unverified), distinct from drifted

## Scenario 4: Drift ignores mtime

<!-- id: scenario.engine.drift.ignores-mtime -->

- Given a spec verified green, whose file mtime is then changed without changing its content (e.g. a fresh checkout or `touch`)
- When the user runs `specify drift <platform>`
- Then the spec is not reported as drifted, because only the content hash is consulted
