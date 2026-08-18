// Package project implements `specify init` — the per-agent projection
// adapters (D2/D4), built from the 0002-init specs.
package project

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/markmals/speckit/internal/config"
)

// Options controls Init.
type Options struct {
	Integration string // claude | codex | copilot | generic
	Force       bool   // proceed even if root is non-empty
}

// Init sets up a SpecKit project at root for the chosen agent, returning the
// paths written. The fork's runtime lives under .speckit/ with no shell scripts
// (D2/D6) and no banner (D1).
//
// SPEC: story.init.basic
func Init(root string, assets fs.FS, opts Options) ([]string, error) {
	adapter, ok := AdapterFor(opts.Integration)
	if !ok {
		return nil, fmt.Errorf("unknown integration %q (have: %v)", opts.Integration, AdapterIDs())
	}

	// SPEC: story.init.basic (scenario.init.basic.non-empty-guard)
	if !opts.Force {
		nonEmpty, err := dirNonEmpty(root)
		if err != nil {
			return nil, err
		}
		if nonEmpty {
			return nil, fmt.Errorf("%s is not empty (use --force)", root)
		}
	}

	commands, err := loadCommands(assets)
	if err != nil {
		return nil, err
	}

	// SPEC: story.init.projection (scenario.init.projection.fork-divergence)
	// The runtime dir is .speckit/, never .specify/; no scripts on disk (D2/D6).
	written, err := writeRuntime(root, assets)
	if err != nil {
		return nil, err
	}

	projected, err := adapter.Project(root, commands)
	if err != nil {
		return nil, err
	}
	written = append(written, projected...)

	// Project the process-discipline skills into the agent's skills dir (D3
	// process-pack), where the agent has one.
	if dir := adapter.SkillsDir(); dir != "" {
		skills, err := loadSkills(assets)
		if err != nil {
			return nil, err
		}
		for _, s := range skills {
			w, err := writeFile(filepath.Join(root, dir, s.Name, "SKILL.md"), []byte(s.Content))
			if err != nil {
				return nil, err
			}
			written = append(written, w)
		}
	}

	// Project the review subagents into the agent's agents dir (the claude-pack),
	// where the agent has one.
	if dir := adapter.AgentsDir(); dir != "" {
		subs, err := loadSubagents(assets)
		if err != nil {
			return nil, err
		}
		for _, s := range subs {
			w, err := writeFile(filepath.Join(root, dir, s.Name+".md"), []byte(s.Content))
			if err != nil {
				return nil, err
			}
			written = append(written, w)
		}
	}

	// Project the always-loaded guidance rules into the agent's rules dir; the
	// orientation file (CLAUDE.md / AGENTS.md / copilot-instructions.md) points at
	// them.
	if dir := adapter.RulesDir(); dir != "" {
		rules, err := loadRules(assets)
		if err != nil {
			return nil, err
		}
		for _, r := range rules {
			w, err := writeFile(filepath.Join(root, dir, r.Name+".md"), []byte(r.Content))
			if err != nil {
				return nil, err
			}
			written = append(written, w)
		}
	}

	// Seed the agent's repo-local memory/ store (its MEMORY.md index), where the
	// agent has one. Skip-if-absent so re-init never clobbers accumulated memory;
	// the orientation file (above) wires loading. Memory is agent-owned working
	// knowledge — the engine never reads it (docs/design/agent-memory.md).
	if dir := adapter.MemoryDir(); dir != "" {
		seed, err := loadMemorySeed(assets)
		if err != nil {
			return nil, err
		}
		w, err := writeFileIfAbsent(filepath.Join(root, dir, "MEMORY.md"), seed)
		if err != nil {
			return nil, err
		}
		if w != "" {
			written = append(written, w)
		}
	}

	// Record the chosen agent in .speckit/specs.json so projected guidance knows
	// which agent owns the runtime dirs. Preserves an existing config's targets
	// (re-init / `init --here`).
	if err := config.SetAgent(root, opts.Integration); err != nil {
		return nil, err
	}
	written = append(written, filepath.Join(root, config.File))

	return written, nil
}

func dirNonEmpty(root string) (bool, error) {
	f, err := os.Open(root)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer f.Close()
	names, _ := f.Readdirnames(1) // io.EOF on empty dir is fine
	return len(names) > 0, nil
}
