---
id: narrative.init
kind: narrative
---

# Narrative: Bootstrapping a SpecKit project

A developer runs `specify init` in a new or existing directory and gets a project that is immediately ready for their agent — Claude Code, Codex, or Copilot — across whichever surface they use (CLI, desktop, or editor extension). The slash-command prompts are projected into that agent's native format; the spec conventions, constitution, and process prompts are in place; the runtime binary is present, so there is no script layer to install and nothing goes stale after init.

Unlike upstream spec-kit — where `specify` is an installer that disappears — the SpecKit binary stays. `init` is just one of its subcommands, the same binary that later runs `scan`, `verify`, and `drift`. Capability is additive, not subtractive: `extension add` layers an extension onto the project rather than pruning a superset, and `extension remove` restores the prior state cleanly.

The test of a good init is boring: the project builds, the agent can read its orientation, and every command prompt invokes `specify`. No surprises, on any of the three host OSes.
