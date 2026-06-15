package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/markmals/speckit/internal/config"
	"github.com/markmals/speckit/internal/coreassets"
	"github.com/markmals/speckit/internal/scaffold"
)

// deployCmd groups the optional deploy surface. No deploy is required; a target
// opts in with `specify deploy add <kind>`, which drops a .github/workflows/
// deploy.yml and records the manifest (kind + op:// secret refs). Secrets are
// wired separately with `specify secrets sync`.
func deployCmd() *cobra.Command {
	c := &cobra.Command{Use: "deploy", Short: "Optional GitHub deploy workflows (none required)"}
	c.AddCommand(deployAddCmd())
	return c
}

func deployAddCmd() *cobra.Command {
	var dir string
	var ciRefs, runtimeRefs []string
	var force bool
	c := &cobra.Command{
		Use:   "add <kind> [target]",
		Short: "Add a deploy workflow for a target (" + strings.Join(config.DeployKinds, " | ") + ")",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := args[0]
			if !slices.Contains(config.DeployKinds, kind) {
				return fmt.Errorf("unknown deploy kind %q (want one of %s)", kind, strings.Join(config.DeployKinds, ", "))
			}
			cfg, found, err := config.Load(".")
			if err != nil {
				return err
			}
			if !found {
				return fmt.Errorf("no %s — add a target first (specify target add)", config.File)
			}
			target, err := resolveTarget(cfg, args)
			if err != nil {
				return err
			}
			t := cfg.Targets[target]
			// The name renders into shell (railway --service) and unquoted YAML; the
			// dir into a YAML scalar. Reject anything that could break or inject.
			if !validTargetName(target) {
				return fmt.Errorf("target name %q is not a safe slug (alphanumeric, . _ -) — it renders into the deploy workflow", target)
			}

			ci, err := parseRefs(ciRefs)
			if err != nil {
				return fmt.Errorf("--ci: %w", err)
			}
			runtime, err := parseRefs(runtimeRefs)
			if err != nil {
				return fmt.Errorf("--runtime: %w", err)
			}

			appDir := dir
			if appDir == "" {
				appDir = filepath.Dir(t.Source) // e.g. apps/web/app -> apps/web
			}
			if !safeRenderValue(appDir) {
				return fmt.Errorf("deploy dir %q contains characters unsafe for a workflow file", appDir)
			}

			// Validate the manifest BEFORE writing anything, so an invalid manifest
			// never leaves a workflow file behind.
			d := &config.Deploy{Kind: kind, CI: ci, Runtime: runtime}
			if errs := d.Validate(target); len(errs) > 0 {
				return errs[0]
			}

			out, err := scaffold.RenderDeploy(coreassets.FS, kind, scaffold.Data{Name: target, Dir: appDir})
			if err != nil {
				return err
			}
			dst := filepath.Join(".github", "workflows", "deploy.yml")
			if _, err := os.Stat(dst); err == nil && !force {
				return fmt.Errorf("%s already exists (use --force to overwrite)", dst)
			}

			// Record the manifest first, then write the file last: a failure leaves
			// at most a recorded manifest with no file (re-running fixes it), never a
			// written workflow paired with stale config.
			if err := config.SetDeploy(".", target, d); err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(dst, out, 0o644); err != nil {
				return err
			}

			fmt.Printf("✓ deploy %q wired for target %q → %s\n", kind, target, dst)
			if len(ci)+len(runtime) > 0 {
				fmt.Printf("  next: specify secrets sync %s   (resolve %d secret reference(s) from 1Password)\n", target, len(ci)+len(runtime))
			} else {
				fmt.Printf("  add secret references with --ci NAME=op://… / --runtime NAME=op://…, then: specify secrets sync %s\n", target)
			}
			if kind == "github-pages-spa" {
				fmt.Println("  note: enable Pages (Settings → Pages → Source: GitHub Actions); no secrets needed.")
			}
			if kind == "app-store-connect" {
				fmt.Println("  note: set your signing team/identity in Project.swift and create the Apple Distribution cert + App Store Connect API key; the workflow uploads on a v* tag.")
			}
			return nil
		},
	}
	c.Flags().StringVar(&dir, "dir", "", "the app directory the workflow builds/deploys (default: the target's source parent)")
	c.Flags().StringArrayVar(&ciRefs, "ci", nil, "CI secret as NAME=op://vault/item/field (repeatable)")
	c.Flags().StringArrayVar(&runtimeRefs, "runtime", nil, "runtime secret as NAME=op://vault/item/field (repeatable)")
	c.Flags().BoolVar(&force, "force", false, "overwrite an existing deploy.yml")
	return c
}

// resolveTarget picks the target: the optional second positional arg, or the sole
// configured target when there's exactly one, else an error listing the choices.
func resolveTarget(cfg config.Config, args []string) (string, error) {
	if len(args) > 1 {
		if _, ok := cfg.Targets[args[1]]; !ok {
			return "", fmt.Errorf("target %q not in %s (have: %s)", args[1], config.File, strings.Join(targetNames(cfg), ", "))
		}
		return args[1], nil
	}
	names := targetNames(cfg)
	if len(names) == 1 {
		return names[0], nil
	}
	return "", fmt.Errorf("specify a target (have: %s)", strings.Join(names, ", "))
}

// validTargetName reports whether a target name is a safe slug:
// [A-Za-z0-9][A-Za-z0-9._-]* — no spaces, control chars, or shell/YAML
// metacharacters, since the name renders into the deploy workflow.
func validTargetName(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		alnum := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if alnum {
			continue
		}
		if i > 0 && (r == '.' || r == '_' || r == '-') {
			continue
		}
		return false
	}
	return true
}

// safeRenderValue reports whether a value (e.g. an app dir) is safe to substitute
// into a YAML scalar / shell line: no control chars and no quote/shell/YAML
// metacharacters.
func safeRenderValue(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
		switch r {
		case '"', '\'', '`', '$', '\\', ':', '\n', '\r':
			return false
		}
	}
	return true
}

// parseRefs parses NAME=op://… pairs into a map, rejecting any value that is not a
// 1Password reference — a raw secret value must never land in committed config.
func parseRefs(pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	m := map[string]string{}
	for _, p := range pairs {
		name, ref, ok := strings.Cut(p, "=")
		if !ok || name == "" {
			return nil, fmt.Errorf("expected NAME=op://… , got %q", p)
		}
		if !config.IsOpRef(ref) {
			return nil, fmt.Errorf("%s = %q is not an op:// reference (commit references, never secret values)", name, ref)
		}
		m[name] = ref
	}
	return m, nil
}
