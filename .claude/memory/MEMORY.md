# Project memory

Durable, non-obvious knowledge about the SpecKit repo that survives across agent
sessions. This index is loaded every session; the topic files are read on demand.
Working knowledge only — required behavior is the spec library (`features/`,
`specs/`), and the engine never reads this dir. Maintain with the `managing-memory`
skill.

## Topics

- [Engine boundaries](engine-boundaries.md) — the offline determinism line; where GitHub/network code may live
- [Dev workflow](dev-workflow.md) — the CI gate, golden regen, the dual exit-code convention, scaffold gotchas
- [Web scaffold](web-scaffold.md) — feature/variant composition, the provider `Wrap` seam, and the toolchain gotchas when adding a `--with`
- [racket-ui via shadcn](rac-ui-shadcn.md) — the proven recipe to replace the foundation with racket-ui (`markmals/racket-ui`) via the shadcn CLI; **gated on committing `registry.json` to that repo's root**
