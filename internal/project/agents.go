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
`
