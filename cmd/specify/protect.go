package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/markmals/speckit/internal/github"
)

// protectCmd provisions the branch-protection ruleset that makes the SpecKit gate
// bite: require the CI contexts, require a PR, block force-pushes on the default
// branch. It codifies the docs/ci-gating.md recipe using gh's inherited token.
// Re-runnable: it updates an existing same-named ruleset in place.
func protectCmd() *cobra.Command {
	var name string
	var reviews int
	var contexts []string
	var yes, jsonOut bool
	c := &cobra.Command{
		Use:   "protect",
		Short: "Provision the branch-protection ruleset (require the gate, require a PR, block force-push)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, repo, err := resolveGitHub()
			if err != nil {
				return err
			}
			show := contexts
			if len(show) == 0 {
				show = github.DefaultGateContexts
			}
			ok, err := confirmAction(os.Stdin, os.Stdout,
				fmt.Sprintf("Provision ruleset %q on %s's default branch — require [%s], require a PR (%d approval(s)), block force-push?",
					name, repo, strings.Join(show, ", "), reviews), yes)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("aborted")
			}
			id, updated, err := client.ProvisionGateRuleset(cmd.Context(), repo, github.GateRulesetOptions{
				Name: name, Contexts: contexts, RequiredReviews: reviews,
			})
			if err != nil {
				return err
			}
			verb := "created"
			if updated {
				verb = "updated"
			}
			if jsonOut {
				return writeJSON(os.Stdout, map[string]any{"ruleset_id": id, "updated": updated, "contexts": show})
			}
			fmt.Printf("✓ %s ruleset %q (id %d) on %s — required: %s\n", verb, name, id, repo, strings.Join(show, ", "))
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "speckit-gate", "ruleset name")
	c.Flags().IntVar(&reviews, "reviews", 0, "required approving reviews on a PR")
	c.Flags().StringArrayVar(&contexts, "require", nil, "required status check context (repeatable; default: quality + verify / verify)")
	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	c.Flags().BoolVar(&jsonOut, "json", false, "emit output as JSON")
	return c
}
