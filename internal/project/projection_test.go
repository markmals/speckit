package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/config"
	"github.com/markmals/speckit/internal/coreassets"
)

// [scenario.init.projection.shared-prompt-content] the same authored prompt
// content reaches every adapter: for each command, the projected body under
// claude, codex, generic, and copilot (both agent and prompt files) is
// byte-identical to the authored Command body — only the agent-specific
// wrapper/location differs.
//
// SPEC: story.init.projection
func TestInitSharedPromptContent(t *testing.T) {
	commands := initCommandSet(t)

	roots := map[string]string{}
	for _, agent := range []string{"claude", "codex", "copilot", "generic"} {
		root := t.TempDir()
		if _, err := Init(root, coreassets.FS, Options{Integration: agent}); err != nil {
			t.Fatal(err)
		}
		roots[agent] = root
	}

	for _, c := range commands {
		projections := map[string]string{
			"claude skill":   filepath.Join(roots["claude"], ".claude", "skills", "speckit-"+c.Name, "SKILL.md"),
			"codex skill":    filepath.Join(roots["codex"], ".agents", "skills", "speckit-"+c.Name, "SKILL.md"),
			"generic skill":  filepath.Join(roots["generic"], ".agents", "skills", "speckit-"+c.Name, "SKILL.md"),
			"copilot agent":  filepath.Join(roots["copilot"], ".github", "agents", "speckit."+c.Name+".agent.md"),
			"copilot prompt": filepath.Join(roots["copilot"], ".github", "prompts", "speckit."+c.Name+".prompt.md"),
		}
		for label, p := range projections {
			if body := projectedBody(t, p); body != c.Body {
				t.Errorf("%s of %s diverges from the authored prompt body", label, c.Name)
			}
		}
	}
}

// projectedBody reads a projected command file and strips the agent-specific
// frontmatter wrapper, returning the prompt body.
func projectedBody(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read projection: %v", err)
	}
	s := string(b)
	const end = "\n---\n\n"
	i := strings.Index(s, end)
	if !strings.HasPrefix(s, "---\n") || i < 0 {
		t.Fatalf("%s: no frontmatter wrapper", path)
	}
	return s[i+len(end):]
}

// [scenario.init.projection.generic] generic projects the command set to
// .agents/skills/speckit-<cmd>/SKILL.md with AGENTS.md as the orientation
// file, and the projection is identical to codex's — same paths, same bytes —
// because .agents/ + AGENTS.md is the vendor-neutral convention. The only
// divergence in the whole tree is .speckit/specs.json recording which
// integration was chosen.
//
// SPEC: story.init.projection
func TestInitGenericMatchesCodex(t *testing.T) {
	generic, codex := t.TempDir(), t.TempDir()
	if _, err := Init(generic, coreassets.FS, Options{Integration: "generic"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(codex, coreassets.FS, Options{Integration: "codex"}); err != nil {
		t.Fatal(err)
	}

	for _, c := range initCommandSet(t) {
		mustExist(t, filepath.Join(generic, ".agents", "skills", "speckit-"+c.Name, "SKILL.md"))
	}
	mustExist(t, filepath.Join(generic, "AGENTS.md"))

	genManifest, codexManifest := manifestOf(t, generic), manifestOf(t, codex)
	if genManifest != codexManifest {
		t.Fatalf("generic and codex trees differ in shape:\n--- generic ---\n%s--- codex ---\n%s", genManifest, codexManifest)
	}
	for _, rel := range strings.Split(strings.TrimSpace(genManifest), "\n") {
		gb, err := os.ReadFile(filepath.Join(generic, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		cb, err := os.ReadFile(filepath.Join(codex, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if rel == config.File {
			// The lone permitted divergence: each records its own integration.
			g, _, _ := config.Load(generic)
			c, _, _ := config.Load(codex)
			if g.Agent != "generic" || c.Agent != "codex" {
				t.Errorf("recorded integrations = %q / %q, want generic / codex", g.Agent, c.Agent)
			}
			if strings.ReplaceAll(string(gb), "generic", "codex") != string(cb) {
				t.Errorf("%s differs beyond the recorded integration id", rel)
			}
			continue
		}
		if string(gb) != string(cb) {
			t.Errorf("generic and codex projections differ at %s", rel)
		}
	}
}

// [scenario.init.basic.cross-os] the projected tree is host-neutral: every
// path Init returns is built with the native separator (no foreign separator
// hardcoded anywhere in the relative tree), projection works from a root
// containing both a separator and a space, and the slash-normalized manifest
// equals the per-agent golden — so each supported host produces the identical
// tree modulo path separators. Init writes no git-hook trampolines today, so
// there is no .sh/.cmd variant to normalize. Run on each host's CI, this test
// is the per-host instantiation of the scenario; a projection hardcoding "\\"
// fails here on Linux/macOS, and one hardcoding "/" fails on Windows.
//
// SPEC: story.init.basic
func TestInitCrossOSPaths(t *testing.T) {
	foreign := "\\"
	if os.PathSeparator == '\\' {
		foreign = "/"
	}
	for _, agent := range []string{"claude", "codex", "copilot", "generic"} {
		t.Run(agent, func(t *testing.T) {
			// A root with a space exercises joining beyond the happy path.
			root := filepath.Join(t.TempDir(), "cross os")
			if err := os.MkdirAll(root, 0o755); err != nil {
				t.Fatal(err)
			}
			written, err := Init(root, coreassets.FS, Options{Integration: agent})
			if err != nil {
				t.Fatal(err)
			}
			prefix := root + string(os.PathSeparator)
			for _, p := range written {
				if filepath.Clean(p) != p {
					t.Errorf("unclean path returned: %q", p)
				}
				if !strings.HasPrefix(p, prefix) {
					t.Errorf("path escapes the init root: %q", p)
					continue
				}
				if rel := p[len(prefix):]; strings.Contains(rel, foreign) {
					t.Errorf("path hardcodes the foreign separator %q: %q", foreign, p)
				}
				if fi, err := os.Stat(p); err != nil || fi.IsDir() {
					t.Errorf("returned path is not a written file: %q (%v)", p, err)
				}
			}

			want, err := os.ReadFile(filepath.Join("testdata", "goldens", "init", agent+".files.txt"))
			if err != nil {
				t.Fatal(err)
			}
			if got := manifestOf(t, root); got != string(want) {
				t.Errorf("init tree not host-neutral for %s:\n--- got ---\n%s--- want ---\n%s", agent, got, want)
			}
		})
	}
}

// [scenario.init.projection.one-adapter] every integration is served by exactly
// one registered projection adapter (the Adapter interface), and choosing one
// integration writes only that adapter's surfaces — no other agent's
// directories or orientation files appear. The other half of D4 — the shared
// prompts staying unchanged across adapters — is proven by
// TestInitSharedPromptContent.
//
// SPEC: story.init.projection
func TestInitOneAdapterPerIntegration(t *testing.T) {
	surfaces := map[string][]string{
		"claude":  {".claude", "CLAUDE.md"},
		"codex":   {".agents", "AGENTS.md"},
		"generic": {".agents", "AGENTS.md"},
		"copilot": {".github"},
	}
	for _, integ := range []string{"claude", "codex", "copilot", "generic"} {
		t.Run(integ, func(t *testing.T) {
			a, ok := AdapterFor(integ)
			if !ok {
				t.Fatalf("no adapter registered for %q", integ)
			}
			if a.ID() != integ {
				t.Fatalf("adapter identity mismatch: %q -> %q", integ, a.ID())
			}

			root := t.TempDir()
			if _, err := Init(root, coreassets.FS, Options{Integration: integ}); err != nil {
				t.Fatal(err)
			}
			own := map[string]bool{}
			for _, s := range surfaces[integ] {
				own[s] = true
				mustExist(t, filepath.Join(root, s))
			}
			for other, marks := range surfaces {
				if other == integ {
					continue
				}
				for _, s := range marks {
					if own[s] {
						continue // codex/generic share the vendor-neutral surface
					}
					if _, err := os.Stat(filepath.Join(root, s)); !os.IsNotExist(err) {
						t.Errorf("init %s left %s's surface behind: %s", integ, other, s)
					}
				}
			}
		})
	}
}
