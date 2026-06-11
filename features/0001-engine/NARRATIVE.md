---
id: narrative.engine
kind: narrative
---

# Narrative: The spec engine

An agent and a human share one spec library and N native implementations of it. The scarce thing is not generating code — it's _trustworthy closure_: knowing which specs are honestly satisfied on which platform, and being told loudly when that stops being true.

The engine is the connective tissue. It reads the spec library (`domain.specmodel`), checks it is internally well-formed (`scan`), runs each platform's real test suite and joins the results back to the scenarios that were supposed to be proven (`verify`), and records a content-hash acknowledgment of what was last green so it can later say — deterministically, without trusting mtimes or memory — what has drifted (`drift`).

Everything the engine reports must be earned. A green run means "these scenarios were joined to passing tests at this exact spec content," not "the tests passed." A scenario the engine cannot join to a test is not quietly ignored; it is a failure. A platform deviation is surfaced for a human to sign off, not rubber-stamped. The engine's value is precisely that it refuses to lie on the agent's behalf.

This feature is the engine's own specification — SpecKit specifying the tool that implements SpecKit. The Go tests under `internal/` carry `// SPEC:` pointers back to these stories.
