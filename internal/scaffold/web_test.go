package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markmals/speckit/internal/coreassets"
)

// TestWebScaffold renders the real embedded web scaffold — a guard against a
// broken scaffold.json or an unrenderable template.
func TestWebScaffold(t *testing.T) {
	sub, err := fs.Sub(coreassets.FS, "templates/scaffolds/web")
	if err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(sub)
	if err != nil {
		t.Fatal(err)
	}
	if m.Stack != "web" {
		t.Fatalf("stack = %q", m.Stack)
	}
	data := Data{Name: "web", Dir: "apps/web"}

	app := t.TempDir()
	if _, err := Render(sub, app, data); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		"package.json", "mise.toml", "tsconfig.json", "vitest.config.ts",
		"tsr.config.json", ".oxlintrc.json", ".oxfmtrc.json",
		"app/root.tsx", "app/router.tsx", "app/routes.ts", "app/routes/home.tsx",
		"app/providers.tsx", "pnpm-workspace.yaml",
		"app/styles/tailwind.css", "app/styles/cva.ts", "app/components/foundation/button.tsx",
		"app/lib/greeting.ts", "app/lib/greeting.test.ts",
	} {
		if _, err := os.Stat(filepath.Join(app, p)); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}
	// the .tmpl suffix must be stripped
	if _, err := os.Stat(filepath.Join(app, "package.json.tmpl")); !os.IsNotExist(err) {
		t.Error("package.json.tmpl suffix not stripped")
	}
	// package.json wires the #/* subpath import alias + the target name.
	pkg, _ := os.ReadFile(filepath.Join(app, "package.json"))
	for _, want := range []string{`"name": "web"`, `"#/*": "./app/*"`} {
		if !strings.Contains(string(pkg), want) {
			t.Errorf("package.json missing %q:\n%s", want, pkg)
		}
	}
	// vite.config.ts is runtime-specific (the cloudflare runtime is the default).
	// Render it over the base and assert TanStack Start + React Compiler + Tailwind
	// + the cloudflare() plugin, plus wrangler.jsonc and the pnpm build allowlist.
	if m.RuntimeDefault != "cloudflare" {
		t.Errorf("web runtimeDefault = %q, want cloudflare", m.RuntimeDefault)
	}
	for _, k := range []string{"cloudflare", "node"} {
		if _, ok := m.Runtime[k]; !ok {
			t.Errorf("web scaffold missing --runtime %q", k)
		}
	}
	if _, err := RenderVariant(sub, m.Runtime["cloudflare"], app, data); err != nil {
		t.Fatal(err)
	}
	vite, _ := os.ReadFile(filepath.Join(app, "vite.config.ts"))
	for _, want := range []string{"tanstackStart(", `srcDirectory: "app"`, "reactCompilerPreset(", "tailwindcss(", "cloudflare("} {
		if !strings.Contains(string(vite), want) {
			t.Errorf("cloudflare vite.config.ts missing %q", want)
		}
	}
	wrangler, _ := os.ReadFile(filepath.Join(app, "wrangler.jsonc"))
	if !strings.Contains(string(wrangler), `"name": "web"`) || strings.Contains(string(wrangler), "{{") {
		t.Errorf("wrangler.jsonc name not substituted:\n%s", wrangler)
	}
	if _, err := os.Stat(filepath.Join(app, "pnpm-workspace.yaml")); err != nil {
		t.Errorf("cloudflare runtime missing pnpm-workspace.yaml (build allowlist): %v", err)
	}
	// the node runtime ships a plain (non-cloudflare) vite.config.
	if nv := m.Runtime["node"]; nv.Files == "" {
		t.Error("node runtime variant has no files")
	}
	// the quality CI job calls these standard task names — they must exist.
	mise, err := os.ReadFile(filepath.Join(app, "mise.toml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range []string{"[tasks.lint]", `[tasks."fmt:check"]`, "[tasks.typecheck]"} {
		if !strings.Contains(string(mise), task) {
			t.Errorf("mise.toml missing quality task %s", task)
		}
	}

	root := t.TempDir()
	w, err := RenderRoot(sub, root, data)
	if err != nil {
		t.Fatal(err)
	}
	if len(w) == 0 {
		t.Fatal("RenderRoot seeded no example feature")
	}
	if _, err := os.Stat(filepath.Join(root, "features/0001-welcome/stories/welcome.greet.md")); err != nil {
		t.Errorf("seeded feature missing: %v", err)
	}

	rt, err := RenderTarget(m, data)
	if err != nil {
		t.Fatal(err)
	}
	if rt.Format != "junit" || rt.Source != "apps/web/app" {
		t.Errorf("RenderTarget = %+v", rt)
	}

	// The github/ subtree drops a project-root CI workflow. Its one GitHub
	// expression (${{ github.ref }}) must survive Go text/template intact, and
	// the scaffold vars must be substituted.
	proj := t.TempDir()
	gh, err := RenderGitHub(sub, proj, data)
	if err != nil {
		t.Fatal(err)
	}
	ciPath := filepath.Join(proj, ".github/workflows/ci.yml")
	ci, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("RenderGitHub did not write ci.yml: %v (wrote %v)", err, gh)
	}
	for _, want := range []string{"target: web", "working_directory: apps/web", "group: ci-${{ github.ref }}", "fmt:check"} {
		if !strings.Contains(string(ci), want) {
			t.Errorf("ci.yml missing %q\n%s", want, ci)
		}
	}
	// the Go-template escape artifact (`{{ "${{" }}`) must be fully resolved —
	// only the GitHub expression ${{ github.ref }} should remain.
	if strings.Contains(string(ci), `{{ "`) || strings.Contains(string(ci), `.Name`) {
		t.Errorf("ci.yml has unrendered template syntax:\n%s", ci)
	}

	// Pillar 2: the github/ subtree also drops the defect-intake surface. Every
	// file must land at its double-nested .github/ path.
	for _, p := range []string{
		".github/PULL_REQUEST_TEMPLATE.md",
		".github/ISSUE_TEMPLATE/defect.yml",
		".github/ISSUE_TEMPLATE/config.yml",
		".github/CODEOWNERS",
		".github/dependabot.yml",
	} {
		if _, err := os.Stat(filepath.Join(proj, filepath.FromSlash(p))); err != nil {
			t.Errorf("RenderGitHub did not write %s: %v", p, err)
		}
	}
	// the defect form stamps the Bug issue type; CODEOWNERS gates the spec library.
	defect, _ := os.ReadFile(filepath.Join(proj, ".github/ISSUE_TEMPLATE/defect.yml"))
	if !strings.Contains(string(defect), "type: Bug") {
		t.Errorf("defect.yml missing `type: Bug`:\n%s", defect)
	}
	owners, _ := os.ReadFile(filepath.Join(proj, ".github/CODEOWNERS"))
	if !strings.Contains(string(owners), "/features/") || !strings.Contains(string(owners), "/specs/") {
		t.Errorf("CODEOWNERS must route /features and /specs:\n%s", owners)
	}
	// dependabot.yml is templated: the npm ecosystem points at the app dir.
	dep, _ := os.ReadFile(filepath.Join(proj, ".github/dependabot.yml"))
	if !strings.Contains(string(dep), "directory: /apps/web") {
		t.Errorf("dependabot.yml npm directory not substituted to the app dir:\n%s", dep)
	}
	if strings.Contains(string(dep), "{{") {
		t.Errorf("dependabot.yml has unrendered template syntax:\n%s", dep)
	}
	// the convex data variant overwrites the base router and adds the convex files
	if m.DataDefault != "convex" {
		t.Errorf("web dataDefault = %q, want convex", m.DataDefault)
	}
	for _, k := range []string{"convex", "none"} {
		if _, ok := m.Data[k]; !ok {
			t.Errorf("web scaffold missing --data %q", k)
		}
	}
	cvx := m.Data["convex"]
	if len(cvx.Add) == 0 || len(cvx.Scripts) == 0 {
		t.Errorf("convex variant should declare deps + a codegen script: %+v", cvx)
	}
	dataDir := t.TempDir()
	if _, err := Render(sub, dataDir, data); err != nil {
		t.Fatal(err)
	}
	if _, err := RenderVariant(sub, cvx, dataDir, data); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"convex/schema.ts", "convex/messages.ts", "app/data/convex.ts", "pnpm-workspace.yaml"} {
		if _, err := os.Stat(filepath.Join(dataDir, filepath.FromSlash(p))); err != nil {
			t.Errorf("convex variant missing %s: %v", p, err)
		}
	}
	// RenderData overwrites the shared base router with the Convex-wired one — which
	// still delegates its Wrap to <Providers>, so feature providers compose with it.
	router, _ := os.ReadFile(filepath.Join(dataDir, "app/router.tsx"))
	if !strings.Contains(string(router), "ConvexProvider") {
		t.Errorf("convex variant did not overwrite app/router.tsx:\n%s", router)
	}
	if !strings.Contains(string(router), "<Providers>") {
		t.Errorf("convex variant router.tsx must keep the <Providers> Wrap seam:\n%s", router)
	}

	// the clerk --with feature adds @clerk and wraps root.tsx with ClerkProvider
	clerk, ok := m.Features["clerk"]
	if !ok || len(clerk.Add) == 0 {
		t.Errorf("web scaffold missing the clerk feature with deps: %+v", m.Features)
	}
	featDir := t.TempDir()
	if _, err := Render(sub, featDir, data); err != nil {
		t.Fatal(err)
	}
	if _, err := RenderFeature(sub, clerk, featDir, data); err != nil {
		t.Fatal(err)
	}
	croot, _ := os.ReadFile(filepath.Join(featDir, "app/root.tsx"))
	if !strings.Contains(string(croot), "ClerkProvider") {
		t.Errorf("clerk feature did not wrap root.tsx with ClerkProvider:\n%s", croot)
	}
	if _, err := os.Stat(filepath.Join(featDir, "app/start.ts")); err != nil {
		t.Errorf("clerk feature missing app/start.ts: %v", err)
	}

	// the tiptap --with feature is purely additive: it adds the @tiptap deps and a
	// new foundation editor component, and overwrites no shared file — so it
	// composes with clerk (which owns root.tsx) and with any data/runtime variant.
	tiptap, ok := m.Features["tiptap"]
	if !ok || len(tiptap.Add) != 3 {
		t.Errorf("web scaffold missing the tiptap feature with its 3 deps: %+v", m.Features)
	}
	ttDir := t.TempDir()
	if _, err := Render(sub, ttDir, data); err != nil {
		t.Fatal(err)
	}
	rootBefore, _ := os.ReadFile(filepath.Join(ttDir, "app/root.tsx"))
	if _, err := RenderFeature(sub, tiptap, ttDir, data); err != nil {
		t.Fatal(err)
	}
	editor, err := os.ReadFile(filepath.Join(ttDir, "app/components/foundation/editor.tsx"))
	if err != nil {
		t.Errorf("tiptap feature missing app/components/foundation/editor.tsx: %v", err)
	}
	if !strings.Contains(string(editor), "RichTextEditor") || !strings.Contains(string(editor), "useEditor") {
		t.Errorf("tiptap editor component missing RichTextEditor/useEditor:\n%s", editor)
	}
	if rootAfter, _ := os.ReadFile(filepath.Join(ttDir, "app/root.tsx")); string(rootAfter) != string(rootBefore) {
		t.Error("tiptap feature must not overwrite the shared app/root.tsx (it must stay additive)")
	}

	// the provider seam: the base router delegates its Wrap to <Providers>, and the
	// base providers.tsx is a no-op until a provider feature is selected.
	if br, _ := os.ReadFile(filepath.Join(app, "app/router.tsx")); !strings.Contains(string(br), "<Providers>") {
		t.Errorf("base router.tsx Wrap does not delegate to <Providers>:\n%s", br)
	}
	if bp, _ := os.ReadFile(filepath.Join(app, "app/providers.tsx")); strings.Contains(string(bp), "PostHogProvider") {
		t.Errorf("base providers.tsx (no features) must be a no-op, got:\n%s", bp)
	}

	// the posthog --with feature adds posthog-js and activates the PostHogProvider
	// block in the shared providers.tsx (the feature carries no files — it composes
	// at the Wrap seam), so it stacks with clerk (root.tsx) and any data provider.
	posthog, ok := m.Features["posthog"]
	if !ok || len(posthog.Add) == 0 {
		t.Errorf("web scaffold missing the posthog feature with deps: %+v", m.Features)
	}
	phDir := t.TempDir()
	if _, err := Render(sub, phDir, Data{Name: "web", Dir: "apps/web", Features: map[string]bool{"posthog": true}}); err != nil {
		t.Fatal(err)
	}
	if pp, _ := os.ReadFile(filepath.Join(phDir, "app/providers.tsx")); !strings.Contains(string(pp), "PostHogProvider") {
		t.Errorf("posthog feature did not wire PostHogProvider into providers.tsx:\n%s", pp)
	}

	// the email --with feature is additive: it adds resend + react-email (v6) and
	// ships a React Email template + a server-only send helper, overwriting no
	// shared file.
	email, ok := m.Features["email"]
	if !ok || len(email.Add) != 3 {
		t.Errorf("web scaffold missing the email feature with its 3 deps: %+v", m.Features)
	}
	emDir := t.TempDir()
	if _, err := Render(sub, emDir, data); err != nil {
		t.Fatal(err)
	}
	emRootBefore, _ := os.ReadFile(filepath.Join(emDir, "app/root.tsx"))
	if _, err := RenderFeature(sub, email, emDir, data); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"app/emails/welcome.tsx", "app/server/send-email.tsx"} {
		if _, err := os.Stat(filepath.Join(emDir, filepath.FromSlash(p))); err != nil {
			t.Errorf("email feature missing %s: %v", p, err)
		}
	}
	if emRootAfter, _ := os.ReadFile(filepath.Join(emDir, "app/root.tsx")); string(emRootAfter) != string(emRootBefore) {
		t.Error("email feature must not overwrite the shared app/root.tsx (it must stay additive)")
	}

	// a second target must not clobber an existing ci.yml
	if err := os.WriteFile(ciPath, []byte("sentinel"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RenderGitHub(sub, proj, data); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(ciPath); string(b) != "sentinel" {
		t.Error("RenderGitHub clobbered an existing ci.yml")
	}
}
