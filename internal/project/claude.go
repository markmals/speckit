package project

import (
	"fmt"
	"os"
	"path/filepath"
)

func init() { register(claudeAdapter{}) }

// claudeAdapter projects commands to .claude/skills/speckit-<cmd>/SKILL.md and
// writes CLAUDE.md as the orientation file.
//
// SPEC: story.init.projection (scenario.init.projection.claude)
type claudeAdapter struct{}

func (claudeAdapter) ID() string { return "claude" }

func (claudeAdapter) SkillsDir() string { return ".claude/skills" }

func (claudeAdapter) AgentsDir() string { return ".claude/agents" }

func (claudeAdapter) Project(root string, commands []Command) ([]string, error) {
	var written []string
	for _, c := range commands {
		dir := filepath.Join(root, ".claude", "skills", "speckit-"+c.Name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		p := filepath.Join(dir, "SKILL.md")
		if err := os.WriteFile(p, []byte(skillDoc(c)), 0o644); err != nil {
			return nil, err
		}
		written = append(written, p)
	}
	orient := filepath.Join(root, "CLAUDE.md")
	if err := os.WriteFile(orient, []byte(claudeOrientation), 0o644); err != nil {
		return nil, err
	}
	return append(written, orient), nil
}

func skillDoc(c Command) string {
	return fmt.Sprintf("---\nname: \"speckit-%s\"\ndescription: %q\nuser-invocable: true\n---\n\n%s",
		c.Name, c.Description, c.Body)
}

const claudeOrientation = `# SpecKit

This project uses SpecKit. The ` + "`/speckit.*`" + ` commands live in
` + "`.claude/skills/`" + ` and the runtime state in ` + "`.speckit/`" + `.
Run ` + "`specify`" + ` for the engine (scan / verify / drift / parity).
`
