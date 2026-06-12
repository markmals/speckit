package project

import (
	"errors"
	"io/fs"
)

// Skill is a process-discipline skill (TDD, verification-before-completion,
// adversarial-review, …) projected into the agent's skills directory alongside
// the command prompts. Its content is agent-neutral SKILL.md markdown.
type Skill struct {
	Name    string
	Content string
}

// loadSkills reads templates/skills/<name>/SKILL.md from the embedded assets.
func loadSkills(assets fs.FS) ([]Skill, error) {
	entries, err := fs.ReadDir(assets, "templates/skills")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var skills []Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		b, err := fs.ReadFile(assets, "templates/skills/"+e.Name()+"/SKILL.md")
		if err != nil {
			return nil, err
		}
		skills = append(skills, Skill{Name: e.Name(), Content: string(b)})
	}
	return skills, nil
}
