// Command specify is the SpecKit CLI: a present-at-runtime Go binary that is
// both the project bootstrapper and the spec-engine / verification tool (D2).
// Because the binary is present at runtime there is no bash/PowerShell script
// layer — slash-command prompts call `specify <subcommand> --json`.
//
// Built on Cobra: each subcommand owns its flags, and (via pflag) flags may be
// interspersed with positional arguments.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/markmals/speckit/internal/config"
	"github.com/markmals/speckit/internal/coreassets"
	"github.com/markmals/speckit/internal/engine"
	"github.com/markmals/speckit/internal/project"
	"github.com/markmals/speckit/internal/scaffold"
	"github.com/markmals/speckit/internal/specmodel"
)

// version is overridden at release time via -ldflags "-X main.version=...".
var version = "0.0.0-dev"

func main() {
	// Cancel in-flight work (e.g. paginating GitHub calls) on Ctrl-C / SIGTERM;
	// the offline engine commands ignore it.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := rootCmd().ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:          "specify",
		Short:        "SpecKit — project bootstrapper and spec engine",
		SilenceUsage: true,
		Version:      version,
	}
	root.SetErrPrefix("specify:")
	root.AddCommand(
		versionCmd(), kindsCmd(), initCmd(), scanCmd(), packsCmd(), targetCmd(),
		lockCmd(), driftCmd(), coverCmd(), verifyCmd(), parityCmd(), gateCmd(),
		issuesCmd(), deployCmd(), secretsCmd(), protectCmd(), workCmd(),
	)
	// Planned-but-unimplemented commands (D5): registered so they report intent
	// rather than "unknown command".
	for _, name := range []string{"ledger", "apply", "reconcile", "extension", "preset", "bench"} {
		root.AddCommand(&cobra.Command{
			Use:    name,
			Short:  "(planned — not implemented yet)",
			Hidden: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				return fmt.Errorf("%q is not implemented yet (planned)", cmd.Name())
			},
		})
	}
	return root
}

func versionCmd() *cobra.Command {
	var jsonOut bool
	c := &cobra.Command{
		Use:   "version",
		Short: "Print the binary version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOut {
				return writeJSON(os.Stdout, map[string]string{"version": version})
			}
			fmt.Println(version)
			return nil
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "emit output as JSON")
	return c
}

func kindsCmd() *cobra.Command {
	var jsonOut bool
	c := &cobra.Command{
		Use:   "kinds",
		Short: "List the spec kind taxonomy",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOut {
				return writeJSON(os.Stdout, specmodel.Kinds)
			}
			for _, k := range specmodel.Kinds {
				fmt.Printf("%-14s %s\n", k, k.Prefix()+"…")
			}
			return nil
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "emit output as JSON")
	return c
}

// SPEC: story.init.basic
func initCmd() *cobra.Command {
	var integration string
	var force, here bool
	c := &cobra.Command{
		Use:   "init [name]",
		Short: "Scaffold a SpecKit project for an agent",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := ""
			if len(args) > 0 {
				root = args[0]
			}
			if here || root == "." {
				root = "."
			}
			if root == "" {
				return fmt.Errorf("init: provide a project name or --here")
			}
			if integration == "" {
				return fmt.Errorf("init: --integration required (one of %v)", project.AdapterIDs())
			}
			if root != "." {
				if err := os.MkdirAll(root, 0o755); err != nil {
					return err
				}
			}
			written, err := project.Init(root, coreassets.FS, project.Options{Integration: integration, Force: force})
			if err != nil {
				return err
			}
			fmt.Println(renderInit(integration, root, len(written)))
			return nil
		},
	}
	c.Flags().StringVar(&integration, "integration", "", "agent integration (claude|codex|copilot|generic)")
	c.Flags().BoolVar(&force, "force", false, "proceed even if the target directory is non-empty")
	c.Flags().BoolVar(&here, "here", false, "initialize in the current directory")
	return c
}

// packsCmd projects the platform skill packs for the target stacks declared in
// .speckit/specs.json, into the configured agent's skills dir.
func packsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "packs [path]",
		Short: "Project platform skill packs for the configured target stacks",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) > 0 {
				root = args[0]
			}
			cfg, found, err := config.Load(root)
			if err != nil {
				return err
			}
			if !found {
				return fmt.Errorf("no %s — define your targets first (run specify init)", config.File)
			}
			if cfg.Agent == "" {
				return fmt.Errorf(`%s: set "agent" to project packs`, config.File)
			}
			stacks := cfg.Stacks()
			if len(stacks) == 0 {
				return fmt.Errorf(`no target declares a "stack" in %s`, config.File)
			}
			written, err := project.ProjectPacks(root, coreassets.FS, cfg.Agent, stacks)
			if err != nil {
				return err
			}
			fmt.Printf("Projected %d pack skill(s) for stacks: %s\n", len(written), strings.Join(stacks, ", "))
			return nil
		},
	}
}

// targetCmd scaffolds and registers targets.
func targetCmd() *cobra.Command {
	c := &cobra.Command{Use: "target", Short: "Scaffold and register targets"}
	c.AddCommand(targetAddCmd())
	c.AddCommand(targetRegisterCmd())
	return c
}

// targetAddCmd scaffolds a stack's starter into <dir>, registers the target in
// .speckit/specs.json, projects the stack's pack, and runs the install.
func targetAddCmd() *cobra.Command {
	var stack, dir, product, dataKind, runtimeKind string
	var with []string
	var noInstall bool
	c := &cobra.Command{
		Use:   "add <name>",
		Short: "Scaffold a target's stack and register it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if !validTargetName(name) {
				return fmt.Errorf("target add: name %q is not a safe slug (alphanumeric, . _ -); it becomes a path and renders into CI/deploy workflows", name)
			}
			if stack == "" {
				return fmt.Errorf("target add: --stack required")
			}
			sub, err := fs.Sub(coreassets.FS, "templates/scaffolds/"+stack)
			if err != nil {
				return err
			}
			m, err := scaffold.LoadManifest(sub)
			if err != nil {
				return fmt.Errorf("unknown or invalid stack %q: %w", stack, err)
			}
			// Some stacks render {{pascal .Name}} as a source identifier (Swift modules
			// / types) and require more than a safe slug — reject before any render.
			if err := m.ValidateName(name); err != nil {
				return fmt.Errorf("target add: %w", err)
			}
			// Default placement is stack-specific (the manifest's memberDir): a web
			// app lands in apps/, a go-service in cmd/, a library in packages/.
			if dir == "" {
				memberDir := m.MemberDir
				if memberDir == "" {
					memberDir = "apps"
				}
				dir = filepath.Join(memberDir, name)
			}
			features := map[string]bool{}
			for _, f := range with {
				if _, ok := m.Features[f]; !ok {
					return fmt.Errorf("stack %q has no --with feature %q", stack, f)
				}
				features[f] = true
			}
			// Resolve the runtime + data axes (--runtime/--data, else the stack
			// defaults). drizzle's D1 driver, e.g., requires the cloudflare runtime.
			runtimeVariant, resolvedRuntime, err := resolveVariant(stack, "runtime", runtimeKind, m.RuntimeDefault, m.Runtime, m.RuntimeKinds())
			if err != nil {
				return err
			}
			dataVariant, resolvedData, err := resolveVariant(stack, "data", dataKind, m.DataDefault, m.Data, m.DataKinds())
			if err != nil {
				return err
			}
			if dataVariant.RequiresRuntime != "" && dataVariant.RequiresRuntime != resolvedRuntime {
				return fmt.Errorf("--data %q requires --runtime %q (got %q)", resolvedData, dataVariant.RequiresRuntime, resolvedRuntime)
			}
			data := scaffold.Data{Name: name, Dir: dir, Product: product, Features: features, Vars: map[string]string{}}
			// Shared-module stacks resolve the repo's Go module path so members can
			// import their own generated/internal packages by full path.
			if m.SharedModule {
				data.Module = resolveModulePath(".")
			}

			written, err := scaffold.Render(sub, dir, data)
			if err != nil {
				return err
			}
			// Runtime + data layers render over the base, overwriting shared files
			// (the runtime's vite.config.ts, the data layer's router.tsx).
			for _, v := range []scaffold.Variant{runtimeVariant, dataVariant} {
				w, err := scaffold.RenderVariant(sub, v, dir, data)
				if err != nil {
					return err
				}
				written = append(written, w...)
			}
			// The data layer's runtime-specific overlay (e.g. drizzle's D1 db module
			// + wrangler binding for cloudflare vs the node:sqlite one for node).
			if overlay := dataVariant.RuntimeFiles[resolvedRuntime]; overlay != "" {
				w, err := scaffold.RenderVariant(sub, scaffold.Variant{Files: overlay}, dir, data)
				if err != nil {
					return err
				}
				written = append(written, w...)
			}
			// --with features render LAST (explicit opt-ins win) and contribute
			// their own deps + scripts to the install.
			var featureScripts []scaffold.Script
			for f := range features {
				feat := m.Features[f]
				w, err := scaffold.RenderFeature(sub, feat, dir, data)
				if err != nil {
					return err
				}
				written = append(written, w...)
				featureScripts = append(featureScripts, variantInstallScripts(scaffold.Variant{Add: feat.Add, AddDev: feat.AddDev, Scripts: feat.Scripts})...)
			}
			// Seed the scaffold's example feature into the project root, but only
			// when the spec library is empty — never clobber existing specs.
			if featuresEmpty(".") {
				w, err := scaffold.RenderRoot(sub, ".", data)
				if err != nil {
					return err
				}
				written = append(written, w...)
			}
			// Drop the project-root .github/ tree (CI workflow). Skips any file
			// the repo already has, so a second target never clobbers ci.yml.
			gh, err := scaffold.RenderGitHub(sub, ".", data)
			if err != nil {
				return err
			}
			written = append(written, gh...)

			// Shared-module stacks (go-service) compose into ONE repo-root go.mod —
			// each member a cmd/<name> sharing internal/ packages (trove's shape) —
			// instead of a self-contained module per member. Create it if the repo
			// isn't a Go module yet; a second member just joins it.
			if m.SharedModule {
				created, err := ensureRootGoMod(".", data.Module)
				if err != nil {
					return err
				}
				if created {
					written = append(written, "go.mod")
				}
			}

			rt, err := scaffold.RenderTarget(m, data)
			if err != nil {
				return err
			}
			if err := config.AddTarget(".", name, config.Target{
				Stack: stack, Product: product,
				Command: rt.Command, Format: rt.Format, Report: rt.Report, Source: config.SourcePaths{rt.Source}, Bindings: rt.Bindings,
			}); err != nil {
				return err
			}
			if err := wireMonorepo("."); err != nil {
				return fmt.Errorf("wiring mise monorepo: %w", err)
			}

			if cfg, found, _ := config.Load("."); found && cfg.Agent != "" {
				if _, err := project.ProjectPacks(".", coreassets.FS, cfg.Agent, []string{stack}); err != nil {
					fmt.Fprintf(os.Stderr, "specify: pack projection skipped: %v\n", err)
				}
			}

			// Post-render setup: run the scaffold's phased scripts in the target
			// dir. This is where versions are resolved by running the tool (e.g.
			// `pnpm add`, which pins each dependency to its current latest) rather
			// than frozen into a template. Phases run in order; a Silent step's
			// failure is logged and skipped.
			if !noInstall {
				// Base install + the runtime's and data layer's deps/codegen, all
				// phase-ordered together.
				allScripts := append([]scaffold.Script{}, m.Scripts...)
				allScripts = append(allScripts, variantInstallScripts(runtimeVariant)...)
				allScripts = append(allScripts, variantInstallScripts(dataVariant)...)
				allScripts = append(allScripts, featureScripts...)
				scripts, err := scaffold.Manifest{Scripts: allScripts}.PhasedScripts(data)
				if err != nil {
					return err
				}
				for _, s := range scripts {
					for _, command := range s.Commands {
						fmt.Printf("running: %s\n", command)
						if err := runIn(dir, command, s.Silent); err != nil {
							if s.Silent {
								fmt.Fprintf(os.Stderr, "specify: step failed (ignored): %v\n", err)
								continue
							}
							return fmt.Errorf("scaffold step %q failed: %w", command, err)
						}
					}
				}
			}

			fmt.Printf("✓ scaffolded %q (%s) → %s/ (%d files)\n  next: specify verify %s\n", name, stack, dir, len(written), name)
			return nil
		},
	}
	c.Flags().StringVar(&stack, "stack", "", "the stack to scaffold (required)")
	c.Flags().StringVar(&dir, "dir", "", "where to scaffold (default apps/<name>)")
	c.Flags().StringVar(&product, "product", "", "product label for the target")
	c.Flags().StringVar(&dataKind, "data", "", "data layer for stacks that offer it (e.g. convex|drizzle|none)")
	c.Flags().StringVar(&runtimeKind, "runtime", "", "runtime for stacks that offer it (e.g. cloudflare|node)")
	c.Flags().StringArrayVar(&with, "with", nil, "optional scaffold features (repeatable)")
	c.Flags().BoolVar(&noInstall, "no-install", false, "skip the scaffold's post-render scripts (dependency install, codegen)")
	return c
}

// targetRegisterCmd registers an EXISTING member as a target in .speckit/specs.json
// without scaffolding or installing anything — the onboarding path for adopting
// SpecKit in a repo whose code already exists (converting a Workbench-shaped repo
// like trove). It seeds the target's test wiring from the stack's scaffold manifest
// when one exists (web, go-service); for stacks without a scaffold (ts-lib, go-cli)
// — or to match a member wired differently than the scaffold — pass the fields as
// flags. Unlike `target add`, it writes no files and runs no scripts.
func targetRegisterCmd() *cobra.Command {
	var stack, dir, product, format, command, report, bindings string
	var source []string
	c := &cobra.Command{
		Use:   "register <name> [flags]",
		Short: "Register an existing member as a target (no scaffolding)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return registerTarget(".", regOpts{
				name: args[0], stack: stack, dir: dir, product: product,
				format: format, command: command, report: report, source: source, bindings: bindings,
			})
		},
	}
	c.Flags().StringVar(&stack, "stack", "", "the member's stack (selects its pack + seeds wiring from its scaffold, if any)")
	c.Flags().StringVar(&dir, "dir", "", "the existing member's path (default: <memberDir>/<name> from the stack)")
	c.Flags().StringVar(&product, "product", "", "product label for the target")
	c.Flags().StringVar(&format, "format", "", "report format (junit|swift|gotest) — overrides the stack default")
	c.Flags().StringVar(&command, "command", "", "test command that produces the report — overrides the stack default")
	c.Flags().StringVar(&report, "report", "", "report path the engine joins (root-relative) — overrides the stack default")
	c.Flags().StringArrayVar(&source, "source", nil, "source dir scanned for bindings (repeatable for multi-source targets) — overrides the stack default")
	c.Flags().StringVar(&bindings, "bindings", "", "binding mode (strict|scoped) — overrides the stack default")
	return c
}

// regOpts are the inputs to registerTarget (the flags of `target register`).
type regOpts struct {
	name, stack, dir, product, format, command, report, bindings string
	source                                                       []string
}

// registerTarget records an existing member as a target under root's
// .speckit/specs.json. Stack-manifest defaults fill the wiring; non-empty opts
// override them. It validates the member dir exists and the assembled wiring is
// complete + well-formed before writing.
func registerTarget(root string, o regOpts) error {
	if !validTargetName(o.name) {
		return fmt.Errorf("target register: name %q is not a safe slug (alphanumeric, . _ -)", o.name)
	}

	// Seed wiring from the stack's scaffold manifest when it has one (web,
	// go-service). Stacks without a scaffold (ts-lib, go-cli) leave rt empty — the
	// fields then come from flags.
	var rt scaffold.RenderedTarget
	if o.stack != "" {
		if sub, err := fs.Sub(coreassets.FS, "templates/scaffolds/"+o.stack); err == nil {
			if m, err := scaffold.LoadManifest(sub); err == nil {
				if o.dir == "" {
					memberDir := m.MemberDir
					if memberDir == "" {
						memberDir = "apps"
					}
					o.dir = filepath.Join(memberDir, o.name)
				}
				rt, _ = scaffold.RenderTarget(m, scaffold.Data{Name: o.name, Dir: o.dir, Product: o.product})
			}
		}
	}
	if o.dir == "" {
		return fmt.Errorf("target register: --dir is required (the existing member's path) for a stack without a scaffold")
	}
	if fi, err := os.Stat(filepath.Join(root, o.dir)); err != nil || !fi.IsDir() {
		return fmt.Errorf("target register: member dir %q does not exist under the project — register is for existing members; use `target add` to scaffold a new one", o.dir)
	}

	// Flag overrides win over the manifest defaults.
	t := config.Target{
		Stack:    o.stack,
		Product:  o.product,
		Command:  firstNonEmpty(o.command, rt.Command),
		Format:   firstNonEmpty(o.format, rt.Format),
		Report:   firstNonEmpty(o.report, rt.Report),
		Source:   resolveSources(o.source, rt.Source),
		Bindings: firstNonEmpty(o.bindings, rt.Bindings),
	}
	if t.Format == "" || t.Report == "" || len(t.Source) == 0 {
		return fmt.Errorf("target register: incomplete wiring — provide --format, --report, and --source (stack %q has no scaffold to derive them from)", o.stack)
	}
	if t.Format != "junit" && t.Format != "swift" && t.Format != "gotest" {
		return fmt.Errorf("target register: unknown --format %q (want junit|swift|gotest)", t.Format)
	}
	if t.Bindings != "" && t.Bindings != "strict" && t.Bindings != "scoped" {
		return fmt.Errorf("target register: unknown --bindings %q (want strict|scoped)", t.Bindings)
	}

	if err := config.AddTarget(root, o.name, t); err != nil {
		return err
	}
	if err := wireMonorepo(root); err != nil {
		return fmt.Errorf("wiring mise monorepo: %w", err)
	}
	// Project the stack's pack (agent skills) when an agent is configured — same as
	// `target add`. Skipped on a config that has no agent yet (e.g. a fresh
	// register before `init`).
	if o.stack != "" {
		if cfg, found, _ := config.Load(root); found && cfg.Agent != "" {
			if _, err := project.ProjectPacks(root, coreassets.FS, cfg.Agent, []string{o.stack}); err != nil {
				fmt.Fprintf(os.Stderr, "specify: pack projection skipped: %v\n", err)
			}
		}
	}
	fmt.Printf("✓ registered target %q (%s) → %s\n  next: specify verify %s\n", o.name, t.Format, o.dir, o.name)
	return nil
}

// firstNonEmpty returns a if it's non-empty, else b.
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// resolveSources picks the explicit --source paths when any were given, else the
// stack manifest's single derived source (dropped when empty).
func resolveSources(flags []string, fallback string) config.SourcePaths {
	if len(flags) > 0 {
		return config.SourcePaths(flags)
	}
	if fallback == "" {
		return nil
	}
	return config.SourcePaths{fallback}
}

// resolveVariant picks an axis option (runtime/data): the flag value, else the
// manifest default. Errors if the stack offers options but the chosen kind is
// unknown, or if a kind is given for a stack with no such axis.
func resolveVariant(stack, axis, flag, def string, variants map[string]scaffold.Variant, kinds []string) (scaffold.Variant, string, error) {
	if len(variants) == 0 {
		if flag != "" {
			return scaffold.Variant{}, "", fmt.Errorf("stack %q has no %s layers (--%s not supported)", stack, axis, axis)
		}
		return scaffold.Variant{}, "", nil
	}
	kind := flag
	if kind == "" {
		kind = def
	}
	v, ok := variants[kind]
	if !ok {
		return scaffold.Variant{}, "", fmt.Errorf("stack %q has no --%s %q (have: %s)", stack, axis, kind, strings.Join(kinds, ", "))
	}
	return v, kind, nil
}

// variantInstallScripts turns a variant's declared deps into pnpm-add steps
// (phase 2, after the base install) plus the variant's own scripts (e.g. codegen).
func variantInstallScripts(v scaffold.Variant) []scaffold.Script {
	var out []scaffold.Script
	if len(v.Add) > 0 {
		out = append(out, scaffold.Script{Phase: 2, Commands: []string{"pnpm add " + strings.Join(v.Add, " ")}})
	}
	if len(v.AddDev) > 0 {
		out = append(out, scaffold.Script{Phase: 2, Commands: []string{"pnpm add -D " + strings.Join(v.AddDev, " ")}})
	}
	return append(out, v.Scripts...)
}

// featuresEmpty reports whether the project has no spec library yet (so a
// scaffold may seed an example feature without clobbering anything).
func featuresEmpty(root string) bool {
	es, err := os.ReadDir(filepath.Join(root, "features"))
	if err != nil {
		return true // absent
	}
	return len(es) == 0
}

// goModVersion is the Go directive seeded into a freshly created shared root
// go.mod; the scaffold's phase-0 `go mod tidy` normalizes it afterward. Kept in
// lockstep with the go-service mise.toml pin.
const goModVersion = "1.26"

// ensureRootGoMod makes the project a single Go module so shared-module members
// (go-service) compose into ONE root go.mod — each a cmd/<name> sharing internal/
// packages — instead of a self-contained module per member. No-op (returns false)
// when a go.mod already exists: a prior member, or a hand-authored module like a
// converted trove. modulePath is the resolved path to write (so it matches the
// member templates' imports). Reports whether it created the file.
func ensureRootGoMod(projectRoot, modulePath string) (bool, error) {
	gomod := filepath.Join(projectRoot, "go.mod")
	if _, err := os.Stat(gomod); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	content := fmt.Sprintf("module %s\n\ngo %s\n", modulePath, goModVersion)
	if err := os.WriteFile(gomod, []byte(content), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// resolveModulePath is the module path a shared-module member should import its
// own packages under: the existing repo-root go.mod's module line if the repo is
// already a module (a prior member, or a converted trove), else the path
// deriveModulePath would create. Both paths agree, so member imports match the
// go.mod ensureRootGoMod writes.
func resolveModulePath(projectRoot string) string {
	if b, err := os.ReadFile(filepath.Join(projectRoot, "go.mod")); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
				return strings.TrimSpace(rest)
			}
		}
	}
	return deriveModulePath(projectRoot)
}

// deriveModulePath picks a Go module path for a new root go.mod: the repo's
// origin remote mapped to host/owner/repo (e.g. github.com/markmals/trove), else
// the project directory's base name. Offline — reads only local git config.
func deriveModulePath(projectRoot string) string {
	out, err := exec.Command("git", "-C", projectRoot, "remote", "get-url", "origin").Output()
	if err == nil {
		if mp := moduleFromRemote(strings.TrimSpace(string(out))); mp != "" {
			return mp
		}
	}
	if abs, err := filepath.Abs(projectRoot); err == nil {
		return filepath.Base(abs)
	}
	return filepath.Base(projectRoot)
}

// moduleFromRemote maps a git remote URL to a Go module path (host/owner/repo),
// or "" if it can't. Handles https://, ssh://, and scp-style git@host:owner/repo.
func moduleFromRemote(url string) string {
	url = strings.TrimSuffix(url, ".git")
	if url == "" {
		return ""
	}
	// scp-style: git@github.com:owner/repo
	if !strings.Contains(url, "://") && strings.Contains(url, "@") && strings.Contains(url, ":") {
		url = url[strings.Index(url, "@")+1:]
		return strings.Replace(url, ":", "/", 1)
	}
	// scheme://[user@]host/owner/repo
	if i := strings.Index(url, "://"); i != -1 {
		url = url[i+3:]
		if at := strings.Index(url, "@"); at != -1 {
			if slash := strings.Index(url, "/"); slash == -1 || at < slash {
				url = url[at+1:]
			}
		}
		return url
	}
	return ""
}

// runIn runs one scaffold script command (a developer/SpecKit-controlled shell
// string from the embedded manifest — same trust boundary as a verify command)
// in dir. A silent step's output is discarded; otherwise it streams.
func runIn(dir, command string, silent bool) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Dir = dir
	if !silent {
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	}
	return cmd.Run()
}

// SPEC: story.engine.scan
func scanCmd() *cobra.Command {
	var jsonOut bool
	c := &cobra.Command{
		Use:   "scan [path]",
		Short: "Lint the spec library (I1–I6)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) > 0 {
				root = args[0]
			}
			findings, err := engine.Scan(os.DirFS(root))
			if err != nil {
				return err
			}
			// Validate .speckit/specs.json too, when present.
			var configErrs []string
			if cfg, found, err := config.Load(root); err != nil {
				configErrs = append(configErrs, err.Error())
			} else if found {
				for _, e := range cfg.Validate() {
					configErrs = append(configErrs, e.Error())
				}
			}
			if jsonOut {
				if err := writeJSON(os.Stdout, map[string]any{"library": findings, "config": configErrs}); err != nil {
					return err
				}
			} else {
				fmt.Println(renderScan(findings))
				for _, e := range configErrs {
					fmt.Printf("  ✗ %s: %s\n", config.File, e)
				}
			}
			if len(findings) > 0 || len(configErrs) > 0 {
				os.Exit(1) // SPEC: scenario.engine.scan.* — findings or config errors exit non-zero
			}
			return nil
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "emit findings as JSON")
	return c
}

// SPEC: story.engine.lock
func lockCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lock <target> <spec-id>",
		Short: "Acknowledge a spec green on a target",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := engine.Lock(".", args[0], specmodel.SpecID(args[1])); err != nil {
				return err
			}
			fmt.Println(renderLock(args[0], args[1]))
			return nil
		},
	}
}

// SPEC: story.engine.drift
func driftCmd() *cobra.Command {
	var jsonOut bool
	c := &cobra.Command{
		Use:   "drift <target> [path]",
		Short: "Report specs that drifted from the lock",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, root := args[0], "."
			if len(args) > 1 {
				root = args[1]
			}
			report, err := engine.Drift(root, target)
			if err != nil {
				return err
			}
			if jsonOut {
				if err := writeJSON(os.Stdout, report); err != nil {
					return err
				}
			} else {
				fmt.Println(renderDrift(report, target))
			}
			if report.HasDrift() {
				os.Exit(1) // SPEC: scenario.engine.drift.edited-spec-red
			}
			return nil
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "emit output as JSON")
	return c
}

// SPEC: story.engine.cover
func coverCmd() *cobra.Command {
	var jsonOut bool
	c := &cobra.Command{
		Use:   "cover <spec-id> [path]",
		Short: "Show a spec's per-target coverage",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, root := args[0], "."
			if len(args) > 1 {
				root = args[1]
			}
			report, err := engine.Cover(root, specmodel.SpecID(id))
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(os.Stdout, report)
			}
			fmt.Println(renderCover(report))
			return nil
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "emit output as JSON")
	return c
}

// SPEC: story.engine.verify
func verifyCmd() *cobra.Command {
	var jsonOut bool
	var format string
	c := &cobra.Command{
		Use:   "verify <target> [path]",
		Short: "Run a target's tests and lock what passes",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmtv, err := resolveFormat(format, jsonOut)
			if err != nil {
				return err
			}
			target, root := args[0], "."
			if len(args) > 1 {
				root = args[1]
			}
			cfg, err := verifyConfigFor(root, target)
			if err != nil {
				return err
			}
			v, locked, err := engine.Verify(root, target, cfg)
			if err != nil {
				return err
			}
			switch fmtv {
			case formatJSON:
				if err := writeJSON(os.Stdout, map[string]any{"result": v, "locked": locked}); err != nil {
					return err
				}
			case formatGitHub:
				locs, err := engine.SpecLocations(root)
				if err != nil {
					return err
				}
				for _, line := range verifyAnnotations(v, locs) {
					fmt.Println(line)
				}
			default:
				fmt.Println(renderVerify(v, locked, target))
			}
			if !v.Green() {
				os.Exit(1) // SPEC: scenario.engine.verify.* — a non-green verify exits non-zero
			}
			return nil
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "emit output as JSON (alias for --format json)")
	c.Flags().StringVar(&format, "format", "text", "output format: text|json|github (github emits CI annotations)")
	return c
}

// SPEC: story.engine.parity
func parityCmd() *cobra.Command {
	var jsonOut, gate bool
	var format string
	c := &cobra.Command{
		Use:   "parity <target> [path]",
		Short: "The five-state parity matrix",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmtv, err := resolveFormat(format, jsonOut)
			if err != nil {
				return err
			}
			target, root := args[0], "."
			if len(args) > 1 {
				root = args[1]
			}
			cfg, err := verifyConfigFor(root, target)
			if err != nil {
				return err
			}
			report, err := engine.Parity(root, target, cfg)
			if err != nil {
				return err
			}
			switch fmtv {
			case formatJSON:
				if err := writeJSON(os.Stdout, report); err != nil {
					return err
				}
			case formatGitHub:
				locs, err := engine.SpecLocations(root)
				if err != nil {
					return err
				}
				for _, line := range parityAnnotations(report, locs) {
					fmt.Println(line)
				}
			default:
				fmt.Println(renderParity(report))
			}
			if gate && report.Gated() {
				os.Exit(1) // SPEC: scenario.engine.parity.suspect-lying-marker
			}
			return nil
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "emit output as JSON (alias for --format json)")
	c.Flags().BoolVar(&gate, "gate", false, "exit non-zero unless every cell is conforming")
	c.Flags().StringVar(&format, "format", "text", "output format: text|json|github (github emits CI annotations)")
	return c
}

// SPEC: story.engine.gate
func gateCmd() *cobra.Command {
	g := &cobra.Command{
		Use:   "gate",
		Short: "Enforcement subchecks for git/CI (D8)",
	}
	g.AddCommand(gateFirewallCmd(), gateGeneratedCmd(), gateScopeCmd())
	return g
}

func gateFirewallCmd() *cobra.Command {
	var against, format string
	c := &cobra.Command{
		Use:   "firewall",
		Short: "Block a scenario-tagged test change whose spec didn't change",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmtv, err := parseFormat(format)
			if err != nil {
				return err
			}
			changed, err := changedFiles(against)
			if err != nil {
				return err
			}
			f, err := engine.TestEditFirewall(".", changed)
			if err != nil {
				return err
			}
			return reportGate(f, fmtv)
		},
	}
	c.Flags().StringVar(&against, "against", "", "diff against this ref (default: staged changes)")
	c.Flags().StringVar(&format, "format", "text", "output format: text|json|github (github emits CI annotations)")
	return c
}

func gateGeneratedCmd() *cobra.Command {
	var against, format string
	c := &cobra.Command{
		Use:   "generated",
		Short: "Block edits to generated paths",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmtv, err := parseFormat(format)
			if err != nil {
				return err
			}
			changed, err := changedFiles(against)
			if err != nil {
				return err
			}
			return reportGate(engine.GeneratedBlock(changed), fmtv)
		},
	}
	c.Flags().StringVar(&against, "against", "", "diff against this ref (default: staged changes)")
	c.Flags().StringVar(&format, "format", "text", "output format: text|json|github (github emits CI annotations)")
	return c
}

func gateScopeCmd() *cobra.Command {
	var msgFile, format string
	c := &cobra.Command{
		Use:   "scope [subject]",
		Short: "Validate a commit subject's scope",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmtv, err := parseFormat(format)
			if err != nil {
				return err
			}
			var subject string
			switch {
			case msgFile != "":
				data, err := os.ReadFile(msgFile)
				if err != nil {
					return err
				}
				subject = strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)[0]
			case len(args) > 0:
				subject = args[0]
			default:
				return fmt.Errorf("provide a subject or --message <file>")
			}
			scopes, err := engine.DefinedScopes(".")
			if err != nil {
				return err
			}
			return reportGate(engine.ScopedCommit(subject, scopes), fmtv)
		},
	}
	c.Flags().StringVar(&msgFile, "message", "", "read the subject from a commit-message file (first line)")
	c.Flags().StringVar(&format, "format", "text", "output format: text|json|github (github emits CI annotations)")
	return c
}

// changedFiles lists repo-relative paths changed in the staged set (default) or
// against a ref.
func changedFiles(against string) ([]string, error) {
	var c *exec.Cmd
	if against == "" {
		c = exec.Command("git", "diff", "--cached", "--name-only")
	} else {
		c = exec.Command("git", "diff", "--name-only", against)
	}
	out, err := c.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// reportGate renders gate findings in the requested format and exits non-zero
// if any. github format emits one workflow-command annotation per finding (no
// trailing summary), so the CI step both fails and annotates the offending file.
func reportGate(findings []engine.GateFinding, format outputFormat) error {
	switch format {
	case formatJSON:
		if findings == nil {
			findings = []engine.GateFinding{}
		}
		if err := writeJSON(os.Stdout, map[string]any{"findings": findings}); err != nil {
			return err
		}
	case formatGitHub:
		for _, f := range findings {
			fmt.Println(ghCommand("error", f.Path, f.Line, f.Message))
		}
	default:
		fmt.Println(renderGate(findings))
	}
	if len(findings) > 0 {
		os.Exit(1)
	}
	return nil
}

// verifyConfigFor resolves a target's verify wiring from .speckit/specs.json.
func verifyConfigFor(root, target string) (engine.VerifyConfig, error) {
	cfg, found, err := config.Load(root)
	if err != nil {
		return engine.VerifyConfig{}, err
	}
	if !found {
		return engine.VerifyConfig{}, fmt.Errorf("no %s — define your targets first (run specify init)", config.File)
	}
	t, ok := cfg.Targets[target]
	if !ok {
		return engine.VerifyConfig{}, fmt.Errorf("target %q not in %s (have: %s)", target, config.File, strings.Join(targetNames(cfg), ", "))
	}
	return engine.VerifyConfig{Command: t.Command, Format: t.Format, Report: t.Report, Source: []string(t.Source), Bindings: t.Bindings}, nil
}

// targetNames lists the configured target names, sorted.
func targetNames(cfg config.Config) []string {
	names := make([]string, 0, len(cfg.Targets))
	for n := range cfg.Targets {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
