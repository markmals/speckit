package project

import (
	"errors"
	"io/fs"
	"strings"
)

// Subagent is a Claude Code review/automation subagent (spec-reviewer,
// test-gap-finder, drift-hunter, handoff-builder, …) projected into the agent's
// agents dir. Content is the full <name>.md including frontmatter and is written
// verbatim — it's already in the agent's native format (the claude-pack).
type Subagent struct {
	Name    string
	Content string
}

// loadSubagents reads templates/agents/<name>.md from the embedded assets.
func loadSubagents(assets fs.FS) ([]Subagent, error) {
	entries, err := fs.ReadDir(assets, "templates/agents")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var subs []Subagent
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		b, err := fs.ReadFile(assets, "templates/agents/"+e.Name())
		if err != nil {
			return nil, err
		}
		subs = append(subs, Subagent{Name: strings.TrimSuffix(e.Name(), ".md"), Content: string(b)})
	}
	return subs, nil
}
