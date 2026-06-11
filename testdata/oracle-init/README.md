# Oracle init manifests (upstream reference — not the fork's contract)

Captured from the pinned Python CLI (`fork-base`) via
`specify init <dir> --integration <agent> --ignore-agent-tools --script sh`
(D14: capture goldens from the oracle; never read its source as the spec).

Each `<agent>.files.txt` is the sorted file manifest **upstream** init produces.
The fork's init deliberately **diverges** — these are a reference for the
_retained_ behavior (the per-agent command projection + orientation file), not a
tree to match byte-for-byte.

## Shared base vs per-agent projection

~25 files are common to every agent (the `.specify/` runtime, templates,
constitution, the agent-context extension). The per-agent delta is the command
projection + orientation file + integration manifest:

| agent | command projection | orientation |
| --- | --- | --- |
| claude | `.claude/skills/speckit-*/SKILL.md` | `CLAUDE.md` |
| codex | `.agents/skills/speckit-*/SKILL.md` | `AGENTS.md` |
| copilot | `.github/agents/*.agent.md` + `.github/prompts/*.prompt.md` | `.github/copilot-instructions.md` |
| generic | `.agent/commands/speckit.*.md` (configurable dir) | `AGENTS.md` |

## Retained vs divergent (the fork's init, per FORK-PLAN)

- **Retained** (specced from these manifests): the per-agent projection above;
  the orientation file; templates; the constitution; the agent-context extension.
- **Divergent** (specced as the fork's _own_ behavior): `.specify/` → `.speckit/`
  (D6); **no** `.specify/scripts/bash/*.sh` (D2 — logic moves into the binary;
  only git-hook trampolines remain); **no** workflow engine / `workflows/`
  (deferred); three agents + generic only (D4); `--json` on every command (D2);
  no "GitHub Spec Kit" banner (D1 affiliation).

The fork's OWN init goldens get captured from the fork's `specify init` once it
is implemented (Phase 2); those become the durable golden-tree contract and
these upstream manifests retire at that point.
