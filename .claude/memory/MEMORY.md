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
- [npm-package scaffold](npm-package-scaffold.md) — the node-family single-TS-library twin of swift-package (replaced the old `ts-lib` roster name): scoped bindings, source-read junit binding, the tsdown/oxfmt gotchas
- [racket-ui via shadcn](rac-ui-shadcn.md) — the proven recipe to replace the foundation with racket-ui (`markmals/racket-ui`) via the shadcn CLI; **gated on committing `registry.json` to that repo's root**
- [Apple scaffold](apple-scaffold.md) — the `apple` stack + the `swift-package`/`swift-cli` sibling library stacks: verify=headless Core/package, the swift event-stream command, the dynamic-module/static-dir `path:` trick, `.tmpl`-for-substitution-only, the `.scenario()` binding, packless-stack handling
- [Mise monorepo](mise-monorepo.md) — generated root config invariant; the unstable-parser comment-preserving merge; the family↔member drift coupling
- [Fork & upstream](fork-and-upstream.md) — markmals/spec-kit is a FORK of github/spec-kit; gh/PR ops default to the upstream, so target the fork with `-R markmals/spec-kit`
