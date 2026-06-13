package specmodel

import (
	"io/fs"
	"regexp"
	"strings"
)

// Spec is a parsed spec file from the library.
type Spec struct {
	Frontmatter
	Path      string     // slash path within the library FS
	Scenarios []Scenario // populated for stories
}

// Scenario is a Gherkin scenario heading and its declared sub-ID (empty if the
// sub-ID is missing — an I6 violation). Line is the 1-based line of the sub-ID
// declaration (the heading line if the sub-ID is absent) — used to point CI
// annotations at the scenario.
type Scenario struct {
	Heading string
	SubID   string
	Line    int
}

var scenarioSubID = regexp.MustCompile(`<!--\s*id:\s*(scenario\.[a-z0-9.\-]+)\s*-->`)

// ParseFrontmatter extracts a spec's frontmatter. ok is false when the content
// has no frontmatter or no id (e.g. a README) — such files are not specs.
//
// Minimal hand-parse (no YAML dependency): the convention's frontmatter is
// simple — scalar id/kind/status plus a flat depends-on list.
func ParseFrontmatter(content string) (fm Frontmatter, ok bool) {
	if !strings.HasPrefix(content, "---") {
		return Frontmatter{}, false
	}
	rest := content[len("---"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return Frontmatter{}, false
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "id:"):
			fm.ID = SpecID(strings.TrimSpace(line[len("id:"):]))
		case strings.HasPrefix(line, "kind:"):
			fm.Kind = Kind(strings.TrimSpace(line[len("kind:"):]))
		case strings.HasPrefix(line, "status:"):
			fm.Status = strings.TrimSpace(line[len("status:"):])
		case strings.HasPrefix(line, "depends-on:"):
			fm.DependsOn = parseIDList(line[len("depends-on:"):])
		}
	}
	if fm.ID == "" {
		return fm, false
	}
	return fm, true
}

func parseIDList(s string) []SpecID {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	var out []SpecID
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, SpecID(p))
		}
	}
	return out
}

// parseScenarios finds "## Scenario ..." headings and the sub-ID comment that
// follows each one before the next heading.
func parseScenarios(content string) []Scenario {
	lines := strings.Split(content, "\n")
	var out []Scenario
	for i, line := range lines {
		if !strings.HasPrefix(line, "## Scenario") {
			continue
		}
		sc := Scenario{Heading: strings.TrimSpace(strings.TrimPrefix(line, "##")), Line: i + 1}
		for j := i + 1; j < len(lines); j++ {
			if strings.HasPrefix(lines[j], "## ") {
				break
			}
			if m := scenarioSubID.FindStringSubmatch(lines[j]); m != nil {
				sc.SubID = m[1]
				sc.Line = j + 1 // point annotations at the sub-id declaration
				break
			}
		}
		out = append(out, sc)
	}
	return out
}

// LoadLibrary walks specs/ and features/ in fsys and parses every markdown file
// that carries frontmatter. Files without frontmatter (READMEs) are skipped.
//
// SPEC: domain.specmodel
func LoadLibrary(fsys fs.FS) ([]Spec, error) {
	var specs []Spec
	for _, dir := range []string{"specs", "features"} {
		if _, err := fs.Stat(fsys, dir); err != nil {
			continue
		}
		err := fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(p, ".md") {
				return nil
			}
			b, err := fs.ReadFile(fsys, p)
			if err != nil {
				return err
			}
			fm, ok := ParseFrontmatter(string(b))
			if !ok {
				return nil
			}
			s := Spec{Frontmatter: fm, Path: p}
			if fm.Kind == KindStory {
				s.Scenarios = parseScenarios(string(b))
			}
			specs = append(specs, s)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return specs, nil
}
