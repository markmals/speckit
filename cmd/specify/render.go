package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"github.com/markmals/speckit/internal/engine"
	"github.com/markmals/speckit/internal/specmodel"
)

// The default (non --json) output is styled with Lip Gloss; it degrades to
// plain, color-free text when stdout is not a TTY (pipes, CI). --json is the
// machine-readable path.

var (
	cGreen  = lipgloss.Color("42")
	cRed    = lipgloss.Color("203")
	cYellow = lipgloss.Color("221")
	cOrange = lipgloss.Color("208")
	cFaint  = lipgloss.Color("245")
	cBlue   = lipgloss.Color("39")

	titleStyle = lipgloss.NewStyle().Bold(true)
	faintStyle = lipgloss.NewStyle().Foreground(cFaint)
)

func stateColor(state string) lipgloss.Color {
	switch state {
	case "conforming", "clean", "passed", "locked":
		return cGreen
	case "declared-deviation":
		return cYellow
	case "drifted", "failed":
		return cRed
	case "suspect":
		return cOrange
	default: // missing and anything else
		return cFaint
	}
}

func stateSymbol(state string) string {
	switch state {
	case "conforming", "clean", "passed", "locked":
		return "✓"
	case "declared-deviation":
		return "~"
	case "drifted", "failed":
		return "✗"
	case "suspect":
		return "⚠"
	default:
		return "·"
	}
}

func colored(c lipgloss.Color, s string) string { return lipgloss.NewStyle().Foreground(c).Render(s) }

// title renders a "command · subject" heading.
func title(cmd, subject string) string {
	return titleStyle.Render(cmd) + faintStyle.Render(" · ") + colored(cBlue, subject)
}

func shortScenario(id specmodel.SpecID) string { return strings.TrimPrefix(string(id), "scenario.") }

// stateTable builds a two-or-three-column table whose state column is colored
// per row by lookup(row).
func stateTable(headers []string, rows [][]string, stateCol int, lookup func(row int) string) string {
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(faintStyle).
		Headers(headers...).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := lipgloss.NewStyle().Padding(0, 1)
			switch {
			case row == table.HeaderRow:
				return s.Bold(true).Foreground(cFaint)
			case col == stateCol && row < len(rows):
				return s.Foreground(stateColor(lookup(row)))
			case col == len(headers)-1 && len(headers) == 3:
				return s.Foreground(cFaint) // the trailing note column
			default:
				return s
			}
		})
	for _, r := range rows {
		t.Row(r...)
	}
	return t.String()
}

func renderParity(r engine.ParityReport) string {
	rows := make([][]string, len(r.Cells))
	for i, c := range r.Cells {
		rows[i] = []string{shortScenario(c.Scenario), stateSymbol(c.State) + " " + c.State, c.Reason}
	}
	tbl := stateTable([]string{"SCENARIO", "STATE", "NOTE"}, rows, 1, func(row int) string { return r.Cells[row].State })

	counts := map[string]int{}
	for _, c := range r.Cells {
		counts[c.State]++
	}
	var parts []string
	for _, s := range []string{"conforming", "declared-deviation", "drifted", "suspect", "missing"} {
		if counts[s] > 0 {
			parts = append(parts, colored(stateColor(s), fmt.Sprintf("%d %s", counts[s], s)))
		}
	}
	return title("parity", r.Target) + "\n" + tbl + "\n  " + strings.Join(parts, faintStyle.Render("  ·  "))
}

func renderCover(r engine.CoverReport) string {
	if len(r.Cells) == 0 {
		return title("cover", string(r.Spec)) + "\n  " + faintStyle.Render("no targets have lock state yet")
	}
	rows := make([][]string, len(r.Cells))
	for i, c := range r.Cells {
		rows[i] = []string{c.Target, stateSymbol(c.State) + " " + c.State}
	}
	tbl := stateTable([]string{"PLATFORM", "STATE"}, rows, 1, func(row int) string { return r.Cells[row].State })
	return title("cover", string(r.Spec)) + "\n" + tbl
}

func renderDrift(r engine.DriftReport, target string) string {
	var b strings.Builder
	b.WriteString(title("drift", target) + "\n")
	for _, id := range r.Drifted {
		b.WriteString("  " + colored(cRed, "✗ "+string(id)) + "\n")
	}
	for _, id := range r.Missing {
		b.WriteString("  " + faintStyle.Render("· "+string(id)) + "\n")
	}
	var parts []string
	if len(r.Drifted) > 0 {
		parts = append(parts, colored(cRed, fmt.Sprintf("%d drifted", len(r.Drifted))))
	}
	if len(r.Missing) > 0 {
		parts = append(parts, faintStyle.Render(fmt.Sprintf("%d missing", len(r.Missing))))
	}
	parts = append(parts, colored(cGreen, fmt.Sprintf("%d clean", len(r.Clean))))
	b.WriteString("  " + strings.Join(parts, faintStyle.Render("  ·  ")))
	return b.String()
}

func renderVerify(v engine.VerifyResult, locked []specmodel.SpecID, target string) string {
	var b strings.Builder
	b.WriteString(title("verify", target) + "\n")
	for _, s := range v.Failed {
		b.WriteString("  " + colored(cRed, "✗ failed      "+string(s)) + "\n")
	}
	for _, s := range v.Unjoinable {
		b.WriteString("  " + colored(cOrange, "⚠ unjoinable  "+string(s)) + "\n")
	}
	for _, bd := range v.Dangling {
		b.WriteString("  " + colored(cOrange, "⚠ dangling    "+string(bd.Scenario)) + "\n")
	}
	for _, r := range v.Unbound {
		b.WriteString("  " + colored(cOrange, "⚠ unbound     "+r.Name) + "\n")
	}
	status := colored(cGreen, "✓ green")
	if !v.Green() {
		status = colored(cRed, "✗ not green")
	}
	b.WriteString("  " + status + faintStyle.Render(fmt.Sprintf("   %d passed · %d failed · %d locked",
		len(v.Passed), len(v.Failed), len(locked))))
	return b.String()
}

func renderScan(findings []specmodel.Finding) string {
	if len(findings) == 0 {
		return colored(cGreen, "✓ scan: clean")
	}
	var b strings.Builder
	for _, f := range findings {
		b.WriteString(colored(cRed, "✗ "+f.Invariant) + "  " + f.Path + faintStyle.Render("  "+f.Message) + "\n")
	}
	b.WriteString(faintStyle.Render(fmt.Sprintf("%d finding(s)", len(findings))))
	return b.String()
}

func renderGate(findings []engine.GateFinding) string {
	if len(findings) == 0 {
		return colored(cGreen, "✓ gate: clean")
	}
	lines := make([]string, len(findings))
	for i, f := range findings {
		line := colored(cRed, "✗ "+f.Check)
		if f.Path != "" {
			line += "  " + f.Path
		}
		lines[i] = line + faintStyle.Render("  "+f.Message)
	}
	return strings.Join(lines, "\n")
}

func renderInit(integration, root string, n int) string {
	return colored(cGreen, "✓ ") + titleStyle.Render("Initialized SpecKit") +
		faintStyle.Render(fmt.Sprintf(" (%s) at %s — %d paths", integration, root, n))
}

func renderLock(target, id string) string {
	return colored(cGreen, "✓ locked ") + id + faintStyle.Render(" on "+target)
}
