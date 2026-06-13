package project

import (
	"errors"
	"io/fs"
	"strings"
)

// Rule is an always-loaded guidance file (code quality, commit discipline, spec
// conventions, the enforcement hierarchy) projected into the agent's rules dir.
// Content is agent-neutral markdown written verbatim.
type Rule struct {
	Name    string
	Content string
}

// loadRules reads templates/rules/<name>.md from the embedded assets.
func loadRules(assets fs.FS) ([]Rule, error) {
	entries, err := fs.ReadDir(assets, "templates/rules")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rules []Rule
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		b, err := fs.ReadFile(assets, "templates/rules/"+e.Name())
		if err != nil {
			return nil, err
		}
		rules = append(rules, Rule{Name: strings.TrimSuffix(e.Name(), ".md"), Content: string(b)})
	}
	return rules, nil
}
