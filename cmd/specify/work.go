package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/markmals/speckit/internal/config"
	"github.com/markmals/speckit/internal/work"
	"github.com/markmals/speckit/internal/work/beads"
	"github.com/markmals/speckit/internal/work/ghprojects"
	"github.com/markmals/speckit/internal/work/markdown"
)

// workCmd drives the configured work-tracking provider (the "work" block in
// .speckit/specs.json; absent, the markdown provider on WORK.md). Durable
// truth stays in the repo — specs, locks, memory; work items are
// coordination.
//
// The board flags (--repo, --project, --owner, --status-field, --column)
// belong to the github-projects provider; the local providers ignore no
// flags because they read none of these.
func workCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "work",
		Short: "Track work items in the configured provider",
	}
	addRepoFlag(c.PersistentFlags())
	c.PersistentFlags().Int("project", 0, "github-projects: board number (default: the config's work.project)")
	c.PersistentFlags().String("owner", "", "github-projects: board owner (default: the config's work.owner, else the repo owner)")
	c.PersistentFlags().String("status-field", "", "github-projects: the single-select field holding the column (default Status)")
	c.PersistentFlags().StringArray("column", nil, "github-projects: map a state to a column as state=Column (repeatable), e.g. --column ready=Todo")
	c.AddCommand(workReadyCmd(), workCreateCmd(), workClaimCmd(), workMoveCmd(), workListCmd())
	return c
}

func workReadyCmd() *cobra.Command {
	var jsonOut bool
	c := &cobra.Command{
		Use:   "ready",
		Short: "List items ready to pick up",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := resolveWorkProvider(cmd)
			if err != nil {
				return err
			}
			if p == nil {
				return noWorkProvider()
			}
			items, err := p.Ready(cmd.Context())
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(os.Stdout, itemsOrEmpty(items))
			}
			if len(items) == 0 {
				fmt.Println("Nothing ready.")
				return nil
			}
			for _, it := range items {
				fmt.Println(itemLine(it, false))
			}
			return nil
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "emit output as JSON")
	return c
}

func workCreateCmd() *cobra.Command {
	var itemType, spec string
	var jsonOut, yes bool
	c := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a work item in the ready state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := strings.TrimSpace(args[0])
			if title == "" {
				return fmt.Errorf("create: a title is required")
			}
			switch itemType {
			case "", work.TypeTask, work.TypeDefect:
			default:
				return fmt.Errorf("create: unknown type %q (want %s or %s)", itemType, work.TypeTask, work.TypeDefect)
			}
			p, err := resolveWorkProvider(cmd)
			if err != nil {
				return err
			}
			if p == nil {
				return noWorkProvider()
			}
			if err := confirmOutward(p, fmt.Sprintf("Create %q?", title), yes); err != nil {
				return err
			}
			it, err := p.Create(cmd.Context(), work.CreateRequest{Title: title, Type: itemType, Spec: spec})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(os.Stdout, it)
			}
			fmt.Printf("✓ created %s\n", itemLine(it, false))
			return nil
		},
	}
	c.Flags().StringVar(&itemType, "type", work.TypeTask, "item type: task|defect")
	c.Flags().StringVar(&spec, "spec", "", "spec id this item advances")
	c.Flags().BoolVar(&jsonOut, "json", false, "emit the created item as JSON")
	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt (github-projects)")
	return c
}

func workClaimCmd() *cobra.Command {
	var jsonOut, yes bool
	c := &cobra.Command{
		Use:   "claim <id>",
		Short: "Claim an item: take it and move it to in-progress",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := resolveWorkProvider(cmd)
			if err != nil {
				return err
			}
			if p == nil {
				return noWorkProvider()
			}
			if err := confirmOutward(p, fmt.Sprintf("Claim %s (assign yourself, move to %s)?", args[0], work.StateInProgress), yes); err != nil {
				return err
			}
			it, err := p.Claim(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(os.Stdout, it)
			}
			fmt.Printf("✓ claimed %s → %s\n", itemLine(it, false), it.State)
			return nil
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "emit the claimed item as JSON")
	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt (github-projects)")
	return c
}

func workMoveCmd() *cobra.Command {
	var jsonOut, yes bool
	c := &cobra.Command{
		Use:   "move <id> <state>",
		Short: "Move an item to a state",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := resolveWorkProvider(cmd)
			if err != nil {
				return err
			}
			if p == nil {
				return noWorkProvider()
			}
			if err := confirmOutward(p, fmt.Sprintf("Move %s to %q?", args[0], args[1]), yes); err != nil {
				return err
			}
			it, err := p.Move(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(os.Stdout, it)
			}
			fmt.Printf("✓ moved %s → %s\n", it.ID, it.State)
			return nil
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "emit the moved item as JSON")
	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt (github-projects)")
	return c
}

func workListCmd() *cobra.Command {
	var state string
	var jsonOut bool
	c := &cobra.Command{
		Use:   "list",
		Short: "List work items, optionally by state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := resolveWorkProvider(cmd)
			if err != nil {
				return err
			}
			if p == nil {
				return noWorkProvider()
			}
			items, err := p.List(cmd.Context(), state)
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(os.Stdout, itemsOrEmpty(items))
			}
			if len(items) == 0 {
				fmt.Println("No items.")
				return nil
			}
			for _, it := range items {
				fmt.Println(itemLine(it, true))
			}
			return nil
		},
	}
	c.Flags().StringVar(&state, "state", "", "only items in this state")
	c.Flags().BoolVar(&jsonOut, "json", false, "emit output as JSON")
	return c
}

// resolveWorkProvider maps the config's work block to a provider — the only
// place the adapter packages are wired. Provider "none" returns (nil, nil):
// the verbs print one line and exit 0.
func resolveWorkProvider(cmd *cobra.Command) (work.Provider, error) {
	cfg, _, err := config.Load(".")
	if err != nil {
		return nil, err
	}
	w := cfg.WorkConfig()
	switch w.Provider {
	case config.WorkNone:
		return nil, nil
	case config.WorkMarkdown:
		return markdown.New(".", w.File), nil
	case config.WorkBeads:
		return beads.New()
	case config.WorkGitHubProjects:
		client, repo, err := resolveGitHub(repoFlag(cmd))
		if err != nil {
			return nil, err
		}
		opts := ghprojects.Options{Project: w.Project, Owner: w.Owner}
		if v, _ := cmd.Flags().GetInt("project"); v > 0 {
			opts.Project = v
		}
		if v, _ := cmd.Flags().GetString("owner"); v != "" {
			opts.Owner = v
		}
		opts.StatusField, _ = cmd.Flags().GetString("status-field")
		specs, _ := cmd.Flags().GetStringArray("column")
		opts.Columns, err = parseColumnOverrides(specs)
		if err != nil {
			return nil, err
		}
		return ghprojects.New(client, repo, opts)
	default:
		return nil, fmt.Errorf("unknown work provider %q (want one of %s)", w.Provider, strings.Join(config.WorkProviders, ", "))
	}
}

// parseColumnOverrides reads repeated --column state=Column pairs.
func parseColumnOverrides(specs []string) (map[string]string, error) {
	if len(specs) == 0 {
		return nil, nil
	}
	m := make(map[string]string, len(specs))
	for _, s := range specs {
		state, col, ok := strings.Cut(s, "=")
		if !ok || state == "" || col == "" {
			return nil, fmt.Errorf("--column wants state=Column (e.g. ready=Todo), got %q", s)
		}
		m[state] = col
	}
	return m, nil
}

// confirmOutward gates the github-projects provider's mutations behind the
// standard prompt — they are outward and hard to undo. The local providers
// (markdown, beads) write only committed or local state and never prompt.
func confirmOutward(p work.Provider, prompt string, assumeYes bool) error {
	if p.Name() != config.WorkGitHubProjects {
		return nil
	}
	ok, err := confirmAction(os.Stdin, os.Stdout, prompt, assumeYes)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("aborted")
	}
	return nil
}

// noWorkProvider reports the "none" provider. Configured off is not an
// error.
func noWorkProvider() error {
	fmt.Println("no work provider configured")
	return nil
}

// itemLine renders one item: id, then title, then the spec pointer when
// present; withState also shows the state (for list).
func itemLine(it work.Item, withState bool) string {
	var sb strings.Builder
	sb.WriteString(it.ID)
	if withState {
		sb.WriteString("  [")
		sb.WriteString(it.State)
		sb.WriteString("]")
	}
	sb.WriteString("  ")
	sb.WriteString(it.Title)
	if it.Spec != "" {
		sb.WriteString(" · spec: ")
		sb.WriteString(it.Spec)
	}
	return sb.String()
}

// itemsOrEmpty keeps --json emitting [] rather than null for an empty list.
func itemsOrEmpty(items []work.Item) []work.Item {
	if items == nil {
		return []work.Item{}
	}
	return items
}
