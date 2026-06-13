package main

import (
	"fmt"
	"strings"
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

// ghCommand formats one GitHub Actions workflow-command annotation line. A
// blank file omits the file property, so the annotation attaches to the run
// rather than a source line. See GitHub's "Workflow commands for GitHub
// Actions" — message and property values use distinct escape sets.
func ghCommand(level, file, message string) string {
	if file == "" {
		return fmt.Sprintf("::%s::%s", level, ghEscapeData(message))
	}
	return fmt.Sprintf("::%s file=%s::%s", level, ghEscapeProp(file), ghEscapeData(message))
}

var (
	ghDataEscaper = strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A")
	ghPropEscaper = strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A", ":", "%3A", ",", "%2C")
)

func ghEscapeData(s string) string { return ghDataEscaper.Replace(s) }
func ghEscapeProp(s string) string { return ghPropEscaper.Replace(s) }
