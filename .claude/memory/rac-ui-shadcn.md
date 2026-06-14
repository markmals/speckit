---
description: The proven recipe to replace the web scaffold's hand-rolled foundation with racket-ui (markmals/racket-ui) via the official shadcn CLI — gated on committing registry.json to that repo's root.
---

# rac-ui via the shadcn CLI (web foundation replacement)

**Goal (Mark's call):** retire the web scaffold's ad-hoc `foundation/` (React-Aria +
cva recipe) and instead consume **racket-ui** — a shadcn/ui-compatible registry rebuilt
on React Aria Components + Tailwind v4 + cva + Tabler — via the **official shadcn
CLI**. Local dir: `~/Developer/Libraries/racket-ui` (registry items under
`registry/default/*`); GitHub registry: **`github.com/markmals/racket-ui`** → the shadcn
address is **`markmals/racket-ui/<item>`** (NOT `markmals/racket-ui` — published under the
`racket-ui` name).

## ✅ Gate RESOLVED (2026-06-14) — the real fix was the per-item distribution

The handoff said "commit `registry.json`" — that was **necessary but insufficient**.
The aggregate `registry.json` at the repo root makes the GitHub *shorthand*
(`markmals/racket-ui/<item>`) + `shadcn list` work, but **every rac-ui item's
`registryDependencies` is namespaced `@racket-ui/cva`** — and shadcn resolves a
namespaced dep through the consumer's `components.json` **`registries` map**, which
needs a **per-item** URL template (`…/{name}.json`). racket-ui's per-item distribution
(`public/r/*.json`, built by `mise run registry:build` → `shadcn build`) was
**gitignored**, so `@racket-ui/cva` 404'd and every `base`/`button` add failed
("Unknown registry @racket-ui" / "…cva.json was not found"). **Fix shipped in
[racket-ui#1](https://github.com/markmals/racket-ui/pull/1) (merged):** regenerate +
**commit `public/r/`** and un-gitignore it. The consumer then sets
`registries: {"@racket-ui": "https://raw.githubusercontent.com/markmals/racket-ui/main/public/r/{name}.json"}`
and runs `shadcn add @racket-ui/base @racket-ui/button` (namespaced — fetches raw JSON,
**no `git ls-remote`**, so it also dodges shadcn's flaky 15s cold-start timeout).
**Templatized + verified green-on-arrival** across node+none, cf+convex (default),
cf+drizzle, node+drizzle, and node+all-5-features (shipped in speckit; see [[web-scaffold]]).
(Local stand-in for re-prototyping: `cd racket-ui && python3 -m http.server 8899`,
`registries: {"@racket-ui": "http://localhost:8899/public/r/{name}.json"}`.)

## The proven recipe (a real scaffold restructured + installed → fully green)

shadcn writes each registry item to **its root-relative `target`** (rac-ui →
`components/ui/*`, `lib/cva.ts`, `app/globals.css`); aliases only rewrite imports, not
write paths. So the scaffold must move to rac-ui's shadcn-native shape: **`@/*` → the
member root** (components/ui + lib at root; `app/` keeps routes/entry/globals).

1. **tsconfig:** `compilerOptions.paths = {"@/*": ["./*"]}`. **No `baseUrl`** — tsgo /
   @typescript/native-preview removed it (TS5102); paths are tsconfig-dir-relative.
2. **vite.config (BOTH runtime variants):** `resolve: { tsconfigPaths: true }` — Vite 8
   native tsconfig-paths resolution (no manual alias).
3. **package.json:** drop the `#/*` subpath import (migrate `#db/*` → `@/db/*` too).
4. **.oxlintrc.json:** `"import/extensions": "off"` — rac-ui components are extensionless
   (`@/lib/cva`) but TanStack codegen imports carry extensions (`./routes.gen.ts`); both
   must coexist, so the rule can't be `always` or `never`.
5. **components.json (new):** `style: new-york`, `tailwind.css: app/globals.css`,
   `iconLibrary: tabler`, aliases `@/components` / `@/lib` / ui `@/components/ui` / utils
   `@/lib/cva`, and the **`registries` map** `"@racket-ui":
   "https://raw.githubusercontent.com/markmals/racket-ui/main/public/r/{name}.json"`.
6. **install script (phase 0, after pnpm install):**
   `pnpm dlx shadcn@latest add @racket-ui/base @racket-ui/button --yes` (namespaced — base
   pulls in `cva` + globals.css + deps; button pulls in `cva` + `components/ui/button.tsx`).
   (`@rac-ui/base` brings globals.css + deps @tabler/icons-react, cva@beta,
   react-aria-components, tailwind-merge + the `lib/cva.ts` helper — identical to the old
   `styles/cva.ts`).
7. **remove:** `app/components/foundation/`, `app/styles/cva.ts`, `app/styles/tailwind.css`.
8. **app files:** `root.tsx` css → `@/app/globals.css?url`; `router.tsx` → `@/app/providers.tsx`;
   `home.tsx` → rac-ui `Button` from `@/components/ui/button` + Tabler `IconRocket` +
   `@/app/lib/greeting.ts`. App code is `@/app/*`; rac-ui is `@/components`/`@/lib` (root).
9. **deps:** drop react-aria-components / cva@beta / tailwind-merge / **lucide-react** from
   the base `pnpm add` (shadcn installs the first three + Tabler via rac-ui). **Keep
   tw-animate-css** — rac-ui's globals.css `@import`s it but doesn't declare it.

## Full templatization scope (atomic — all combos must stay green)

Base + **both runtimes** (vite.config) + **convex/drizzle data variants** (their
`router.tsx` / `data/*` / `#db`) + **clerk/tiptap/email features** (all reference `#/` —
e.g. tiptap `editor.tsx` `cx` → `@/lib/cva`, clerk/email css/imports) + `web_test.go`.
Verified green-on-arrival against the real registry (racket-ui#1 merged) across the full
matrix: `{cloudflare,node}` × `{convex,drizzle,none}` + all 5 features stacked.
