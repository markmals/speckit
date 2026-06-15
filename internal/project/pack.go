package project

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// packSkill is one platform skill from templates/packs/<stack>/<name>/ — its
// whole directory (SKILL.md plus any references/ files), keyed by path relative
// to the skill dir, so multi-file skills keep their depth when projected.
type packSkill struct {
	Name  string
	Files map[string]string // relpath (e.g. "SKILL.md", "references/x.md") -> content
}

// packAgentsDir is the reserved subdir of a stack's pack that holds its per-stack
// agents (not skills) — projected into the agent's agents dir, where it has one.
const packAgentsDir = "agents"

// loadPack reads the platform skills for a stack from templates/packs/<stack>/.
// Each immediate subdir is a skill — except the reserved "agents" dir (see
// loadPackAgents) — and every file beneath a skill is carried verbatim (SKILL.md
// is required; references/ and any other nested files come along).
func loadPack(assets fs.FS, stack string) ([]packSkill, error) {
	dir := "templates/packs/" + stack
	entries, err := fs.ReadDir(assets, dir)
	if errors.Is(err, fs.ErrNotExist) {
		// A real scaffold may ship no pack (library stacks: swift-package, swift-cli);
		// only a stack that isn't a real scaffold is an error. This keeps `specify
		// packs` from failing on a packless target. The scaffold-layout path mirrors
		// cmd/specify (fs.Sub "templates/scaffolds/<stack>") + scaffold.LoadManifest
		// (reads "scaffold.json"); project/ doesn't import scaffold/, so it's duplicated.
		if _, serr := fs.Stat(assets, "templates/scaffolds/"+stack+"/scaffold.json"); serr == nil {
			return nil, nil // a real scaffold that ships no pack
		}
		return nil, fmt.Errorf("unknown pack %q", stack)
	}
	if err != nil {
		return nil, err
	}
	var skills []packSkill
	for _, e := range entries {
		if !e.IsDir() || e.Name() == packAgentsDir {
			continue
		}
		skillDir := dir + "/" + e.Name()
		files := map[string]string{}
		err := fs.WalkDir(assets, skillDir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			b, err := fs.ReadFile(assets, p)
			if err != nil {
				return err
			}
			files[strings.TrimPrefix(p, skillDir+"/")] = string(b)
			return nil
		})
		if err != nil {
			return nil, err
		}
		if _, ok := files["SKILL.md"]; !ok {
			return nil, fmt.Errorf("pack skill %q has no SKILL.md", e.Name())
		}
		skills = append(skills, packSkill{Name: e.Name(), Files: files})
	}
	return skills, nil
}

// loadPackAgents reads a stack's per-stack agents from
// templates/packs/<stack>/agents/<name>.md. A stack with no agents dir returns
// nil (not an error) — most stacks ship skills only.
func loadPackAgents(assets fs.FS, stack string) ([]Subagent, error) {
	dir := "templates/packs/" + stack + "/" + packAgentsDir
	entries, err := fs.ReadDir(assets, dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var agents []Subagent
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		b, err := fs.ReadFile(assets, dir+"/"+e.Name())
		if err != nil {
			return nil, err
		}
		agents = append(agents, Subagent{Name: strings.TrimSuffix(e.Name(), ".md"), Content: string(b)})
	}
	return agents, nil
}

// ProjectPacks projects the platform pack for the given stacks into the agent's
// dirs, returning the written paths. A stack's skills (whole directory, so
// references/ survive) go into the skills dir; its per-stack agents go into the
// agents dir, where the adapter has one (Claude-only today — codex, generic, and
// Copilot skip stack agents, like the review subagents). The agent comes from
// .speckit/specs.json's `agent`.
func ProjectPacks(root string, assets fs.FS, agentID string, stacks []string) ([]string, error) {
	adapter, ok := AdapterFor(agentID)
	if !ok {
		return nil, fmt.Errorf("unknown agent %q (have: %v)", agentID, AdapterIDs())
	}
	skillsDir := adapter.SkillsDir()
	if skillsDir == "" {
		return nil, fmt.Errorf("agent %q has no skills directory to project packs into", agentID)
	}
	agentsDir := adapter.AgentsDir() // "" for codex/generic/copilot — skip stack agents
	var written []string
	for _, stack := range stacks {
		skills, err := loadPack(assets, stack)
		if err != nil {
			return nil, err
		}
		for _, s := range skills {
			rels := make([]string, 0, len(s.Files))
			for rel := range s.Files {
				rels = append(rels, rel)
			}
			sort.Strings(rels) // deterministic write order
			for _, rel := range rels {
				dest := filepath.Join(root, skillsDir, s.Name, filepath.FromSlash(rel))
				w, err := writeFile(dest, []byte(s.Files[rel]))
				if err != nil {
					return nil, err
				}
				written = append(written, w)
			}
		}

		if agentsDir == "" {
			continue
		}
		agents, err := loadPackAgents(assets, stack)
		if err != nil {
			return nil, err
		}
		for _, a := range agents {
			w, err := writeFile(filepath.Join(root, agentsDir, a.Name+".md"), []byte(a.Content))
			if err != nil {
				return nil, err
			}
			written = append(written, w)
		}
	}
	return written, nil
}
