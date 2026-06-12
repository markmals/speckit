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
	"os/exec"
	"path/filepath"
	"strings"

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
		lockCmd(), driftCmd(), coverCmd(), verifyCmd(), parityCmd(), gateCmd(),
	)
	// Planned-but-unimplemented commands (D5): registered so they report intent
	// rather than "unknown command".
	for _, name := range []string{"ledger", "apply", "reconcile", "extension", "preset", "work", "bench", "issues"} {
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
			if jsonOut {
				if err := writeJSON(os.Stdout, findings); err != nil {
					return err
				}
			} else {
				fmt.Println(renderScan(findings))
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
			fmt.Println(renderLock(args[0], args[1]))
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
				fmt.Println(renderDrift(report, platform))
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
				fmt.Println(renderVerify(v, locked, platform))
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
				fmt.Println(renderParity(report))
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
	var against string
	c := &cobra.Command{
		Use:   "firewall",
		Short: "Block a scenario-tagged test change whose spec didn't change",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			changed, err := changedFiles(against)
			if err != nil {
				return err
			}
			f, err := engine.TestEditFirewall(".", changed)
			if err != nil {
				return err
			}
			return reportGate(f)
		},
	}
	c.Flags().StringVar(&against, "against", "", "diff against this ref (default: staged changes)")
	return c
}

func gateGeneratedCmd() *cobra.Command {
	var against string
	c := &cobra.Command{
		Use:   "generated",
		Short: "Block edits to generated paths",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			changed, err := changedFiles(against)
			if err != nil {
				return err
			}
			return reportGate(engine.GeneratedBlock(changed))
		},
	}
	c.Flags().StringVar(&against, "against", "", "diff against this ref (default: staged changes)")
	return c
}

func gateScopeCmd() *cobra.Command {
	var msgFile string
	c := &cobra.Command{
		Use:   "scope [subject]",
		Short: "Validate a commit subject's scope",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			return reportGate(engine.ScopedCommit(subject, scopes))
		},
	}
	c.Flags().StringVar(&msgFile, "message", "", "read the subject from a commit-message file (first line)")
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

// reportGate prints gate findings and exits non-zero if any.
func reportGate(findings []engine.GateFinding) error {
	fmt.Println(renderGate(findings))
	if len(findings) > 0 {
		os.Exit(1)
	}
	return nil
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
