---
description: The web scaffold's feature/variant mechanics — the provider-composition Wrap seam and the toolchain gotchas that bite when adding a --with feature.
---

# Web scaffold (internal/coreassets/templates/scaffolds/web)

How `--with` features and `--data`/`--runtime` variants compose, and the
prototype-first discipline for adding one. Engine: `internal/scaffold` (see
[[dev-workflow]] for the golden/CI gate).

## Composition model

- **Variants** (`--data`/`--runtime`) and **Features** (`--with`) render *over*
  the base files (whole-file overwrite). Render order: base → runtime → data →
  data-runtime-overlay → **features LAST** (opt-ins win). Feature iteration is a
  Go map → **non-deterministic** order, so two features must never write the same
  file.
- **Provider seam (Mark's call: TanStack Router `Wrap`).** Client-side providers
  do **not** each overwrite `root.tsx`/`router.tsx` (that collides — and `router.tsx`
  is already owned by the convex data variant). Instead the base ships
  `app/providers.tsx` (a Go template) — the single place providers compose, via an
  accumulator gated by `{{if .Features.<name>}}`:
  `let tree = children; {{if ...}}tree = <X>{tree}</X>;{{end}} return <>{tree}</>`.
  Both the base **and** the convex `router.tsx` delegate their `Wrap` to
  `<Providers>`. Adding a Wrap-provider = a `{{if}}` block in `providers.tsx.tmpl`
  (the base enumerates providers) + the feature's `add` deps; the feature usually
  carries **no files** (posthog is the model). `clerk` is orthogonal — it owns
  `root.tsx` (the HTML doc), so it composes with Wrap providers for free.

## Foundation = rac-ui via the shadcn CLI (shadcn-native layout)

The hand-rolled `app/components/foundation/` is **retired**; components come from
**rac-ui** (`markmals/racket-ui`) via the official shadcn CLI — see [[rac-ui-shadcn]].
Structural consequences for the templates:

- **`@/*` → the member root** (tsconfig `paths` + Vite 8 `resolve.tsconfigPaths`; no
  `baseUrl`, no plugin). App code is `@/app/*`; rac-ui is `@/components/ui/*` + `@/lib/cva`
  (root). The old `#/*`/`#db/*` package subpath imports are gone (`#db` → `@/db`).
- shadcn writes to root-relative targets (`components/ui/*`, `lib/cva.ts`,
  `app/globals.css`) at install time — those files are **NOT in the templates**, so
  `web_test.go` must not assert them. `components.json` IS templated (registries map +
  `@/` aliases). fmt/lint stay scoped to `app/` (rac-ui vendored code is typechecked +
  built but not linted).
- `import/extensions` is **`off`** (rac-ui imports are extensionless, TanStack codegen
  carries extensions — both must coexist). `tw-animate-css` stays a base dep (rac-ui's
  globals.css `@import`s it but doesn't declare it); `react-aria-components`/`cva@beta`/
  `tailwind-merge`/Tabler are installed BY shadcn (dropped from the base `pnpm add`).

## Gotchas that cost real time

- **JSX `{{` collides with Go template delims.** A `.tmpl` JSX file can't contain
  inline object props like `options={{ ... }}` — Go reads `{{` as an action. Hoist
  objects to `let` vars and pass single-brace refs (`options={posthogOptions}`).
- **Render must be oxfmt-clean in BOTH modes** (feature on AND off) — the scaffold
  runs no formatter, so green-on-arrival `fmt:check` depends on the emitted bytes.
  Verify by rendering each mode and running `fmt:check`.
- **pnpm 11 build gate.** A dep with a postinstall (e.g. `posthog-js` → `core-js`)
  hard-errors `pnpm add` unless `pnpm-workspace.yaml` has `dangerouslyAllowAllBuilds: true`.
  The base now ships one (was missing only on node+none); variants overwrite it.
- **`prefer-let` lint:** function-local bindings must be `let` (even if never
  reassigned); a non-exported top-level `const` must be `let` or `UPPER_CASE`
  (exported `const` is fine). Match `app/router.tsx`.

## Verifying a new feature (never assert green)

Build the real app in a throwaway `mktemp -d` (NOT the repo; never `cd` in a
`$()` subshell): `specify init demo` → `cd demo` → `target add web --with <feat> …`
→ `fmt:check`/`lint`/`typecheck`/`test`/`build` + `specify verify web`. Verify the
representative combos (a provider feature: ±feature × {none, convex}, the default
cloudflare+convex, and stacked with clerk). Add a render-test block in
`internal/scaffold/web_test.go` (parallel to clerk/tiptap/posthog) for the fast
offline regression.
