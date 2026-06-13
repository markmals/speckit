package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/markmals/speckit/internal/github"
)

// resolveGitHub resolves the current repo from gh's context and a token-bearing
// client (gh auth inherited). Both halves give actionable errors so a missing repo
// or unauthenticated gh reads clearly. Only the GitHub-native commands call this —
// the offline engine never does.
func resolveGitHub() (*github.Client, github.Repo, error) {
	repo, err := github.CurrentRepo()
	if err != nil {
		return nil, github.Repo{}, err
	}
	c, err := github.New()
	if err != nil {
		return nil, github.Repo{}, err
	}
	return c, repo, nil
}

// resolveGitHubRepo resolves just the owner/name string from gh's context, for
// commands that delegate auth to gh itself (e.g. `gh secret set`) and so need no
// in-process token.
func resolveGitHubRepo() (string, error) {
	repo, err := github.CurrentRepo()
	if err != nil {
		return "", err
	}
	return repo.String(), nil
}

// confirmAction gates an outward, hard-to-undo action (create issue, move a card,
// provision a ruleset). assumeYes (the --yes flag) short-circuits to true; a
// non-interactive stdin with no --yes fails safe (returns false + error) rather
// than hanging or proceeding silently — mirrors the reconcile/taskstoissues guard.
func confirmAction(r io.Reader, w io.Writer, prompt string, assumeYes bool) (bool, error) {
	if assumeYes {
		return true, nil
	}
	fmt.Fprintf(w, "%s [y/N]: ", prompt)
	sc := bufio.NewScanner(r)
	if !sc.Scan() {
		return false, fmt.Errorf("no confirmation on stdin (pass --yes to proceed non-interactively)")
	}
	switch strings.ToLower(strings.TrimSpace(sc.Text())) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
