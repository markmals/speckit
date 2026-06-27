// Package scaffold renders a stack's starter template — a scaffold.json manifest
// plus a files/ tree — into a target directory via Go text/template.
// See docs/design/stack-scaffolding.md.
package scaffold

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"unicode"
)

// Manifest is a scaffold's scaffold.json.
type Manifest struct {
	Stack string `json:"stack"`
	// MemberDir is where `target add` places this stack's member by default
	// (overridable with --dir): "apps" for an app, "cmd" for a go-service, "packages"
	// for a library. Empty defaults to "apps".
	MemberDir string `json:"memberDir,omitempty"`
	// NameRule constrains the member name beyond the CLI's base slug check (D-naming).
	// "identifier" requires that pascal(name) be a valid source identifier — for stacks
	// that render {{pascal .Name}} as a module/type name (the swift stacks, apple), where
	// not every safe slug works (e.g. a leading digit). Empty = the base rule only.
	NameRule string `json:"nameRule,omitempty"`
	// Family groups stacks that share a mise monorepo contribution — toolchain
	// pins (hoisted to the root mise.toml) and [task_templates] (hoisted once the
	// family has two members). e.g. "node" (web), "swift" (apple/swift-package/
	// swift-cli), "go" (go-service). Empty = the stack contributes no family.
	Family string `json:"family,omitempty"`
	// SharedModule makes members of this stack compose into ONE repo-root go.mod
	// (each member a cmd/<name> sharing internal/ packages — trove's shape) rather
	// than a self-contained module per member. `target add` creates the root go.mod
	// if the repo isn't a module yet; members render no go.mod of their own.
	SharedModule   bool               `json:"sharedModule,omitempty"`
	Scripts        []Script           `json:"scripts,omitempty"`
	Target         ManifestTarget     `json:"target"`
	Variables      []Variable         `json:"variables,omitempty"`
	Features       map[string]Feature `json:"features,omitempty"`
	DataDefault    string             `json:"dataDefault,omitempty"`    // the --data kind used when none is given
	Data           map[string]Variant `json:"data,omitempty"`           // selectable data layers (--data <kind>)
	RuntimeDefault string             `json:"runtimeDefault,omitempty"` // the --runtime kind used when none is given
	Runtime        map[string]Variant `json:"runtime,omitempty"`        // selectable runtimes (--runtime <kind>)
}

// Variant is one selectable axis option — a data layer (--data) or a runtime
// (--runtime). Its Files subtree is rendered OVER the base (overwriting shared
// files like router.tsx / vite.config.ts), its deps are pnpm-added, and its
// Scripts (e.g. codegen) run after the base install. RequiresRuntime, when set,
// constrains the option to a runtime (e.g. drizzle's D1 driver requires cloudflare).
type Variant struct {
	Files           string   `json:"files,omitempty"`
	Add             []string `json:"add,omitempty"`    // runtime deps
	AddDev          []string `json:"addDev,omitempty"` // dev deps
	Scripts         []Script `json:"scripts,omitempty"`
	RequiresRuntime string   `json:"requiresRuntime,omitempty"`
	// RuntimeFiles overlays a runtime-specific subtree on top of Files, keyed by the
	// resolved --runtime (e.g. drizzle ships a D1 db module for cloudflare and a
	// node:sqlite one for node). Rendered after Files.
	RuntimeFiles map[string]string `json:"runtimeFiles,omitempty"`
}

// DataKinds lists the manifest's data-variant kinds, sorted.
func (m Manifest) DataKinds() []string { return variantKinds(m.Data) }

// RuntimeKinds lists the manifest's runtime-variant kinds, sorted.
func (m Manifest) RuntimeKinds() []string { return variantKinds(m.Runtime) }

func variantKinds(m map[string]Variant) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Script is a phase of post-render setup: shell commands the CLI runs in the
// scaffolded target dir after its files are written. Phases run in ascending
// order; commands within a script run in sequence. A Silent script's failures
// (and output) are swallowed — for best-effort steps like a codegen that needs
// a prior login.
//
// This is the mechanism that lets a scaffold resolve values by *running a tool*
// instead of freezing them into a template: the canonical case is `pnpm add`,
// which makes the package manager resolve each dependency to its latest version
// and pin it into package.json at scaffold time — so templates never hardcode a
// version. The same mechanism carries framework codegen (router/wrangler/convex
// typegen) and formatting. Modeled on create-sprinkles' phased runScripts.
type Script struct {
	Commands []string `json:"commands"`
	Phase    int      `json:"phase"`
	Silent   bool     `json:"silent,omitempty"`
}

// ManifestTarget holds the specs.json target fields as text/template strings
// (e.g. "{{.Dir}}/junit.xml"), resolved against the Data by RenderTarget.
type ManifestTarget struct {
	Command  string `json:"command,omitempty"`
	Format   string `json:"format"`
	Report   string `json:"report"`
	Source   string `json:"source"`
	Bindings string `json:"bindings,omitempty"` // "strict" (default) | "scoped"
}

// Variable is a manifest-declared template variable (resolved by the CLI).
type Variable struct {
	Name    string `json:"name"`
	Default string `json:"default,omitempty"`
	From    string `json:"from,omitempty"` // "flag" | "git"
}

// Feature is an optional --with add-on: an extra files subtree (rendered over the
// base), optional vars, and — like a Variant — its own deps + scripts.
type Feature struct {
	Files   string            `json:"files,omitempty"`
	Vars    map[string]string `json:"vars,omitempty"`
	Add     []string          `json:"add,omitempty"`
	AddDev  []string          `json:"addDev,omitempty"`
	Scripts []Script          `json:"scripts,omitempty"`
}

// Data is the text/template context every scaffold sees.
type Data struct {
	Name    string
	Dir     string
	Product string
	// Module is the repo-root Go module path (e.g. github.com/markmals/trove) for
	// SharedModule stacks, so a member can import its own generated/internal
	// packages by full path (`{{.Module}}/{{.Dir}}/internal/api`). Empty otherwise.
	Module   string
	Vars     map[string]string
	Features map[string]bool
}

// RenderedTarget is a manifest's target after substitution — the specs.json entry.
type RenderedTarget struct{ Command, Format, Report, Source, Bindings string }

var funcs = template.FuncMap{
	"lower":  strings.ToLower,
	"upper":  strings.ToUpper,
	"kebab":  kebab,
	"pascal": pascal,
	"camel":  camel,
}

// LoadManifest reads scaffold.json from a scaffold's root fs.
func LoadManifest(src fs.FS) (Manifest, error) {
	b, err := fs.ReadFile(src, "scaffold.json")
	if err != nil {
		return Manifest{}, fmt.Errorf("scaffold.json: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return Manifest{}, fmt.Errorf("scaffold.json: %w", err)
	}
	return m, nil
}

// npmReserved is the set of names npm rejects for a NEW package (validate-npm-package-name's
// validForNewPackages): the two blacklisted names plus the Node.js core modules (a
// core-module collision is a validForNewPackages=false warning). Bare names only — the
// '_'-prefixed, 'node:'-prefixed, and '/'-bearing builtins can't pass the base slug check
// in the first place. This is a conservative snapshot (Node 22/24, removed names like `sys`
// omitted): the list only ever grows, so a stale entry fails OPEN — a brand-new builtin
// slips through to publish time, but a valid name is never falsely rejected.
var npmReserved = map[string]bool{
	"node_modules": true, "favicon.ico": true,
	"assert": true, "async_hooks": true, "buffer": true, "child_process": true,
	"cluster": true, "console": true, "constants": true, "crypto": true,
	"dgram": true, "diagnostics_channel": true, "dns": true, "domain": true,
	"events": true, "fs": true, "http": true, "http2": true, "https": true,
	"inspector": true, "module": true, "net": true, "os": true, "path": true,
	"perf_hooks": true, "process": true, "punycode": true, "querystring": true,
	"readline": true, "repl": true, "stream": true, "string_decoder": true,
	"timers": true, "tls": true, "trace_events": true, "tty": true, "url": true,
	"util": true, "v8": true, "vm": true, "wasi": true, "worker_threads": true,
	"zlib": true,
}

// ValidateName checks a member name against the manifest's NameRule, a constraint
// beyond the CLI's base slug check (`target add` calls it after LoadManifest, before
// any render). The rule exists because a stack may render {{pascal .Name}} directly as
// a source identifier (a Swift module / type), and not every safe slug pascal-cases
// into a valid one. Returns a descriptive, actionable error when the name can't satisfy
// the rule. `target register` does NOT apply it — it writes no files, so it records
// whatever member already exists.
func (m Manifest) ValidateName(name string) error {
	switch m.NameRule {
	case "", "default":
		return nil
	case "identifier":
		// pascal() strips separators and keeps only alphanumerics, so its output is
		// always [A-Za-z0-9]* — the only way it fails to be a valid identifier is a
		// leading digit (e.g. "3d-tool" -> "3dTool") or an empty result ("--" -> "").
		p := pascal(name)
		if p == "" || !unicode.IsLetter([]rune(p)[0]) {
			return fmt.Errorf(
				"stack %q needs a name that pascal-cases into a valid identifier, but %q -> %q can't be a module/type name (an identifier must start with a letter) — try a letter-led name",
				m.Stack, name, p)
		}
		return nil
	case "npm":
		// The member name becomes the npm package name (package.json "name": "{{.Name}}"),
		// so it must be publishable. The base slug check (validTargetName) runs first and
		// already excludes spaces, '@', '/', leading '.'/'_'/'-', and other non-URL-friendly
		// chars; this adds the npm-specific gates a safe slug still slips through, so the
		// failure surfaces at `target add` instead of at `npm publish`.
		if low := strings.ToLower(name); low != name {
			return fmt.Errorf(
				"stack %q needs an npm-publishable name, but %q has capital letters — npm package names must be lowercase (try %q)",
				m.Stack, name, low)
		}
		if len(name) > 214 {
			return fmt.Errorf(
				"stack %q needs an npm-publishable name, but %q is %d characters — npm package names must be 214 or fewer",
				m.Stack, name, len(name))
		}
		if npmReserved[name] {
			return fmt.Errorf(
				"stack %q can't use %q — npm rejects it for a new package (a blacklisted name or a Node.js core module); pick a distinct name",
				m.Stack, name)
		}
		return nil
	default:
		return fmt.Errorf("scaffold.json: unknown nameRule %q", m.NameRule)
	}
}

// PhasedScripts returns the manifest's scripts in ascending phase order (stable
// within a phase), with every command rendered against data through the same
// text/template engine the files use. The CLI runs them in the scaffolded
// target dir after Render — see cmd/specify targetAddCmd.
func (m Manifest) PhasedScripts(data Data) ([]Script, error) {
	ordered := make([]Script, len(m.Scripts))
	copy(ordered, m.Scripts)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Phase < ordered[j].Phase })
	for si := range ordered {
		cmds := make([]string, len(ordered[si].Commands))
		for ci, c := range ordered[si].Commands {
			r, err := renderString("script", c, data)
			if err != nil {
				return nil, err
			}
			cmds[ci] = r
		}
		ordered[si].Commands = cmds
	}
	return ordered, nil
}

// Render walks the scaffold's files/ subtree, renders *.tmpl through
// text/template (stripping the suffix) and copies everything else verbatim into
// destDir. Returns the written paths (relative to destDir).
func Render(src fs.FS, destDir string, data Data) ([]string, error) {
	return renderSubtree(src, "files", destDir, data, false)
}

// renderSubtree renders one subtree (files/ or a feature's dir) into destDir.
// When skipExisting is set, a destination file that already exists is left
// untouched (and omitted from the returned paths) — so a shared subtree like
// github/ never clobbers config the repo already has.
func renderSubtree(src fs.FS, root, destDir string, data Data, skipExisting bool) ([]string, error) {
	var written []string
	err := fs.WalkDir(src, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(p, root), "/")
		if rel == "" {
			return nil
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(destDir, rel), 0o755)
		}
		b, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		out := rel
		if strings.HasSuffix(rel, ".tmpl") {
			s, err := renderString(p, string(b), data)
			if err != nil {
				return err
			}
			b, out = []byte(s), strings.TrimSuffix(rel, ".tmpl")
		}
		dst := filepath.Join(destDir, out)
		if skipExisting {
			if _, err := os.Stat(dst); err == nil {
				return nil // keep the repo's existing file
			}
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			return err
		}
		written = append(written, out)
		return nil
	})
	return written, err
}

// RenderRoot renders the scaffold's optional root/ subtree into the project root
// (e.g. an example feature into features/, so a fresh target is green on
// `specify verify`). Returns nil if the scaffold has no root/ subtree.
func RenderRoot(src fs.FS, projectRoot string, data Data) ([]string, error) {
	if _, err := fs.Stat(src, "root"); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return renderSubtree(src, "root", projectRoot, data, false)
}

// RenderGitHub renders the scaffold's optional github/ subtree (a .github/ tree
// — the CI workflow today; PR templates, defect forms, CODEOWNERS later) into
// the project root, skipping any file that already exists so a second
// `target add` never clobbers the repo's existing GitHub config. Returns nil if
// the scaffold has no github/ subtree.
func RenderGitHub(src fs.FS, projectRoot string, data Data) ([]string, error) {
	if _, err := fs.Stat(src, "github"); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return renderSubtree(src, "github", projectRoot, data, true)
}

// RenderVariant renders a variant's files subtree into destDir, OVERWRITING
// shared base files (e.g. app/router.tsx, vite.config.ts) so the chosen data
// layer / runtime wins. Returns nil if the variant has no files subtree.
func RenderVariant(src fs.FS, v Variant, destDir string, data Data) ([]string, error) {
	if v.Files == "" {
		return nil, nil
	}
	return renderSubtree(src, v.Files, destDir, data, false)
}

// RenderFeature renders a feature's files subtree into destDir.
func RenderFeature(src fs.FS, f Feature, destDir string, data Data) ([]string, error) {
	if f.Files == "" {
		return nil, nil
	}
	return renderSubtree(src, f.Files, destDir, data, false)
}

// RenderDeploy renders the deploy workflow for a kind —
// templates/deploy/<kind>/deploy.yml.tmpl — against data. It uses [[ ]] delimiters
// (not {{ }}) so a deploy workflow's many GitHub ${{ … }} expressions pass through
// verbatim while SpecKit's own [[.Dir]]/[[.Name]] vars are substituted. Returns a
// clear error for an unknown kind.
func RenderDeploy(assets fs.FS, kind string, data Data) ([]byte, error) {
	b, err := fs.ReadFile(assets, "templates/deploy/"+kind+"/deploy.yml.tmpl")
	if err != nil {
		return nil, fmt.Errorf("unknown deploy kind %q: %w", kind, err)
	}
	t, err := template.New(kind).Delims("[[", "]]").Funcs(funcs).Parse(string(b))
	if err != nil {
		return nil, fmt.Errorf("deploy/%s: %w", kind, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("deploy/%s: %w", kind, err)
	}
	return buf.Bytes(), nil
}

// RenderTarget resolves the manifest's target fields against data.
func RenderTarget(m Manifest, data Data) (RenderedTarget, error) {
	var rt RenderedTarget
	var err error
	if rt.Command, err = renderString("target.command", m.Target.Command, data); err != nil {
		return rt, err
	}
	if rt.Format, err = renderString("target.format", m.Target.Format, data); err != nil {
		return rt, err
	}
	if rt.Report, err = renderString("target.report", m.Target.Report, data); err != nil {
		return rt, err
	}
	if rt.Source, err = renderString("target.source", m.Target.Source, data); err != nil {
		return rt, err
	}
	rt.Bindings = m.Target.Bindings // a fixed enum, not a template
	return rt, nil
}

func renderString(name, text string, data Data) (string, error) {
	if text == "" {
		return "", nil
	}
	t, err := template.New(name).Funcs(funcs).Parse(text)
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return buf.String(), nil
}

// --- casing helpers ---

func tokens(s string) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	var prev rune
	for i, r := range s {
		switch {
		case r == '-' || r == '_' || r == ' ' || r == '.' || r == '/':
			flush()
		case i > 0 && unicode.IsUpper(r) && !unicode.IsUpper(prev):
			flush()
			cur.WriteRune(r)
		default:
			cur.WriteRune(r)
		}
		prev = r
	}
	flush()
	return out
}

func kebab(s string) string {
	parts := tokens(s)
	for i := range parts {
		parts[i] = strings.ToLower(parts[i])
	}
	return strings.Join(parts, "-")
}

func pascal(s string) string {
	parts := tokens(s)
	for i := range parts {
		parts[i] = title1(parts[i])
	}
	return strings.Join(parts, "")
}

func camel(s string) string {
	p := pascal(s)
	if p == "" {
		return p
	}
	r := []rune(p)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func title1(s string) string {
	if s == "" {
		return s
	}
	r := []rune(strings.ToLower(s))
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
