// Command specify is the SpecKit CLI: a present-at-runtime Go binary that is
// both the project bootstrapper and the spec-engine / verification tool (D2).
// Because the binary is present at runtime there is no bash/PowerShell script
// layer — slash-command prompts call `specify <subcommand> --json`.
//
// This is the Phase-0 skeleton: it builds on all three host OSes, prints
// version, exposes the spec kinds, and stubs the command surface. Real
// behavior (Cobra/Fang wiring, the engine) lands in Phases 2–3.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/markmals/speckit/internal/coreassets"
	"github.com/markmals/speckit/internal/project"
	"github.com/markmals/speckit/internal/specmodel"
)

// version is overridden at release time via -ldflags "-X main.version=...".
var version = "0.0.0-dev"

// plannedCommands is the command surface from the fork plan (D5). Phase 0
// stubs them so the projection prompts and CI have a stable shape to target.
var plannedCommands = []string{
	"scan", "verify", "drift", "cover", "parity",
	"gate", "lock", "ledger", "apply", "reconcile",
	"extension", "preset", "work", "bench", "issues",
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "specify:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	jsonOut := false
	rest := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--json" {
			jsonOut = true
			continue
		}
		rest = append(rest, a)
	}

	cmd := ""
	if len(rest) > 0 {
		cmd = rest[0]
	}

	switch cmd {
	case "", "help", "-h", "--help":
		usage(os.Stdout)
		return nil
	case "version", "--version":
		return cmdVersion(jsonOut)
	case "kinds":
		return cmdKinds(jsonOut)
	case "init":
		return cmdInit(rest[1:])
	default:
		if slices.Contains(plannedCommands, cmd) {
			return fmt.Errorf("%q is not implemented yet (Phase 0 skeleton)", cmd)
		}
		usage(os.Stderr)
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func cmdVersion(jsonOut bool) error {
	if jsonOut {
		return writeJSON(os.Stdout, map[string]string{"version": version})
	}
	fmt.Println(version)
	return nil
}

// cmdKinds prints the closed kind taxonomy — a tiny end-to-end check that the
// binary, the shared specmodel package, and --json all wire together.
func cmdKinds(jsonOut bool) error {
	if jsonOut {
		return writeJSON(os.Stdout, specmodel.Kinds)
	}
	for _, k := range specmodel.Kinds {
		fmt.Printf("%-14s %s\n", k, k.Prefix()+"…")
	}
	return nil
}

// cmdInit scaffolds a project for an agent. SPEC: story.init.basic
func cmdInit(args []string) error {
	integration, name := "", ""
	force, here := false, false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--integration":
			i++
			if i < len(args) {
				integration = args[i]
			}
		case "--force":
			force = true
		case "--here":
			here = true
		default:
			if !strings.HasPrefix(args[i], "-") && name == "" {
				name = args[i]
			}
		}
	}
	root := name
	if here || name == "." {
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
	fmt.Printf("Initialized SpecKit (%s) at %s — %d paths written\n", integration, root, len(written))
	return nil
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func usage(w io.Writer) {
	fmt.Fprintf(w, "specify %s — SpecKit CLI (Phase 0 skeleton)\n\n", version)
	fmt.Fprintln(w, "Usage: specify <command> [--json]")
	fmt.Fprintln(w, "\nAvailable now:")
	fmt.Fprintln(w, "  version   print the binary version")
	fmt.Fprintln(w, "  kinds     list the spec kind taxonomy")
	fmt.Fprintln(w, "\nPlanned (stubbed):")
	fmt.Fprintln(w, "  "+strings.Join(plannedCommands, ", "))
}
