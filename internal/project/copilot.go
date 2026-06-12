package project

import (
	"fmt"
	"os"
	"path/filepath"
)

func init() { register(copilotAdapter{}) }

// copilotAdapter projects each command to .github/agents/speckit.<cmd>.agent.md
// and .github/prompts/speckit.<cmd>.prompt.md, plus the
// .github/copilot-instructions.md orientation file.
//
// SPEC: story.init.projection (scenario.init.projection.copilot)
type copilotAdapter struct{}

func (copilotAdapter) ID() string { return "copilot" }

// Copilot has no skills concept; the discipline lives in its instructions.
func (copilotAdapter) SkillsDir() string { return "" }

func (copilotAdapter) Project(root string, commands []Command) ([]string, error) {
	agentsDir := filepath.Join(root, ".github", "agents")
	promptsDir := filepath.Join(root, ".github", "prompts")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(promptsDir, 0o755); err != nil {
		return nil, err
	}

	var written []string
	for _, c := range commands {
		ap := filepath.Join(agentsDir, "speckit."+c.Name+".agent.md")
		if err := os.WriteFile(ap, []byte(copilotAgent(c)), 0o644); err != nil {
			return nil, err
		}
		pp := filepath.Join(promptsDir, "speckit."+c.Name+".prompt.md")
		if err := os.WriteFile(pp, []byte(copilotPrompt(c)), 0o644); err != nil {
			return nil, err
		}
		written = append(written, ap, pp)
	}

	orient := filepath.Join(root, ".github", "copilot-instructions.md")
	if err := os.WriteFile(orient, []byte(copilotOrientation), 0o644); err != nil {
		return nil, err
	}
	return append(written, orient), nil
}

func copilotAgent(c Command) string {
	return fmt.Sprintf("---\nname: speckit.%s\ndescription: %q\n---\n\n%s", c.Name, c.Description, c.Body)
}

func copilotPrompt(c Command) string {
	return fmt.Sprintf("---\ndescription: %q\n---\n\n%s", c.Description, c.Body)
}

const copilotOrientation = `# SpecKit

This project uses SpecKit. The ` + "`/speckit.*`" + ` commands live in
` + "`.github/agents/`" + ` and ` + "`.github/prompts/`" + `; runtime state in
` + "`.speckit/`" + `. Run ` + "`specify`" + ` for the engine (scan / verify / drift / parity).
`
