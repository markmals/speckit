---
id: narrative.work
kind: narrative
---

# Narrative: Pluggable work tracking

Somebody has to remember what's in flight. But a work item is ephemeral coordination — who is doing what right now — and the engine's verdicts (`scan`, `verify`, `drift`, `cover`, `parity`, `gate`) are earned from specs, source, reports, and the lock. The two must never mix: work state is not an input to any engine command, and no work item is a source of spec truth.

So work tracking is a provider behind five verbs — ready, create, claim, move, list — chosen at adoption time. The default is a single committed markdown file: diffable, offline, no binary beyond `specify` itself. A project that wants typed dependencies uses the beads provider over the `bd` CLI; one that lives on a GitHub Projects board drives it directly; one that tracks work elsewhere says `none` and every verb quietly steps aside.

The choice is structural, not aspirational: the engine packages import no provider and no GitHub client, so every engine command runs with no network and no credentials — whichever tracker a team picked, or none at all.
