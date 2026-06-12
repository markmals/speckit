// Command specify is the SpecKit CLI: a present-at-runtime Go binary that is
// both the project bootstrapper and the spec-engine / verification tool (D2).
// Because the binary is present at runtime there is no bash/PowerShell script
// layer — slash-command prompts call `specify <subcommand> --json`.
//
// Built on Cobra: each subcommand owns its flags, and (via pflag) flags may be
// interspersed with positional arguments.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/markmals/speckit/internal/coreassets"
	"github.com/markmals/speckit/internal/engine"
	"github.com/markmals/speckit/internal/project"
	"github.com/markmals/speckit/internal/specmodel"
)

// version is overridden at release time via -ldflags "-X main.version=...".
var version = "0.0.0-dev"

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "specify",
		Short:         "SpecKit — project bootstrapper and spec engine",
		SilenceUsage:  true,
		Version:       version,
	}
	root.SetErrPrefix("specify:")
	root.AddCommand(
		versionCmd(), kindsCmd(), initCmd(), scanCmd(),
		lockCmd(), driftCmd(), coverCmd(), verifyCmd(), parityCmd(),
	)
	// Planned-but-unimplemented commands (D5): registered so they report intent
	// rather than "unknown command".
	for _, name := range []string{"gate", "ledger", "apply", "reconcile", "extension", "preset", "work", "bench", "issues"} {
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
			fmt.Printf("Initialized SpecKit (%s) at %s — %d paths written\n", integration, root, len(written))
			return nil
		},
	}
	c.Flags().StringVar(&integration, "integration", "", "agent integration (claude|codex|copilot|generic)")
	c.Flags().BoolVar(&force, "force", false, "proceed even if the target directory is non-empty")
	c.Flags().BoolVar(&here, "here", false, "initialize in the current directory")
	return c
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
			switch {
			case jsonOut:
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
				os.Exit(1) // SPEC: scenario.engine.scan.* — findings exit non-zero
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
		Use:   "lock <platform> <spec-id>",
		Short: "Acknowledge a spec green on a platform",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := engine.Lock(".", args[0], specmodel.SpecID(args[1])); err != nil {
				return err
			}
			fmt.Printf("locked %s on %s\n", args[1], args[0])
			return nil
		},
	}
}

// SPEC: story.engine.drift
func driftCmd() *cobra.Command {
	var jsonOut bool
	c := &cobra.Command{
		Use:   "drift <platform> [path]",
		Short: "Report specs that drifted from the lock",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			platform, root := args[0], "."
			if len(args) > 1 {
				root = args[1]
			}
			report, err := engine.Drift(root, platform)
			if err != nil {
				return err
			}
			if jsonOut {
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
		Short: "Show a spec's per-platform coverage",
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
			if len(report.Cells) == 0 {
				fmt.Printf("cover %s: no platforms have lock state yet\n", id)
				return nil
			}
			for _, cell := range report.Cells {
				fmt.Printf("%-10s %s\n", cell.Platform, cell.State)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "emit output as JSON")
	return c
}

// SPEC: story.engine.verify
func verifyCmd() *cobra.Command {
	var jsonOut bool
	c := &cobra.Command{
		Use:   "verify <platform> [path]",
		Short: "Run a platform's tests and lock what passes",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			platform, root := args[0], "."
			if len(args) > 1 {
				root = args[1]
			}
			cfg, err := loadVerifyConfig(root, platform)
			if err != nil {
				return err
			}
			v, locked, err := engine.Verify(root, platform, cfg)
			if err != nil {
				return err
			}
			if jsonOut {
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
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "emit output as JSON")
	return c
}

// SPEC: story.engine.parity
func parityCmd() *cobra.Command {
	var jsonOut, gate bool
	c := &cobra.Command{
		Use:   "parity <platform> [path]",
		Short: "The five-state parity matrix",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			platform, root := args[0], "."
			if len(args) > 1 {
				root = args[1]
			}
			cfg, err := loadVerifyConfig(root, platform)
			if err != nil {
				return err
			}
			report, err := engine.Parity(root, platform, cfg)
			if err != nil {
				return err
			}
			if jsonOut {
				if err := writeJSON(os.Stdout, report); err != nil {
					return err
				}
			} else {
				for _, cell := range report.Cells {
					if cell.Reason != "" {
						fmt.Printf("%-18s %s (%s)\n", cell.State, cell.Scenario, cell.Reason)
					} else {
						fmt.Printf("%-18s %s\n", cell.State, cell.Scenario)
					}
				}
			}
			if gate && report.Gated() {
				os.Exit(1) // SPEC: scenario.engine.parity.suspect-lying-marker
			}
			return nil
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "emit output as JSON")
	c.Flags().BoolVar(&gate, "gate", false, "exit non-zero unless every cell is conforming")
	return c
}

// loadVerifyConfig reads a platform's verify adapter config (.speckit/verify/<platform>.json).
func loadVerifyConfig(root, platform string) (engine.VerifyConfig, error) {
	cfgPath := filepath.Join(root, ".speckit", "verify", platform+".json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return engine.VerifyConfig{}, fmt.Errorf("no verify adapter for %q at %s", platform, cfgPath)
	}
	var cfg engine.VerifyConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return engine.VerifyConfig{}, fmt.Errorf("verify config %s: %w", cfgPath, err)
	}
	return cfg, nil
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
