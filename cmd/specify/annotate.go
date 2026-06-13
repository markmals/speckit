package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/markmals/speckit/internal/engine"
	"github.com/markmals/speckit/internal/specmodel"
)

// outputFormat is a gate subcommand's --format value. github emits GitHub
// Actions workflow-command annotations (the same mechanism oxlint's
// `--format github` uses), so a failing spec gate shows up as an inline
// annotation on the offending file in the PR's Files-changed view.
type outputFormat string

const (
	formatText   outputFormat = "text"
	formatJSON   outputFormat = "json"
	formatGitHub outputFormat = "github"
)

// parseFormat validates a --format flag, defaulting empty to text.
func parseFormat(s string) (outputFormat, error) {
	switch outputFormat(s) {
	case "", formatText:
		return formatText, nil
	case formatJSON, formatGitHub:
		return outputFormat(s), nil
	default:
		return "", fmt.Errorf("unknown --format %q (want text|json|github)", s)
	}
}

// resolveFormat reconciles the --format flag with the legacy --json bool (json
// wins if set) for the reporting commands that carry both.
func resolveFormat(format string, jsonOut bool) (outputFormat, error) {
	if jsonOut {
		return formatJSON, nil
	}
	return parseFormat(format)
}

// verifyAnnotations builds GitHub annotations for a non-green verify: each
// unjoinable scenario at its spec line, each dangling binding at the test line,
// each failed scenario at its spec line, and each unbound test (no file) at the
// run. locs maps scenarios to their spec location.
func verifyAnnotations(v engine.VerifyResult, locs map[specmodel.SpecID]engine.SpecLocation) []string {
	var out []string
	for _, s := range v.Failed {
		loc := locs[s]
		out = append(out, ghCommand("error", loc.File, loc.Line, fmt.Sprintf("scenario %s failed", s)))
	}
	for _, s := range v.Unjoinable {
		loc := locs[s]
		out = append(out, ghCommand("error", loc.File, loc.Line, fmt.Sprintf("scenario %s is declared but has no bound test", s)))
	}
	for _, b := range v.Dangling {
		out = append(out, ghCommand("error", b.File, b.Line, fmt.Sprintf("test binds undeclared scenario %s", b.Scenario)))
	}
	for _, r := range v.Unbound {
		out = append(out, ghCommand("error", "", 0, fmt.Sprintf("test %q ran with no scenario binding", r.Name)))
	}
	return out
}

// parityAnnotations builds a GitHub annotation for every non-conforming parity
// cell, at the scenario's spec line.
func parityAnnotations(r engine.ParityReport, locs map[specmodel.SpecID]engine.SpecLocation) []string {
	var out []string
	for _, c := range r.Cells {
		if c.State == "conforming" {
			continue
		}
		loc := locs[c.Scenario]
		msg := fmt.Sprintf("scenario %s is %s", c.Scenario, c.State)
		if c.Reason != "" {
			msg += ": " + c.Reason
		}
		out = append(out, ghCommand("error", loc.File, loc.Line, msg))
	}
	return out
}

// ghCommand formats one GitHub Actions workflow-command annotation line. A
// blank file omits the file property (the annotation attaches to the run); a
// line ≤ 0 omits the line property (the annotation attaches to the file). See
// GitHub's "Workflow commands for GitHub Actions" — message and property values
// use distinct escape sets.
func ghCommand(level, file string, line int, message string) string {
	props := ""
	if file != "" {
		props = " file=" + ghEscapeProp(file)
		if line > 0 {
			props += ",line=" + strconv.Itoa(line)
		}
	}
	return fmt.Sprintf("::%s%s::%s", level, props, ghEscapeData(message))
}

var (
	ghDataEscaper = strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A")
	ghPropEscaper = strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A", ":", "%3A", ",", "%2C")
)

func ghEscapeData(s string) string { return ghDataEscaper.Replace(s) }
func ghEscapeProp(s string) string { return ghPropEscaper.Replace(s) }
