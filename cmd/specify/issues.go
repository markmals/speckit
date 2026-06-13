package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/markmals/speckit/internal/github"
)

// issuesCmd is Pillar 2: defect intake on GitHub Issues. A defect is ephemeral —
// the fix adds/updates a scenario (or a regression test bound to one), and the
// issue closes on a green verify (the lock is the proof). These commands inherit
// gh's auth and need no config block.
func issuesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "issues",
		Short: "Defect intake on GitHub Issues (Pillar 2)",
	}
	addRepoFlag(c.PersistentFlags())
	c.AddCommand(issuesListCmd(), issuesCreateCmd(), issuesCloseCmd())
	return c
}

func issuesListCmd() *cobra.Command {
	var state string
	var labels []string
	var jsonOut bool
	c := &cobra.Command{
		Use:   "list",
		Short: "List issues (defect intake), optionally filtered by label",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, repo, err := resolveGitHub(repoFlag(cmd))
			if err != nil {
				return err
			}
			if state == "" {
				state = "open"
			}
			issues, err := client.ListIssues(cmd.Context(), repo, github.ListOptions{State: state, Labels: labels})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(os.Stdout, issues)
			}
			if len(issues) == 0 {
				fmt.Printf("No %s issues in %s%s.\n", state, repo, labelSuffix(labels))
				return nil
			}
			fmt.Printf("%s issues in %s%s:\n", state, repo, labelSuffix(labels))
			for _, i := range issues {
				lbls := ""
				if names := i.LabelNames(); len(names) > 0 {
					lbls = "  [" + strings.Join(names, ", ") + "]"
				}
				fmt.Printf("  #%-5d %s%s\n", i.Number, i.Title, lbls)
			}
			return nil
		},
	}
	c.Flags().StringVar(&state, "state", "open", "issue state: open|closed|all")
	c.Flags().StringArrayVar(&labels, "label", nil, "filter to issues carrying all of these labels (repeatable)")
	c.Flags().BoolVar(&jsonOut, "json", false, "emit output as JSON")
	return c
}

func issuesCreateCmd() *cobra.Command {
	var title, body, issueType string
	var labels []string
	var jsonOut, yes bool
	c := &cobra.Command{
		Use:   "create",
		Short: "Open an issue (e.g. a defect) — supersedes the agent-driven gh issue create",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(title) == "" {
				return fmt.Errorf("issues create: --title required")
			}
			client, repo, err := resolveGitHub(repoFlag(cmd))
			if err != nil {
				return err
			}
			ok, err := confirmAction(os.Stdin, os.Stdout, fmt.Sprintf("Create issue %q in %s?", title, repo), yes)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("aborted")
			}
			iss, err := client.CreateIssue(cmd.Context(), repo, github.CreateIssueInput{
				Title: title, Body: body, Labels: labels, Type: issueType,
			})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(os.Stdout, iss)
			}
			fmt.Printf("✓ opened #%d — %s\n", iss.Number, iss.HTMLURL)
			return nil
		},
	}
	c.Flags().StringVar(&title, "title", "", "issue title (required)")
	c.Flags().StringVar(&body, "body", "", "issue body (markdown)")
	c.Flags().StringArrayVar(&labels, "label", nil, "label to apply (repeatable); the portable fallback for issue type")
	c.Flags().StringVar(&issueType, "type", "", "org Issue Type (Bug|Feature|Task|Epic); ignored off-org")
	c.Flags().BoolVar(&jsonOut, "json", false, "emit the created issue as JSON")
	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	return c
}

func issuesCloseCmd() *cobra.Command {
	var jsonOut, yes bool
	c := &cobra.Command{
		Use:   "close <number>",
		Short: "Close an issue as completed (close-on-green: the lock is the proof)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("close: %q is not an issue number", args[0])
			}
			client, repo, err := resolveGitHub(repoFlag(cmd))
			if err != nil {
				return err
			}
			ok, err := confirmAction(os.Stdin, os.Stdout, fmt.Sprintf("Close #%d in %s?", number, repo), yes)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("aborted")
			}
			if err := client.CloseIssue(cmd.Context(), repo, number); err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(os.Stdout, map[string]any{"closed": number})
			}
			fmt.Printf("✓ closed #%d\n", number)
			return nil
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "emit output as JSON")
	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	return c
}

func labelSuffix(labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	return " labeled " + strings.Join(labels, "+")
}
