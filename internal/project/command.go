package project

import (
	"io/fs"
	"strings"
)

// Command is an agent-neutral command prompt loaded from coreassets and
// projected per agent (D4). Name is the bare command, e.g. "specify".
//
// SPEC: story.init.projection
type Command struct {
	Name        string
	Description string
	Body        string // markdown body, with .speckit/ paths (D6)
}

// loadCommands reads templates/commands/*.md from the embedded asset FS and
// rewrites the runtime dir to the fork's .speckit/ (D6).
func loadCommands(assets fs.FS) ([]Command, error) {
	entries, err := fs.ReadDir(assets, "templates/commands")
	if err != nil {
		return nil, err
	}
	var cmds []Command
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		b, err := fs.ReadFile(assets, "templates/commands/"+e.Name())
		if err != nil {
			return nil, err
		}
		desc, body := splitFrontmatter(string(b))
		// SPEC: story.init.projection — fork runtime is .speckit/, never .specify/ (D6).
		body = strings.ReplaceAll(body, ".specify/", ".speckit/")
		cmds = append(cmds, Command{
			Name:        strings.TrimSuffix(e.Name(), ".md"),
			Description: desc,
			Body:        body,
		})
	}
	return cmds, nil
}

// splitFrontmatter extracts the description field and the body after the closing
// `---`. Minimal by design — the real YAML frontmatter parser lands with
// specmodel in Phase 3; this only needs `description`.
func splitFrontmatter(s string) (desc, body string) {
	if !strings.HasPrefix(s, "---") {
		return "", s
	}
	rest := s[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", s
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		if d, ok := strings.CutPrefix(strings.TrimSpace(line), "description:"); ok {
			desc = strings.TrimSpace(d)
			break
		}
	}
	body = strings.TrimLeft(rest[end+len("\n---"):], "-\n")
	return desc, body
}
