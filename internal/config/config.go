// Package config loads .speckit/specs.jsonc — the project's targets (each a
// product implementation with its test/report wiring) plus version/agent/paths.
//
// products and contracts are intentionally NOT modeled as first-class config
// yet (see docs/config.md for the rationale and the future shape). Today a
// product is expressed as an optional label on a target; contracts are not
// modeled at all until contract-drift is designed.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// File is the config path relative to the project root.
const File = ".speckit/specs.jsonc"

// Config is the parsed .speckit/specs.jsonc.
type Config struct {
	Version int               `json:"version"`
	Agent   string            `json:"agent"`
	Paths   Paths             `json:"paths"`
	Targets map[string]Target `json:"targets"`
}

// Paths locates the spec library (defaults: specs/ and features/).
type Paths struct {
	Specs    string `json:"specs"`
	Features string `json:"features"`
}

// Target is one implementation of a product: the test command to run (a shell
// string, à la a Mise task's `run`; empty when the report already exists), the
// report format/path the engine joins, the source dir scanned for bindings, and
// an optional product label. A target shared by several products lists them all
// via Products.
type Target struct {
	Product  string   `json:"product,omitempty"`
	Products []string `json:"products,omitempty"`
	Stack    string   `json:"stack,omitempty"` // selects the platform pack: web|apple|android|windows|linux|go-cli|node-cli|rust-cli|website
	Command  string   `json:"command,omitempty"`
	Format   string   `json:"format"` // junit | swift
	Report   string   `json:"report"`
	Source   string   `json:"source"`
}

// Load reads and parses .speckit/specs.jsonc under root. found is false (with a
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
	if err := json.Unmarshal(stripJSONC(b), &cfg); err != nil {
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

// Validate returns every problem with the config (nil = valid). scan surfaces
// these alongside the spec-library checks.
func (c Config) Validate() []error {
	var errs []error
	if len(c.Targets) == 0 {
		errs = append(errs, fmt.Errorf("no targets defined in %s", File))
	}
	for name, t := range c.Targets {
		switch t.Format {
		case "junit", "swift":
		case "":
			errs = append(errs, fmt.Errorf("target %q: missing format (junit|swift)", name))
		default:
			errs = append(errs, fmt.Errorf("target %q: unknown format %q (want junit|swift)", name, t.Format))
		}
		if t.Report == "" {
			errs = append(errs, fmt.Errorf("target %q: missing report path", name))
		}
		if t.Source == "" {
			errs = append(errs, fmt.Errorf("target %q: missing source dir", name))
		}
	}
	return errs
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
