// Command specify is the SpecKit CLI: a present-at-runtime Go binary that is
// both the project bootstrapper and the spec-engine / verification tool (D2).
// Because the binary is present at runtime there is no bash/PowerShell script
// layer — slash-command prompts call `specify <subcommand> --json`.
//
// Arguments are parsed with the standard library flag package: each subcommand
// owns a flag.FlagSet, so flags precede any positional argument
// (e.g. `specify init --integration claude my-project`).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/markmals/speckit/internal/coreassets"
	"github.com/markmals/speckit/internal/engine"
	"github.com/markmals/speckit/internal/project"
	"github.com/markmals/speckit/internal/specmodel"
)

// version is overridden at release time via -ldflags "-X main.version=...".
var version = "0.0.0-dev"

// plannedCommands is the command surface from the fork plan (D5); these are
// stubbed until implemented.
var plannedCommands = []string{
	"gate", "ledger", "apply", "reconcile",
	"extension", "preset", "work", "bench", "issues",
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "specify:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage(os.Stdout)
		return nil
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "help", "-h", "--help":
		usage(os.Stdout)
		return nil
	case "version", "--version":
		return cmdVersion(rest)
	case "kinds":
		return cmdKinds(rest)
	case "init":
		return cmdInit(rest)
	case "scan":
		return cmdScan(rest)
	case "lock":
		return cmdLock(rest)
	case "drift":
		return cmdDrift(rest)
	case "cover":
		return cmdCover(rest)
	case "verify":
		return cmdVerify(rest)
	case "parity":
		return cmdParity(rest)
	default:
		if slices.Contains(plannedCommands, cmd) {
			return fmt.Errorf("%q is not implemented yet (planned)", cmd)
		}
		usage(os.Stderr)
		return fmt.Errorf("unknown command %q", cmd)
	}
}

// jsonFlagSet returns a FlagSet carrying the standard --json flag (D2).
func jsonFlagSet(name string) (*flag.FlagSet, *bool) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "emit output as JSON")
	return fs, jsonOut
}

func cmdVersion(args []string) error {
	fs, jsonOut := jsonFlagSet("version")
	_ = fs.Parse(args)
	if *jsonOut {
		return writeJSON(os.Stdout, map[string]string{"version": version})
	}
	fmt.Println(version)
	return nil
}

func cmdKinds(args []string) error {
	fs, jsonOut := jsonFlagSet("kinds")
	_ = fs.Parse(args)
	if *jsonOut {
		return writeJSON(os.Stdout, specmodel.Kinds)
	}
	for _, k := range specmodel.Kinds {
		fmt.Printf("%-14s %s\n", k, k.Prefix()+"…")
	}
	return nil
}

// cmdInit scaffolds a project for an agent. SPEC: story.init.basic
func cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	integration := fs.String("integration", "", "agent integration (claude|codex|copilot|generic)")
	force := fs.Bool("force", false, "proceed even if the target directory is non-empty")
	here := fs.Bool("here", false, "initialize in the current directory")
	_ = fs.Parse(args)

	root := fs.Arg(0)
	if *here || root == "." {
		root = "."
	}
	if root == "" {
		return fmt.Errorf("init: provide a project name or --here")
	}
	if *integration == "" {
		return fmt.Errorf("init: --integration required (one of %v)", project.AdapterIDs())
	}
	if root != "." {
		if err := os.MkdirAll(root, 0o755); err != nil {
			return err
		}
	}
	written, err := project.Init(root, coreassets.FS, project.Options{Integration: *integration, Force: *force})
	if err != nil {
		return err
	}
	fmt.Printf("Initialized SpecKit (%s) at %s — %d paths written\n", *integration, root, len(written))
	return nil
}

// cmdScan lints the spec library and exits non-zero if there are findings.
// SPEC: story.engine.scan
func cmdScan(args []string) error {
	fs, jsonOut := jsonFlagSet("scan")
	_ = fs.Parse(args)
	root := fs.Arg(0)
	if root == "" {
		root = "."
	}
	findings, err := engine.Scan(os.DirFS(root))
	if err != nil {
		return err
	}
	switch {
	case *jsonOut:
		if err := writeJSON(os.Stdout, findings); err != nil {
			return err
		}
	case len(findings) == 0:
		fmt.Println("scan: clean")
	default:
		for _, f := range findings {
			fmt.Printf("%s  %s  %s\n", f.Invariant, f.Path, f.Message)
		}
	}
	if len(findings) > 0 {
		os.Exit(1) // SPEC: scenario.engine.scan.* — a library with findings exits non-zero
	}
	return nil
}

// cmdLock acknowledges a spec as green on a platform. SPEC: story.engine.lock
func cmdLock(args []string) error {
	fs := flag.NewFlagSet("lock", flag.ExitOnError)
	_ = fs.Parse(args)
	platform, id := fs.Arg(0), fs.Arg(1)
	if platform == "" || id == "" {
		return fmt.Errorf("usage: specify lock <platform> <spec-id>")
	}
	if err := engine.Lock(".", platform, specmodel.SpecID(id)); err != nil {
		return err
	}
	fmt.Printf("locked %s on %s\n", id, platform)
	return nil
}

// cmdDrift reports specs whose content drifted from their locked-green hash.
// SPEC: story.engine.drift
func cmdDrift(args []string) error {
	fs, jsonOut := jsonFlagSet("drift")
	_ = fs.Parse(args)
	platform := fs.Arg(0)
	if platform == "" {
		return fmt.Errorf("usage: specify drift <platform> [path]")
	}
	root := fs.Arg(1)
	if root == "" {
		root = "."
	}
	report, err := engine.Drift(root, platform)
	if err != nil {
		return err
	}
	if *jsonOut {
		if err := writeJSON(os.Stdout, report); err != nil {
			return err
		}
	} else {
		for _, id := range report.Drifted {
			fmt.Printf("drifted  %s\n", id)
		}
		for _, id := range report.Missing {
			fmt.Printf("missing  %s\n", id)
		}
		fmt.Printf("drift(%s): %d drifted, %d missing, %d clean\n",
			platform, len(report.Drifted), len(report.Missing), len(report.Clean))
	}
	if report.HasDrift() {
		os.Exit(1) // SPEC: scenario.engine.drift.edited-spec-red — drift exits non-zero
	}
	return nil
}

// cmdCover shows a spec's per-platform coverage from the lock.
// SPEC: story.engine.cover
func cmdCover(args []string) error {
	fs, jsonOut := jsonFlagSet("cover")
	_ = fs.Parse(args)
	id := fs.Arg(0)
	if id == "" {
		return fmt.Errorf("usage: specify cover <spec-id> [path]")
	}
	root := fs.Arg(1)
	if root == "" {
		root = "."
	}
	report, err := engine.Cover(root, specmodel.SpecID(id))
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(os.Stdout, report)
	}
	if len(report.Cells) == 0 {
		fmt.Printf("cover %s: no platforms have lock state yet\n", id)
		return nil
	}
	for _, c := range report.Cells {
		fmt.Printf("%-10s %s\n", c.Platform, c.State)
	}
	return nil
}

// cmdVerify runs a platform's tests and joins them to scenarios, locking each
// spec that passed. The adapter config lives at .speckit/verify/<platform>.json.
// SPEC: story.engine.verify
func cmdVerify(args []string) error {
	fs, jsonOut := jsonFlagSet("verify")
	_ = fs.Parse(args)
	platform := fs.Arg(0)
	if platform == "" {
		return fmt.Errorf("usage: specify verify <platform> [path]")
	}
	root := fs.Arg(1)
	if root == "" {
		root = "."
	}
	cfgPath := filepath.Join(root, ".speckit", "verify", platform+".json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("no verify adapter for %q at %s", platform, cfgPath)
	}
	var cfg engine.VerifyConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("verify config %s: %w", cfgPath, err)
	}
	v, locked, err := engine.Verify(root, platform, cfg)
	if err != nil {
		return err
	}
	if *jsonOut {
		if err := writeJSON(os.Stdout, map[string]any{"result": v, "locked": locked}); err != nil {
			return err
		}
	} else {
		for _, s := range v.Failed {
			fmt.Printf("FAIL       %s\n", s)
		}
		for _, s := range v.Unjoinable {
			fmt.Printf("unjoinable %s\n", s)
		}
		for _, b := range v.Dangling {
			fmt.Printf("dangling   %s (%s)\n", b.Scenario, b.Identity)
		}
		for _, r := range v.Unbound {
			fmt.Printf("unbound    %s\n", r.Name)
		}
		fmt.Printf("verify(%s): %d passed, %d failed, %d unjoinable, %d dangling, %d unbound; %d locked\n",
			platform, len(v.Passed), len(v.Failed), len(v.Unjoinable), len(v.Dangling), len(v.Unbound), len(locked))
	}
	if !v.Green() {
		os.Exit(1) // SPEC: scenario.engine.verify.* — a non-green verify exits non-zero
	}
	return nil
}

// cmdParity prints a platform's five-state parity matrix; --gate fails unless
// every cell is conforming. SPEC: story.engine.parity
func cmdParity(args []string) error {
	fs := flag.NewFlagSet("parity", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "emit output as JSON")
	gate := fs.Bool("gate", false, "exit non-zero unless every cell is conforming")
	_ = fs.Parse(args)
	platform := fs.Arg(0)
	if platform == "" {
		return fmt.Errorf("usage: specify parity <platform> [path] [--gate]")
	}
	root := fs.Arg(1)
	if root == "" {
		root = "."
	}
	cfgPath := filepath.Join(root, ".speckit", "verify", platform+".json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("no verify adapter for %q at %s", platform, cfgPath)
	}
	var cfg engine.VerifyConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("verify config %s: %w", cfgPath, err)
	}
	report, err := engine.Parity(root, platform, cfg)
	if err != nil {
		return err
	}
	if *jsonOut {
		if err := writeJSON(os.Stdout, report); err != nil {
			return err
		}
	} else {
		for _, c := range report.Cells {
			if c.Reason != "" {
				fmt.Printf("%-18s %s (%s)\n", c.State, c.Scenario, c.Reason)
			} else {
				fmt.Printf("%-18s %s\n", c.State, c.Scenario)
			}
		}
	}
	if *gate && report.Gated() {
		os.Exit(1) // SPEC: scenario.engine.parity.suspect-lying-marker
	}
	return nil
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func usage(w io.Writer) {
	fmt.Fprintf(w, "specify %s — SpecKit CLI\n\n", version)
	fmt.Fprintln(w, "Usage: specify <command> [flags]")
	fmt.Fprintln(w, "\nCommands:")
	fmt.Fprintln(w, "  init --integration <agent> [name]   scaffold a project")
	fmt.Fprintln(w, "  scan [path]                         lint the spec library")
	fmt.Fprintln(w, "  lock <platform> <spec-id>           acknowledge a spec green on a platform")
	fmt.Fprintln(w, "  drift <platform> [path]             report specs that drifted from the lock")
	fmt.Fprintln(w, "  cover <spec-id> [path]              show a spec's per-platform coverage")
	fmt.Fprintln(w, "  verify <platform> [path]            run a platform's tests, lock what passes")
	fmt.Fprintln(w, "  parity <platform> [path] [--gate]   the five-state parity matrix")
	fmt.Fprintln(w, "  version | kinds                     (add --json for structured output)")
	fmt.Fprintln(w, "\nPlanned (stubbed):")
	fmt.Fprintln(w, "  "+strings.Join(plannedCommands, ", "))
}
