package specmodel

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// Finding is a single spec-library lint violation (domain.specmodel I1–I6).
type Finding struct {
	Path      string `json:"path"`
	Invariant string `json:"invariant"`
	Message   string `json:"message"`
}

// Lint checks the spec library against the domain.specmodel invariants and
// returns the findings, sorted by (path, invariant).
//
// SPEC: story.engine.scan
func Lint(specs []Spec) []Finding {
	findings := []Finding{} // non-nil so a clean library marshals to [], not null

	byID := map[SpecID][]string{}
	for _, s := range specs {
		byID[s.ID] = append(byID[s.ID], s.Path)
	}

	// I4 — ID uniqueness (one finding per duplicated id).
	for id, paths := range byID {
		if len(paths) > 1 {
			sort.Strings(paths)
			findings = append(findings, Finding{paths[0], "I4",
				fmt.Sprintf("duplicate id %q (also at %s)", id, strings.Join(paths[1:], ", "))})
		}
	}

	for _, s := range specs {
		// I2 — closed kind. Later checks rely on a valid kind, so stop here.
		if !s.Kind.Valid() {
			findings = append(findings, Finding{s.Path, "I2", fmt.Sprintf("unknown kind %q", s.Kind)})
			continue
		}
		// I3 — prefix agreement (the prefixless singular kinds are exempt).
		if pfx := s.Kind.Prefix(); pfx != "" && !strings.HasPrefix(string(s.ID), pfx) {
			findings = append(findings, Finding{s.Path, "I3",
				fmt.Sprintf("id %q does not start with kind prefix %q", s.ID, pfx)})
		}
		// I1 — filename ↔ id (singular files like CONVENTIONS/NARRATIVE are exempt).
		if !s.Kind.Singular() {
			stem := strings.TrimSuffix(path.Base(s.Path), ".md")
			if tail := idTail(s.ID); stem != tail {
				findings = append(findings, Finding{s.Path, "I1",
					fmt.Sprintf("filename stem %q does not match id tail %q", stem, tail)})
			}
		}
		// I5 — depends-on resolution.
		for _, d := range s.DependsOn {
			if _, ok := byID[d]; !ok {
				findings = append(findings, Finding{s.Path, "I5", fmt.Sprintf("dangling depends-on %q", d)})
			}
		}
		// I6 — story scenarios carry sub-ids.
		if s.Kind == KindStory {
			for _, sc := range s.Scenarios {
				if sc.SubID == "" {
					findings = append(findings, Finding{s.Path, "I6",
						fmt.Sprintf("scenario %q has no sub-id", sc.Heading)})
				}
			}
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Invariant < findings[j].Invariant
	})
	return findings
}

// idTail returns the id with its leading kind-prefix segment removed
// (story.engine.scan → engine.scan; domain.specmodel → specmodel).
func idTail(id SpecID) string {
	s := string(id)
	if i := strings.IndexByte(s, '.'); i >= 0 {
		return s[i+1:]
	}
	return s
}
