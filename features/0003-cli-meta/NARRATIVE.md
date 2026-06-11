---
id: narrative.cli-meta
kind: narrative
---

# Narrative: Meta commands

`specify` is present at runtime (D2), so its smallest commands matter for the
agent loop: an agent needs to know which version it is talking to and whether
the toolchain a task needs is installed — both in a form it can parse.

Two divergences from upstream are deliberate. First, **`--json` works on every
command** (D2): upstream's `version --json` requires `--features`; the fork's
`--json` is universal and unconditional, because the agent-facing contract is
"structured output on demand, always." Second, **no banner**: upstream prints a
"GitHub Spec Kit" ASCII banner; the fork prints none, both to avoid implying
GitHub affiliation (D1) and because banners are noise an agent must skip.

These commands are trivial to implement and are the first end-to-end proof that
the binary, its output modes, and its structured contract all wire together.
