// Package config loads .speckit/specs.json — the project's targets (each a
// product implementation with its test/report wiring) plus version/agent/paths.
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

// Config is the parsed .speckit/specs.json.
type Config struct {
	Version int               `json:"version"`
	Agent   string            `json:"agent,omitempty"`
	Paths   Paths             `json:"paths"`
	Targets map[string]Target `json:"targets"`
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
// SourcePaths marshals as JSON null; Validate rejects empty before any Save, so
// that shape never reaches disk.
func (sp SourcePaths) MarshalJSON() ([]byte, error) {
	if len(sp) == 1 {
		return json.Marshal(sp[0])
	}
	return json.Marshal([]string(sp))
}

// First returns the first source path, or "" when there are none. Used by the
// deploy/secrets app-dir heuristic, which is single-app by nature.
func (sp SourcePaths) First() string {
	if len(sp) == 0 {
		return ""
	}
	return sp[0]
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

// Target is one implementation of a product: the test command to run (a shell
// string, à la a Mise task's `run`; empty when the report already exists), the
// report format/path the engine joins, the source dir scanned for bindings, and
// an optional product label. A target shared by several products lists them all
// via Products.
type Target struct {
	Product  string      `json:"product,omitempty"`
	Products []string    `json:"products,omitempty"`
	Stack    string      `json:"stack,omitempty"` // selects the pack/scaffold: web|website|apple|android|go-cli|go-service|node-cli|swift-package|swift-cli|ts-lib|vscode-extension
	Command  string      `json:"command,omitempty"`
	Format   string      `json:"format"` // junit | swift | gotest
	Report   string      `json:"report"`
	Source   SourcePaths `json:"source"`
	// Bindings is how untagged tests are treated: "strict" (default — every test
	// must bind a scenario) or "scoped" (untagged tests are out of scope, so a
	// suite mixing scenario tests with plain unit tests still verifies what it
	// binds). See engine.VerifyConfig.
	Bindings string  `json:"bindings,omitempty"`
	Deploy   *Deploy `json:"deploy,omitempty"`
}

// Deploy is a target's optional deploy manifest: which platform it ships to, and
// the secret references — 1Password op:// pointers, never values — to wire into CI
// and the platform's own runtime store (`specify deploy add` / `secrets sync`).
// Non-secret identifiers (e.g. CLOUDFLARE_ACCOUNT_ID) live in the stack's own
// config such as wrangler.jsonc, not here. See docs/design/github-integration.md.
type Deploy struct {
	Kind    string            `json:"kind"`
	CI      map[string]string `json:"ci,omitempty"`      // GitHub Actions secrets: ENV -> op:// ref
	Runtime map[string]string `json:"runtime,omitempty"` // platform runtime secrets: ENV -> op:// ref
}

// DeployKinds are the deploy platforms specify can wire. The matching workflow
// template lives at templates/deploy/<kind>/deploy.yml.tmpl.
var DeployKinds = []string{
	"cloudflare-workers-ssr",
	"cloudflare-workers-spa",
	"railway",
	"github-pages-spa",
	"app-store-connect",
}

// Load reads and parses .speckit/specs.json under root. found is false (with a
// nil error) when the file is absent — engine commands that need targets treat
// that as "configure your targets first"; scan treats it as nothing to validate.
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
	cfg.applyDefaults()
	return cfg, true, nil
}

func (c *Config) applyDefaults() {
	if c.Paths.Specs == "" {
		c.Paths.Specs = "specs"
	}
	if c.Paths.Features == "" {
		c.Paths.Features = "features"
	}
}

// Save writes the config to .speckit/specs.json (creating .speckit/ if needed).
func (c Config) Save(root string) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // keep && in shell commands readable, not &&
	enc.SetIndent("", "  ")
	if err := enc.Encode(c); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, ".speckit"), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, File), buf.Bytes(), 0o644)
}

// AddTarget loads the config (or starts a fresh v1 one), adds or replaces the
// named target, and writes it back. Used by `specify target add`.
func AddTarget(root, name string, t Target) error {
	cfg, found, err := Load(root)
	if err != nil {
		return err
	}
	if !found {
		cfg = Config{Version: 1}
		cfg.applyDefaults()
	}
	if cfg.Targets == nil {
		cfg.Targets = map[string]Target{}
	}
	cfg.Targets[name] = t
	return cfg.Save(root)
}

// SetAgent records the agent integration in .speckit/specs.json so `specify target
// add` and `specify packs` can project the stack packs (both are gated on a recorded
// agent). It creates a fresh v1 config (with default paths) when none exists and
// preserves any existing targets/paths otherwise (re-init / `init --here`). Used by
// `specify init`.
func SetAgent(root, agent string) error {
	cfg, found, err := Load(root)
	if err != nil {
		return err
	}
	if !found {
		cfg = Config{Version: 1}
	}
	cfg.applyDefaults()
	if cfg.Targets == nil {
		cfg.Targets = map[string]Target{}
	}
	cfg.Agent = agent
	return cfg.Save(root)
}

// SetDeploy attaches (or replaces) a target's deploy manifest and writes the
// config back. Errors if the target is unknown. Used by `specify deploy add`.
func SetDeploy(root, target string, d *Deploy) error {
	cfg, found, err := Load(root)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no %s — add the target first (specify target add)", File)
	}
	t, ok := cfg.Targets[target]
	if !ok {
		return fmt.Errorf("target %q not in %s", target, File)
	}
	t.Deploy = d
	cfg.Targets[target] = t
	return cfg.Save(root)
}

// Validate returns every problem with the config (nil = valid). scan surfaces
// these alongside the spec-library checks.
func (c Config) Validate() []error {
	var errs []error
	if len(c.Targets) == 0 {
		errs = append(errs, fmt.Errorf("no targets defined in %s", File))
	}
	for name, t := range c.Targets {
		switch t.Format {
		case "junit", "swift", "gotest":
		case "":
			errs = append(errs, fmt.Errorf("target %q: missing format (junit|swift|gotest)", name))
		default:
			errs = append(errs, fmt.Errorf("target %q: unknown format %q (want junit|swift|gotest)", name, t.Format))
		}
		switch t.Bindings {
		case "", "strict", "scoped":
		default:
			errs = append(errs, fmt.Errorf("target %q: unknown bindings mode %q (want strict|scoped)", name, t.Bindings))
		}
		if t.Report == "" {
			errs = append(errs, fmt.Errorf("target %q: missing report path", name))
		}
		errs = append(errs, t.Source.Validate(name)...)
		if t.Deploy != nil {
			errs = append(errs, t.Deploy.Validate(name)...)
		}
	}
	return errs
}

// Validate checks a deploy manifest: a known kind, and every secret a committable
// op:// reference (never a raw value). Exported so `specify deploy add` can reject
// a bad manifest before writing it.
func (d Deploy) Validate(target string) []error {
	var errs []error
	if !slices.Contains(DeployKinds, d.Kind) {
		errs = append(errs, fmt.Errorf("target %q: unknown deploy kind %q (want one of %s)", target, d.Kind, strings.Join(DeployKinds, ", ")))
	}
	for env, ref := range d.CI {
		errs = append(errs, validateSecretEntry(target, "ci", env, ref)...)
	}
	for env, ref := range d.Runtime {
		errs = append(errs, validateSecretEntry(target, "runtime", env, ref)...)
	}
	return errs
}

// validateSecretEntry checks one ENV→ref pair: a valid env-var name and a
// committable op:// reference (never a raw value).
func validateSecretEntry(target, section, env, ref string) []error {
	var errs []error
	if !validEnvName(env) {
		errs = append(errs, fmt.Errorf("target %q: deploy.%s key %q is not a valid env var name", target, section, env))
	}
	if !IsOpRef(ref) {
		errs = append(errs, fmt.Errorf("target %q: deploy.%s[%q] = %q is not an op:// reference (commit references, never secret values)", target, section, env, ref))
	}
	return errs
}

// validEnvName reports whether s is a usable env-var / secret name:
// [A-Za-z_][A-Za-z0-9_]* (not starting with a digit, no spaces or punctuation).
func validEnvName(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		ok := r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9' && i > 0)
		if !ok {
			return false
		}
	}
	return true
}

// IsOpRef reports whether s is a 1Password secret reference: op://vault/item/field
// or op://vault/item/section/field — 3 or 4 non-empty path segments. Exported so
// `specify deploy add` can reject a raw value before it ever lands in committed
// config.
func IsOpRef(s string) bool {
	rest, ok := strings.CutPrefix(s, "op://")
	if !ok {
		return false
	}
	parts := strings.Split(rest, "/")
	if len(parts) < 3 || len(parts) > 4 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
	}
	return true
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

// Stacks lists the distinct, non-empty target stacks (which platform packs
// `specify packs` should project).
func (c Config) Stacks() []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range c.Targets {
		if t.Stack != "" && !seen[t.Stack] {
			seen[t.Stack] = true
			out = append(out, t.Stack)
		}
	}
	sort.Strings(out)
	return out
}

func (t Target) productLabels() []string {
	var ps []string
	if t.Product != "" {
		ps = append(ps, t.Product)
	}
	return append(ps, t.Products...)
}
