package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/markmals/speckit/internal/github"
)

// Default column set, confirmed against APL-Innovation-Lab/projects/1 (the
// "Meeting Room Reservations" board): Backlog → Ready → In Progress → On Hold →
// Cancelled → Closed. "Ready" is the actionable column (the ready queue), not a
// computed field; "On Hold" is the blocked-signal column, which `ready` skips for
// free by only listing the actionable column. These are FLAGS (--column /
// --status-field), so `specify work` drives any board with a different set.
const (
	defaultStatusField = "Status"
	defaultReadyColumn = "Ready"
	defaultDoingColumn = "In Progress"
	discoveredLabel    = "discovered-from"
)

// workCmd is Pillar 3: the agent's work surface on GitHub Projects (Beads-informed,
// simplified). The board is ephemeral coordination, driven via the inlined Projects
// GraphQL client; durable truth stays in the repo (specs/locks/memory).
//
// Column names default to the confirmed APL-Innovation-Lab board set (see the
// const block); --column / --status-field override them for any other board.
func workCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "work",
		Short: "Drive the agent's GitHub Projects board (Pillar 3)",
	}
	addRepoFlag(c.PersistentFlags())
	c.AddCommand(workReadyCmd(), workClaimCmd(), workMoveCmd(), workDiscoverCmd())
	return c
}

func workReadyCmd() *cobra.Command {
	var project int
	var owner, statusField, column string
	var jsonOut bool
	c := &cobra.Command{
		Use:   "ready",
		Short: "List the actionable column (the ready queue)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, repo, err := resolveGitHub(repoFlag(cmd))
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			proj, err := client.ResolveProject(ctx, ownerOr(owner, repo), project)
			if err != nil {
				return err
			}
			items, err := client.ListItems(ctx, proj.ID, statusField)
			if err != nil {
				return err
			}
			ready := items[:0]
			for _, it := range items {
				// Positively keep OPEN items only — excludes CLOSED issues and
				// CLOSED/MERGED PRs in one check.
				if strings.EqualFold(it.Status, column) && strings.EqualFold(it.State, "OPEN") {
					ready = append(ready, it)
				}
			}
			if jsonOut {
				return writeJSON(os.Stdout, ready)
			}
			if len(ready) == 0 {
				fmt.Printf("Nothing in %q on %q.\n", column, proj.Title)
				return nil
			}
			fmt.Printf("Ready (%q) on %q:\n", column, proj.Title)
			for _, it := range ready {
				fmt.Printf("  #%-5d %s\n", it.Number, it.Title)
			}
			return nil
		},
	}
	c.Flags().IntVar(&project, "project", 0, "project number (required)")
	c.Flags().StringVar(&owner, "owner", "", "project owner (default: the repo owner)")
	c.Flags().StringVar(&statusField, "status-field", defaultStatusField, "the single-select field that holds the column")
	c.Flags().StringVar(&column, "column", defaultReadyColumn, "the actionable column")
	c.Flags().BoolVar(&jsonOut, "json", false, "emit output as JSON")
	_ = c.MarkFlagRequired("project")
	return c
}

func workClaimCmd() *cobra.Command {
	var project int
	var owner, statusField, column string
	var yes bool
	c := &cobra.Command{
		Use:   "claim <issue#>",
		Short: "Claim an issue: assign yourself and move it to the in-progress column",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("claim: %q is not an issue number", args[0])
			}
			client, repo, err := resolveGitHub(repoFlag(cmd))
			if err != nil {
				return err
			}
			ok, err := confirmAction(os.Stdin, os.Stdout, fmt.Sprintf("Claim #%d (assign yourself, move to %q) in %s?", number, column, repo), yes)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("aborted")
			}
			ctx := cmd.Context()
			// Preflight everything that can fail WITHOUT side effects, so a bad
			// column/project/issue can't leave the issue half-claimed.
			proj, err := client.ResolveProject(ctx, ownerOr(owner, repo), project)
			if err != nil {
				return err
			}
			field, opt, err := resolveColumn(proj, statusField, column)
			if err != nil {
				return err
			}
			iss, err := client.GetIssue(ctx, repo, number)
			if err != nil {
				return err
			}
			login, err := client.Viewer(ctx)
			if err != nil {
				return err
			}
			// Advisory exclusivity: assignment has no native compare-and-swap, so
			// refuse if someone else already holds it (re-claiming your own is fine).
			if other := assignedToOther(iss, login); other != "" {
				return fmt.Errorf("#%d is already assigned to @%s — not claiming (reassign on GitHub to override)", number, other)
			}
			// Mutate: assign (the claim), then add to board + move. If a post-assign
			// step fails, say so explicitly rather than hiding the partial state.
			if err := client.AssignIssue(ctx, repo, number, []string{login}); err != nil {
				return err
			}
			if _, err := moveItem(ctx, client, proj, field, opt, iss.NodeID); err != nil {
				return fmt.Errorf("assigned #%d to @%s but failed to move the card to %q: %w", number, login, column, err)
			}
			fmt.Printf("✓ claimed #%d as @%s → %q\n", number, login, column)
			return nil
		},
	}
	c.Flags().IntVar(&project, "project", 0, "project number (required)")
	c.Flags().StringVar(&owner, "owner", "", "project owner (default: the repo owner)")
	c.Flags().StringVar(&statusField, "status-field", defaultStatusField, "the single-select field that holds the column")
	c.Flags().StringVar(&column, "column", defaultDoingColumn, "the in-progress column to move to")
	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	_ = c.MarkFlagRequired("project")
	return c
}

func workMoveCmd() *cobra.Command {
	var project int
	var owner, statusField, to string
	var yes bool
	c := &cobra.Command{
		Use:   "move <issue#>",
		Short: "Move an issue's card to a column",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("move: %q is not an issue number", args[0])
			}
			if to == "" {
				return fmt.Errorf("move: --to <column> required")
			}
			client, repo, err := resolveGitHub(repoFlag(cmd))
			if err != nil {
				return err
			}
			ok, err := confirmAction(os.Stdin, os.Stdout, fmt.Sprintf("Move #%d to %q in %s?", number, to, repo), yes)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("aborted")
			}
			ctx := cmd.Context()
			proj, err := client.ResolveProject(ctx, ownerOr(owner, repo), project)
			if err != nil {
				return err
			}
			field, opt, err := resolveColumn(proj, statusField, to)
			if err != nil {
				return err
			}
			iss, err := client.GetIssue(ctx, repo, number)
			if err != nil {
				return err
			}
			if _, err := moveItem(ctx, client, proj, field, opt, iss.NodeID); err != nil {
				return err
			}
			fmt.Printf("✓ moved #%d → %q\n", number, to)
			return nil
		},
	}
	c.Flags().IntVar(&project, "project", 0, "project number (required)")
	c.Flags().StringVar(&owner, "owner", "", "project owner (default: the repo owner)")
	c.Flags().StringVar(&statusField, "status-field", defaultStatusField, "the single-select field that holds the column")
	c.Flags().StringVar(&to, "to", "", "destination column (required)")
	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	_ = c.MarkFlagRequired("project")
	return c
}

func workDiscoverCmd() *cobra.Command {
	var project, from int
	var owner, title, body string
	var labels []string
	var yes, jsonOut bool
	c := &cobra.Command{
		Use:   "discover",
		Short: "File a mid-task follow-up issue with discovered-from provenance",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(title) == "" {
				return fmt.Errorf("discover: --title required")
			}
			if from <= 0 {
				return fmt.Errorf("discover: --from <issue#> required (the issue this was discovered from)")
			}
			client, repo, err := resolveGitHub(repoFlag(cmd))
			if err != nil {
				return err
			}
			ok, err := confirmAction(os.Stdin, os.Stdout, fmt.Sprintf("File follow-up %q (discovered from #%d) in %s?", title, from, repo), yes)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("aborted")
			}
			ctx := cmd.Context()
			// The constant label makes discovered work filterable; the #N backlink is
			// the provenance edge (GitHub cross-references it). Together they fill the
			// one gap GitHub has no native edge for.
			if err := client.EnsureLabel(ctx, repo, discoveredLabel, "5319e7", "Follow-up work discovered mid-task"); err != nil {
				return err
			}
			full := strings.TrimRight(body, "\n")
			if full != "" {
				full += "\n\n"
			}
			full += fmt.Sprintf("Discovered while working on #%d.", from)
			iss, err := client.CreateIssue(ctx, repo, github.CreateIssueInput{
				Title:  title,
				Body:   full,
				Labels: append([]string{discoveredLabel}, labels...),
			})
			if err != nil {
				return err
			}
			if project > 0 {
				proj, err := client.ResolveProject(ctx, ownerOr(owner, repo), project)
				if err == nil {
					_, _ = client.AddItem(ctx, proj.ID, iss.NodeID) // board sync is best-effort, never fatal
				}
			}
			if jsonOut {
				return writeJSON(os.Stdout, iss)
			}
			fmt.Printf("✓ filed #%d (discovered-from #%d) — %s\n", iss.Number, from, iss.HTMLURL)
			return nil
		},
	}
	c.Flags().StringVar(&title, "title", "", "issue title (required)")
	c.Flags().StringVar(&body, "body", "", "issue body (the backlink is appended)")
	c.Flags().IntVar(&from, "from", 0, "the issue this work was discovered from (required)")
	c.Flags().IntVar(&project, "project", 0, "also add the new issue to this project")
	c.Flags().StringVar(&owner, "owner", "", "project owner (default: the repo owner)")
	c.Flags().StringArrayVar(&labels, "label", nil, "extra label (repeatable)")
	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	c.Flags().BoolVar(&jsonOut, "json", false, "emit the created issue as JSON")
	return c
}

// resolveColumn looks up the status field and the destination column option on a
// resolved project — a pure, side-effect-free preflight so claim/move can validate
// before mutating anything.
func resolveColumn(proj github.Project, statusField, column string) (github.Field, github.FieldOption, error) {
	if statusField == "" {
		statusField = defaultStatusField
	}
	field, ok := proj.Field(statusField)
	if !ok {
		return github.Field{}, github.FieldOption{}, fmt.Errorf("project %q has no field %q", proj.Title, statusField)
	}
	opt, ok := field.Option(column)
	if !ok {
		return github.Field{}, github.FieldOption{}, fmt.Errorf("field %q has no column %q (have: %s)", field.Name, column, optionNames(field))
	}
	return field, opt, nil
}

// moveItem ensures the issue (by content node id) is on the board (AddItem is
// idempotent) and sets its status to the resolved column.
func moveItem(ctx context.Context, client *github.Client, proj github.Project, field github.Field, opt github.FieldOption, nodeID string) (string, error) {
	itemID, err := client.AddItem(ctx, proj.ID, nodeID)
	if err != nil {
		return "", err
	}
	if err := client.SetSingleSelect(ctx, proj.ID, itemID, field.ID, opt.ID); err != nil {
		return "", err
	}
	return itemID, nil
}

// assignedToOther returns the login of an assignee that isn't the viewer (so a
// re-claim by the same user is allowed), or "" if unassigned or only self-assigned.
func assignedToOther(iss github.Issue, login string) string {
	for _, a := range iss.Assignees {
		if a.Login != login {
			return a.Login
		}
	}
	return ""
}

// ownerOr returns the explicit owner flag, or the repo owner as the default.
func ownerOr(owner string, repo github.Repo) string {
	if owner != "" {
		return owner
	}
	return repo.Owner
}

func optionNames(f github.Field) string {
	names := make([]string, len(f.Options))
	for i, o := range f.Options {
		names[i] = o.Name
	}
	return strings.Join(names, ", ")
}
