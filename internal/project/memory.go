package project

import "io/fs"

// loadMemorySeed reads templates/memory/MEMORY.md — the seed index written into
// an agent's memory/ dir on init. Topic files are agent-authored thereafter; the
// engine never reads any of it (memory is agent-owned working knowledge, not spec
// truth). See docs/design/agent-memory.md and the managing-memory skill.
func loadMemorySeed(assets fs.FS) ([]byte, error) {
	return fs.ReadFile(assets, "templates/memory/MEMORY.md")
}
