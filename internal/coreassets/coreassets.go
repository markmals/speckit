// Package coreassets embeds the fork's agent-neutral source assets — the
// command prompts and templates that `specify init` projects per agent (D4).
// The prompts are data (cheap to take from upstream, then adapted); the
// projection behavior is implemented from spec in internal/project.
package coreassets

import "embed"

//go:embed all:templates
var FS embed.FS
