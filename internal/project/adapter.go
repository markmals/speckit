package project

// Adapter projects the command set + orientation file into an agent's native
// format and location (D4). Adding an agent is implementing this interface —
// the shared command prompts are unchanged.
//
// SPEC: story.init.projection
type Adapter interface {
	// ID is the integration id, e.g. "claude".
	ID() string
	// Project writes the commands + orientation file under root, returning the
	// paths it wrote.
	Project(root string, commands []Command) ([]string, error)
	// SkillsDir is the directory (relative to root, slash-separated) where
	// process-discipline skills are projected, or "" if the agent has no skills
	// concept.
	SkillsDir() string
	// AgentsDir is the directory (relative to root, slash-separated) where
	// projectable agents are written (review subagents from init, stack agents
	// from packs), or "" if the agent has no subagent/agent-file concept. Today
	// dispatched subagents are a Claude Code feature (the claude-pack).
	AgentsDir() string
	// RulesDir is the directory (relative to root, slash-separated) where the
	// always-loaded guidance rules are projected, or "" if the agent has none.
	RulesDir() string
	// MemoryDir is the directory (relative to root, slash-separated) where the
	// agent's repo-local memory/ store is projected (its seed MEMORY.md index),
	// or "" if the agent has none. Memory is agent-owned working knowledge the
	// engine never reads — see docs/design/agent-memory.md.
	MemoryDir() string
}

var adapters = map[string]Adapter{}

func register(a Adapter) { adapters[a.ID()] = a }

// AdapterFor returns the adapter for an integration id.
func AdapterFor(id string) (Adapter, bool) {
	a, ok := adapters[id]
	return a, ok
}

// AdapterIDs lists the registered integration ids.
func AdapterIDs() []string {
	ids := make([]string, 0, len(adapters))
	for id := range adapters {
		ids = append(ids, id)
	}
	return ids
}
