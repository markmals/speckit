---
id: narrative.adoption
kind: narrative
---

# Narrative: Adopting SpecKit into existing code

The code came first. A repo already has its build, its test runner, its layout — SpecKit arrives second, and it must arrive as a guest, not a landlord. Adoption is one command that records where a target lives, how its tests run, and where the report lands. It renders no files into the target, runs no scripts, and never asks what the project is built with; the engine only needs enough to parse a report and scan sources for bindings.

Nothing about a target is privileged by construction. Which target the others are measured against is configuration — `reference_target` in `.speckit/specs.json` — not a hardcoded choice. When it is unset and several targets exist, none is the reference and the engine privileges none.

Adoption also means surviving history. A config written by an earlier SpecKit still loads: retired keys are ignored with a single notice pointing at the migration guide, older schema versions load, and any write brings the file to the current schema. An adopter is never punished for having adopted early.
