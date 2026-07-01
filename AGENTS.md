# SpecKit (the tool source)

This repo is SpecKit itself — the `specify` Go binary (engine + project bootstrapper) plus the assets it projects (`internal/coreassets/templates/`). It is developed with Codex and dogfoods its own agent-memory feature.

- Engine: `internal/engine` + `internal/specmodel`. CLI: `cmd/specify` (Cobra).
- Projection (`specify init`): `internal/project` + the embedded `templates/`.
- Live tracker for in-flight work: `BACKLOG.md`. Design docs: `docs/design/`.

## Project memory

Durable, non-obvious project knowledge lives in `.claude/memory/`; its index is auto-loaded every session:

`.claude/memory/MEMORY.md`

Maintain it with the discipline in the `managing-memory` skill — agent-owned working knowledge, not spec truth (the engine never reads it).
