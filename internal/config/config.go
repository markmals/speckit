// Package config loads .speckit/specs.json — the project's targets (each a
// product implementation with its test/report wiring), the optional reference
// target, and the work-tracking provider.
//
// The config is stack-agnostic on purpose: a target is described by where it
// lives, how to run its tests, and how to read the resulting report. Nothing
// here names a framework, runtime, or platform.
//
// products and contracts are intentionally NOT modeled as first-class config
// yet (see docs/config.md for the rationale and the future shape). Today a
// product is expressed as an optional label on a target; contracts are not
// modeled at all until contract-drift is designed.
package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// File is the config path relative to the project root.
const File = ".speckit/specs.json"

// SchemaVersion is the version this build writes. Older files load (their
// retired keys are ignored with a notice) and are rewritten at this version on
// the next Save.
const SchemaVersion = 2

// Config is the parsed .speckit/specs.json.
type Config struct {
	Version int    `json:"version"`
	Agent   string `json:"agent,omitempty"`
	// ReferenceTarget names the target whose behavior other targets match when a
	// spec is ambiguous across them. Purely informational — the engine privileges
	// no target — but projected agent guidance reads it instead of hardcoding a
	// platform.
	ReferenceTarget string            `json:"reference_target,omitempty"`
	Paths           Paths             `json:"paths"`
	Targets         map[string]Target `json:"targets"`
	Work            *Work             `json:"work,omitempty"`

	// Notice is a non-fatal load-time message (an older schema version, keys this
	// build no longer honors). Never serialized: it describes the file as read,
	// not the config as written.
	Notice string `json:"-"`
}

// Paths locates the spec library (defaults: specs/ and features/).
type Paths struct {
	Specs    string `json:"specs"`
	Features string `json:"features"`
}

// SourcePaths is one or more source directories scanned for scenario bindings.
// It decodes from either a JSON string (one dir) or a JSON array of dirs, so a
// target whose bound tests span several packages can list them all while a
// single-dir target stays a bare string. See docs/design/multi-source-targets.md.
type SourcePaths []string

// UnmarshalJSON accepts a JSON string ("x") or a JSON array of strings
// (["a","b"]). Each entry is trimmed of surrounding whitespace; blank entries
// are kept so Validate can report them.
func (sp *SourcePaths) UnmarshalJSON(b []byte) error {
	var one string
	if err := json.Unmarshal(b, &one); err == nil {
		*sp = SourcePaths{strings.TrimSpace(one)}
		return nil
	}
	var many []string
	if err := json.Unmarshal(b, &many); err != nil {
		return fmt.Errorf("source must be a string or an array of strings: %w", err)
	}
	out := make(SourcePaths, len(many))
	for i, s := range many {
		out[i] = strings.TrimSpace(s)
	}
	*sp = out
	return nil
}

// MarshalJSON writes one path as a bare string and multiple as an array, so an
// existing single-source config round-trips to the same on-disk shape. An empty
// SourcePaths marshals as []/null (nil → null, empty slice → []); Validate
// rejects empty before any Save, so that shape never reaches disk.
func (sp SourcePaths) MarshalJSON() ([]byte, error) {
	if len(sp) == 1 {
		return json.Marshal(sp[0])
	}
	return json.Marshal([]string(sp))
}

// Validate reports source problems for the named target: no paths at all, or any
// blank/whitespace-only entry.
func (sp SourcePaths) Validate(target string) []error {
	if len(sp) == 0 {
		return []error{fmt.Errorf("target %q: missing source dir", target)}
	}
	var errs []error
	for _, s := range sp {
		if strings.TrimSpace(s) == "" {
			errs = append(errs, fmt.Errorf("target %q: source contains a blank entry", target))
		}
	}
	return errs
}

// Formats are the report formats the engine can read. Adding one is a change to
// internal/reports plus this list — see docs/report-formats.md.
var Formats = []string{"junit", "swift", "gotest"}

// Target is one implementation of a product: where it lives, the test command to
// run (a shell string, à la a Mise task's `run`; empty when the report already
// exists), the report format/path the engine joins, the source dirs scanned for
// bindings, and an optional product label. A target shared by several products
// lists them all via Products.
type Target struct {
	Product  string   `json:"product,omitempty"`
	Products []string `json:"products,omitempty"`
	// Dir is the target's root, relative to the project root. Informational —
	// nothing is rendered into it — but it records what the target *is* now that
	// no platform label does.
	Dir     string      `json:"dir"`
	Command string      `json:"command,omitempty"`
	Format  string      `json:"format"`
	Report  string      `json:"report"`
	Source  SourcePaths `json:"source"`
	// Bindings is how untagged tests are treated: "strict" (default — every test
	// must bind a scenario) or "scoped" (untagged tests are out of scope, so a
	// suite mixing scenario tests with plain unit tests still verifies what it
	// binds). See engine.VerifyConfig.
	Bindings string `json:"bindings,omitempty"`
}

// Work provider ids. markdown is the default: a committed file, no network, no
// external binary. none disables the surface entirely.
const (
	WorkMarkdown       = "markdown"
	WorkBeads          = "beads"
	WorkGitHubProjects = "github-projects"
	WorkNone           = "none"
)

// WorkProviders are the work-tracking backends `specify work` can drive.
var WorkProviders = []string{WorkMarkdown, WorkBeads, WorkGitHubProjects, WorkNone}

// DefaultWorkFile is the markdown provider's committed file.
const DefaultWorkFile = "WORK.md"

// Work selects and configures the work-tracking provider. Absent from the config
// entirely, the markdown provider is used — so `specify work` needs no setup, and
// no engine command needs this block at all.
type Work struct {
	Provider string `json:"provider"`
	// File is the markdown provider's committed work file (default WORK.md).
	File string `json:"file,omitempty"`
	// Project and Owner address a GitHub Projects v2 board. Owner defaults to the
	// resolved repo's owner.
	Project int    `json:"project,omitempty"`
	Owner   string `json:"owner,omitempty"`
}

// Validate checks a work block: a known provider, and the fields that provider
// requires.
func (w Work) Validate() []error {
	if !slices.Contains(WorkProviders, w.Provider) {
		return []error{fmt.Errorf("work: unknown provider %q (want one of %s)", w.Provider, strings.Join(WorkProviders, ", "))}
	}
	if w.Provider == WorkGitHubProjects && w.Project <= 0 {
		return []error{fmt.Errorf(`work: provider %q needs a positive "project" (the board number)`, WorkGitHubProjects)}
	}
	return nil
}

// WorkConfig returns the work block with defaults applied. An absent block means
// the markdown provider on DefaultWorkFile.
func (c Config) WorkConfig() Work {
	w := Work{Provider: WorkMarkdown}
	if c.Work != nil {
		w = *c.Work
	}
	if w.Provider == "" {
		w.Provider = WorkMarkdown
	}
	if w.Provider == WorkMarkdown && strings.TrimSpace(w.File) == "" {
		w.File = DefaultWorkFile
	}
	return w
}

// Reference resolves the reference target: the explicit reference_target, else
// the only target when there is exactly one (nothing else could be the
// reference), else "" — no target is privileged.
func (c Config) Reference() string {
	if c.ReferenceTarget != "" {
		return c.ReferenceTarget
	}
	if len(c.Targets) == 1 {
		for name := range c.Targets {
			return name
		}
	}
	return ""
}

// Load reads and parses .speckit/specs.json under root. found is false (with a
// nil error) when the file is absent — engine commands that need targets treat
// that as "configure your targets first"; scan treats it as nothing to validate.
//
// An older schema version, or keys this build no longer honors, load fine and
// surface as cfg.Notice. A config file is never a hard failure for being old.
func Load(root string) (cfg Config, found bool, err error) {
	b, err := os.ReadFile(filepath.Join(root, File))
	if os.IsNotExist(err) {
		return Config{}, false, nil
	}
	if err != nil {
		return Config{}, false, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, true, fmt.Errorf("%s: %w", File, err)
	}
	cfg.Notice = legacyNotice(b)
	cfg.applyDefaults()
	return cfg, true, nil
}

// retiredKeys are per-target keys earlier versions honored and this one ignores:
// `stack` selected a scaffold and skill pack, `deploy` a deployment platform.
// Both are the adopting project's business, not the spec engine's.
var retiredKeys = []string{"stack", "deploy"}

// legacyNotice describes an out-of-date config in one line, or returns "" when
// the file is current. It reports rather than rejects: an unmigrated project must
// still be able to run every engine command.
func legacyNotice(b []byte) string {
	var raw struct {
		Version int                        `json:"version"`
		Targets map[string]json.RawMessage `json:"targets"`
	}
	if json.Unmarshal(b, &raw) != nil {
		return ""
	}
	found := map[string]bool{}
	for _, t := range raw.Targets {
		var fields map[string]json.RawMessage
		if json.Unmarshal(t, &fields) != nil {
			continue
		}
		for _, k := range retiredKeys {
			if _, ok := fields[k]; ok {
				found[k] = true
			}
		}
	}
	if len(found) == 0 && raw.Version >= SchemaVersion {
		return ""
	}
	msg := fmt.Sprintf("%s: schema v%d (this build writes v%d)", File, raw.Version, SchemaVersion)
	if len(found) > 0 {
		msg += "; ignoring retired keys: " + strings.Join(sortedKeys(found), ", ")
	}
	return msg + " — see MIGRATION.md"
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (c *Config) applyDefaults() {
	if c.Paths.Specs == "" {
		c.Paths.Specs = "specs"
	}
	if c.Paths.Features == "" {
		c.Paths.Features = "features"
	}
	// A target written before `dir` existed is rooted at the project root.
	for name, t := range c.Targets {
		if t.Dir == "" {
			t.Dir = "."
			c.Targets[name] = t
		}
	}
}

// Save writes the config to .speckit/specs.json (creating .speckit/ if needed),
// normalizing it to the current schema version.
func (c Config) Save(root string) error {
	c.Version = SchemaVersion
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // keep && in shell commands readable, not \u0026\u0026
	enc.SetIndent("", "  ")
	if err := enc.Encode(c); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, ".speckit"), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, File), buf.Bytes(), 0o644)
}

// AddTarget loads the config (or starts a fresh one), adds or replaces the named
// target, and writes it back. Used by `specify target add`.
func AddTarget(root, name string, t Target) error {
	cfg, _, err := Load(root)
	if err != nil {
		return err
	}
	cfg.applyDefaults()
	if cfg.Targets == nil {
		cfg.Targets = map[string]Target{}
	}
	cfg.Targets[name] = t
	return cfg.Save(root)
}

// SetAgent records the agent integration in .speckit/specs.json. It creates a
// fresh config (with default paths) when none exists and preserves any existing
// targets/paths otherwise (re-init / `init --here`). Used by `specify init`.
func SetAgent(root, agent string) error {
	cfg, _, err := Load(root)
	if err != nil {
		return err
	}
	cfg.applyDefaults()
	if cfg.Targets == nil {
		cfg.Targets = map[string]Target{}
	}
	cfg.Agent = agent
	return cfg.Save(root)
}

// SetReferenceTarget records which target other targets match when a spec is
// ambiguous across them. The target must already exist.
func SetReferenceTarget(root, name string) error {
	cfg, found, err := Load(root)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no %s — add the target first (specify target add)", File)
	}
	if _, ok := cfg.Targets[name]; !ok {
		return fmt.Errorf("target %q not in %s", name, File)
	}
	cfg.ReferenceTarget = name
	return cfg.Save(root)
}

// Validate returns every problem with the config (nil = valid). scan surfaces
// these alongside the spec-library checks.
func (c Config) Validate() []error {
	var errs []error
	if len(c.Targets) == 0 {
		errs = append(errs, fmt.Errorf("no targets defined in %s", File))
	}
	for _, name := range c.TargetNames() {
		errs = append(errs, c.Targets[name].validate(name)...)
	}
	if c.ReferenceTarget != "" {
		if _, ok := c.Targets[c.ReferenceTarget]; !ok {
			errs = append(errs, fmt.Errorf("reference_target %q is not a defined target in %s", c.ReferenceTarget, File))
		}
	}
	if c.Work != nil {
		errs = append(errs, c.Work.Validate()...)
	}
	return errs
}

func (t Target) validate(name string) []error {
	var errs []error
	switch {
	case t.Format == "":
		errs = append(errs, fmt.Errorf("target %q: missing format (%s)", name, strings.Join(Formats, "|")))
	case !slices.Contains(Formats, t.Format):
		errs = append(errs, fmt.Errorf("target %q: unknown format %q (want %s)", name, t.Format, strings.Join(Formats, "|")))
	}
	switch t.Bindings {
	case "", "strict", "scoped":
	default:
		errs = append(errs, fmt.Errorf("target %q: unknown bindings mode %q (want strict|scoped)", name, t.Bindings))
	}
	if t.Report == "" {
		errs = append(errs, fmt.Errorf("target %q: missing report path", name))
	}
	if strings.TrimSpace(t.Dir) == "" {
		errs = append(errs, fmt.Errorf("target %q: missing dir", name))
	}
	return append(errs, t.Source.Validate(name)...)
}

// TargetNames lists the configured target names, sorted.
func (c Config) TargetNames() []string {
	names := make([]string, 0, len(c.Targets))
	for name := range c.Targets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ProductTargets maps each product label to the targets that carry it. A target
// with no label is omitted; a shared target appears under each of its products.
func (c Config) ProductTargets() map[string][]string {
	m := map[string][]string{}
	for name, t := range c.Targets {
		for _, p := range t.productLabels() {
			m[p] = append(m[p], name)
		}
	}
	return m
}

func (t Target) productLabels() []string {
	var ps []string
	if t.Product != "" {
		ps = append(ps, t.Product)
	}
	return append(ps, t.Products...)
}
