package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

// TestNpmPackageScaffold renders the embedded npm-package scaffold and asserts the
// single-library wiring: members land in packages/ (memberDir), the stack shares the
// node toolchain with web (family), the target uses Vitest's junit format with scoped
// bindings (the scenario id is read from SOURCE — the [scenario.<id>] prefix in the
// it() title — never from the report; scoped lets the library mix scenario tests with
// plain unit/property tests, matching the swift-package twin), the library source is
// name-agnostic (named by package.json, never interpolated), and the seeded story's
// scenario sub-ids match the [scenario.<id>] prefixes in the bound test — the agreement
// that makes a fresh `specify verify` green with the Node toolchain alone.
func TestNpmPackageScaffold(t *testing.T) {
	sub, err := fs.Sub(coreassets.FS, "templates/scaffolds/npm-package")
	if err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(sub)
	if err != nil {
		t.Fatal(err)
	}
	if m.Stack != "npm-package" || m.MemberDir != "packages" {
		t.Errorf("stack=%q memberDir=%q, want npm-package / packages", m.Stack, m.MemberDir)
	}
	if m.Family != "node" {
		t.Errorf("family=%q, want node (shares the node toolchain with web)", m.Family)
	}
	// junit + scoped bindings: the scenario id is read from SOURCE (the [scenario.<id>]
	// prefix in the it() title, scanned by the engine — never from the report, which
	// carries only test identity + outcome). scoped (like the swift library twins) lets a
	// library mix scenario tests with plain unit/property tests without the untagged ones
	// turning verify red.
	if m.Target.Format != "junit" || m.Target.Bindings != "scoped" {
		t.Errorf("target format=%q bindings=%q, want junit / scoped",
			m.Target.Format, m.Target.Bindings)
	}
	// The source is name-agnostic (named via package.json), so it needs no *identifier* rule
	// like the swift stacks. But the member name IS the npm package name, so it carries the
	// "npm" rule — rejecting capitals / reserved names / core modules at `target add` rather
	// than letting them surface at `npm publish`.
	if m.NameRule != "npm" {
		t.Errorf("nameRule=%q, want npm (the member name becomes the npm package name)",
			m.NameRule)
	}
	if m.SharedModule {
		t.Error("npm-package must not be sharedModule")
	}

	dir := t.TempDir()
	data := Data{Name: "string-kit", Dir: "packages/string-kit"}
	if _, err := Render(sub, dir, data); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		"mise.toml", ".gitignore", ".oxfmtrc.json", ".oxlintrc.json",
		"package.json", "tsconfig.json", "tsdown.config.ts", "vitest.config.ts",
		"README.md",
		"src/index.ts",
		"src/slugify.ts",
		"src/slugify.test.ts",
	} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(p))); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}

	// package.json carries the member name; the source files must not (the type is
	// name-agnostic and the module is named by package.json, like swift-package's lib).
	pkg := string(mustRead(t, filepath.Join(dir, "package.json")))
	if !strings.Contains(pkg, `"name": "string-kit"`) {
		t.Errorf("package.json missing the member name:\n%s", pkg)
	}
	src := string(mustRead(t, filepath.Join(dir, "src/slugify.ts")))
	if !strings.Contains(src, "export function slugify") {
		t.Errorf("slugify.ts missing the public function:\n%s", src)
	}
	if strings.Contains(src, "string-kit") || strings.Contains(src, "StringKit") {
		t.Errorf("library source must be name-agnostic, but interpolated the member name:\n%s", src)
	}

	// The bound test imports the sibling source and binds via the [scenario.<id>] prefix
	// on the it() title — the report-carried identity the engine joins against.
	test := string(mustRead(t, filepath.Join(dir, "src/slugify.test.ts")))
	for _, want := range []string{
		`from "vitest"`,
		`import { slugify } from "./slugify.ts"`,
		`it("[scenario.slug.create.basic]`,
		`it("[scenario.slug.create.symbols]`,
	} {
		if !strings.Contains(test, want) {
			t.Errorf("slugify.test.ts missing %q:\n%s", want, test)
		}
	}

	// The mise test task writes the junit report the engine joins, byte-identical to the
	// node family's canonical node:test so a second node member promotes cleanly.
	mise := string(mustRead(t, filepath.Join(dir, "mise.toml")))
	if !strings.Contains(mise, "vitest run --reporter=junit --outputFile=junit.xml") {
		t.Errorf("mise.toml test task must write the junit report:\n%s", mise)
	}

	// The seeded story (root/) and the test's [scenario.<id>] prefixes must name the same
	// scenario sub-ids, or a fresh verify would dangle.
	root := t.TempDir()
	if _, err := RenderRoot(sub, root, data); err != nil {
		t.Fatal(err)
	}
	story := string(mustRead(t, filepath.Join(root, "features/0001-slug/stories/slug.create.md")))
	for _, scen := range []string{"scenario.slug.create.basic", "scenario.slug.create.symbols"} {
		if !strings.Contains(story, "<!-- id: "+scen+" -->") {
			t.Errorf("seeded story missing scenario sub-id %q", scen)
		}
		if !strings.Contains(test, "["+scen+"]") {
			t.Errorf("seeded test missing the [scenario] prefix for %q", scen)
		}
	}

	// The rendered target wiring: verify runs `mise //packages/string-kit:test`, the
	// report lands at the member root, and the source dir is the library's src/.
	rt, err := RenderTarget(m, data)
	if err != nil {
		t.Fatal(err)
	}
	if rt.Command != "mise //packages/string-kit:test" {
		t.Errorf("target command = %q", rt.Command)
	}
	if rt.Report != "packages/string-kit/junit.xml" || rt.Source != "packages/string-kit/src" {
		t.Errorf("target report=%q source=%q", rt.Report, rt.Source)
	}
	if rt.Bindings != "scoped" {
		t.Errorf("target bindings=%q, want scoped", rt.Bindings)
	}
}
