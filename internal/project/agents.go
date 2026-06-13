package project

import (
	"os"
	"path/filepath"
)

func init() {
	register(agentsAdapter{id: "codex"})
	register(agentsAdapter{id: "generic"})
}

// agentsAdapter projects commands to .agents/skills/speckit-<cmd>/SKILL.md and
// writes AGENTS.md. Shared by codex and generic — both read the .agents/ +
// AGENTS.md convention (D4).
//
// SPEC: story.init.projection (scenario.init.projection.codex, scenario.init.projection.generic)
type agentsAdapter struct{ id string }

func (a agentsAdapter) ID() string { return a.id }

func (agentsAdapter) SkillsDir() string { return ".agents/skills" }

// Codex/generic have no projectable subagent-dir convention; the review pack is
// Claude-only for now.
func (agentsAdapter) AgentsDir() string { return "" }

func (agentsAdapter) RulesDir() string { return ".agents/rules" }

func (agentsAdapter) MemoryDir() string { return ".agents/memory" }

func (a agentsAdapter) Project(root string, commands []Command) ([]string, error) {
	var written []string
	for _, c := range commands {
		dir := filepath.Join(root, ".agents", "skills", "speckit-"+c.Name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		p := filepath.Join(dir, "SKILL.md")
		if err := os.WriteFile(p, []byte(skillDoc(c)), 0o644); err != nil {
			return nil, err
		}
		written = append(written, p)
	}
	orient := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(orient, []byte(agentsOrientation), 0o644); err != nil {
		return nil, err
	}
	return append(written, orient), nil
}

const agentsOrientation = `# SpecKit

This project uses SpecKit. The ` + "`/speckit.*`" + ` commands live in
` + "`.agents/skills/`" + ` and the runtime state in ` + "`.speckit/`" + `.
Run ` + "`specify`" + ` for the engine (scan / verify / drift / parity).

## Rules

Follow the conventions in ` + "`.agents/rules/`" + `: ` + "`code-quality.md`" + `,
` + "`commit-discipline.md`" + `, and ` + "`spec-conventions.md`" + ` apply to
every change; ` + "`enforcement-hierarchy.md`" + ` is the standard for where a new
convention lives.

## Project memory

At the start of a session, read the project memory index —
` + "`.agents/memory/MEMORY.md`" + ` — and any topic files it points to. It's
durable, agent-owned working knowledge about this repo (not spec truth; the engine
never reads it). Maintain it with the ` + "`managing-memory`" + ` skill.
`
