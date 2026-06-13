package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/markmals/speckit/internal/config"
)

// secretDest is where a resolved secret value is pushed.
type secretDest int

const (
	destGitHubActions   secretDest = iota // gh secret set — CI credentials
	destPlatformRuntime                   // the platform's own store — app runtime secrets
)

func (d secretDest) String() string {
	if d == destGitHubActions {
		return "github-actions"
	}
	return "platform-runtime"
}

// secretOp is one resolve-and-push step: read Ref via op, push to Env at Dest. The
// op:// value itself is NEVER part of this struct — only the reference.
type secretOp struct {
	Env  string
	Ref  string
	Dest secretDest
}

// syncPlan builds the ordered secret ops for a deploy manifest — every ci entry to
// GitHub Actions, every runtime entry to the platform store. Pure (no op/gh calls)
// so it's unit-testable and drives --dry-run.
func syncPlan(d config.Deploy) []secretOp {
	var ops []secretOp
	for _, env := range sortedKeys(d.CI) {
		ops = append(ops, secretOp{Env: env, Ref: d.CI[env], Dest: destGitHubActions})
	}
	for _, env := range sortedKeys(d.Runtime) {
		ops = append(ops, secretOp{Env: env, Ref: d.Runtime[env], Dest: destPlatformRuntime})
	}
	return ops
}

func sortedKeys(m map[string]string) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func secretsCmd() *cobra.Command {
	c := &cobra.Command{Use: "secrets", Short: "Resolve 1Password (op://) references into GitHub + platform secrets"}
	c.AddCommand(secretsSyncCmd())
	return c
}

func secretsSyncCmd() *cobra.Command {
	var dir string
	var dryRun, yes bool
	c := &cobra.Command{
		Use:   "sync [target]",
		Short: "Push a target's op:// secret references to GitHub Actions + the platform store",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, found, err := config.Load(".")
			if err != nil {
				return err
			}
			if !found {
				return fmt.Errorf("no %s — add a target first (specify target add)", config.File)
			}
			target, err := resolveTarget(cfg, append([]string{"sync"}, args...))
			if err != nil {
				return err
			}
			t := cfg.Targets[target]
			if t.Deploy == nil {
				return fmt.Errorf("target %q has no deploy manifest — run `specify deploy add <kind> %s` first", target, target)
			}
			plan := syncPlan(*t.Deploy)
			if len(plan) == 0 {
				fmt.Printf("No secret references for target %q (deploy kind %q).\n", target, t.Deploy.Kind)
				return nil
			}

			// --dry-run prints the plan (env -> ref, destination) and resolves nothing.
			if dryRun {
				fmt.Printf("Would sync %d secret(s) for %q (%s):\n", len(plan), target, t.Deploy.Kind)
				for _, op := range plan {
					fmt.Printf("  %-22s %-16s ← %s\n", op.Env, "("+op.Dest.String()+")", op.Ref)
				}
				return nil
			}

			repo, err := resolveGitHubRepoForSecrets(plan, repoFlag(cmd))
			if err != nil {
				return err
			}
			ok, err := confirmAction(os.Stdin, os.Stdout,
				fmt.Sprintf("Resolve %d 1Password secret(s) and push them to %s / the %s store?", len(plan), repo, t.Deploy.Kind), yes)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("aborted")
			}

			appDir := dir
			if appDir == "" {
				appDir = filepath.Dir(t.Source)
			}
			for _, op := range plan {
				if err := pushSecret(op, t.Deploy.Kind, repo, appDir); err != nil {
					return fmt.Errorf("%s: %w", op.Env, err)
				}
				fmt.Printf("  ✓ %s → %s\n", op.Env, op.Dest)
			}
			fmt.Printf("✓ synced %d secret(s) for %q\n", len(plan), target)
			return nil
		},
	}
	addRepoFlag(c.Flags())
	c.Flags().StringVar(&dir, "dir", "", "app directory for the platform CLI (default: the target's source parent)")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print the plan (env → reference) without resolving any value")
	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	return c
}

// resolveGitHubRepoForSecrets resolves the repo only when a CI (GitHub Actions)
// secret is in the plan — a runtime-only sync needs no GitHub context.
func resolveGitHubRepoForSecrets(plan []secretOp, repoOverride string) (string, error) {
	for _, op := range plan {
		if op.Dest == destGitHubActions {
			repo, err := resolveGitHubRepo(repoOverride)
			if err != nil {
				return "", err
			}
			return repo, nil
		}
	}
	return "", nil
}

// pushSecret resolves op://ref and pushes the value to its destination. The value
// is read from `op`, fed via stdin (never argv, never logged) where the tool
// supports it, and discarded immediately after. Railway is the exception: its CLI
// only takes KEY=VALUE on argv (a known limitation, surfaced to the user).
func pushSecret(op secretOp, kind, repo, appDir string) error {
	switch op.Dest {
	case destGitHubActions:
		return feedSecretStdin(op.Ref, exec.Command("gh", "secret", "set", op.Env, "--repo", repo))
	case destPlatformRuntime:
		switch {
		case strings.HasPrefix(kind, "cloudflare-"):
			c := exec.Command("wrangler", "secret", "put", op.Env)
			c.Dir = appDir
			return feedSecretStdin(op.Ref, c)
		case kind == "railway":
			fmt.Fprintf(os.Stderr, "specify: note — Railway sets runtime vars via the command line (argv); %s is briefly visible to local process listing.\n", op.Env)
			val, err := opRead(op.Ref)
			if err != nil {
				return err
			}
			c := exec.Command("railway", "variables", "--set", op.Env+"="+val)
			c.Dir = appDir
			return runRedacting(c, val)
		case kind == "github-pages-spa":
			return fmt.Errorf("github-pages-spa has no runtime secret store (it uses GITHUB_TOKEN) — move %q to the ci section or drop it", op.Env)
		default:
			return fmt.Errorf("no runtime secret store known for deploy kind %q", kind)
		}
	}
	return fmt.Errorf("unknown secret destination")
}

// feedSecretStdin reads op://ref and feeds the value to dst on stdin (so the value
// never appears in argv or logs), then runs dst with the value redacted from any
// error output.
func feedSecretStdin(ref string, dst *exec.Cmd) error {
	val, err := opRead(ref)
	if err != nil {
		return err
	}
	dst.Stdin = strings.NewReader(val)
	return runRedacting(dst, val)
}

// opRead returns the value at a 1Password reference, trimming the trailing newline
// `op` adds (matching `$(op read …)` semantics). The value is never logged.
func opRead(ref string) (string, error) {
	out, err := exec.Command("op", "read", ref).Output()
	if err != nil {
		return "", fmt.Errorf("op read %s: %w", ref, opStderr(err))
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

// runRedacting runs c, discarding stdout (which could echo a value) and, on
// failure, returning the child's stderr with `secret` scrubbed out — a defense
// against a tool (gh/wrangler/railway) that echoes the rejected value it received.
func runRedacting(c *exec.Cmd, secret string) error {
	var stderr bytes.Buffer
	c.Stderr = &stderr
	c.Stdout = nil
	if err := c.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if secret != "" && msg != "" {
			msg = strings.ReplaceAll(msg, secret, "[redacted]")
		}
		if msg != "" {
			return fmt.Errorf("%s: %s", filepath.Base(c.Path), msg)
		}
		return err
	}
	return nil
}

// opStderr enriches an exec error with the tool's stderr when available.
func opStderr(err error) error {
	if ee, ok := err.(*exec.ExitError); ok {
		if msg := strings.TrimSpace(string(ee.Stderr)); msg != "" {
			return fmt.Errorf("%s", msg)
		}
	}
	return err
}
