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

## ⛔ Gate: publish `registry.json`, then templatize

As of 2026-06-14 `github.com/markmals/racket-ui` is **public**, BUT `registry.json` is
**gitignored** there (untracked) so it isn't at the repo root — and shadcn's GitHub
registry *requires* `registry.json` at the root. So `shadcn add markmals/racket-ui/…`
**404s** ("raw.githubusercontent.com did not return a root registry.json file") and the
templatization **cannot merge** yet (every `target add web` would fail). **Fix first, in
the racket-ui repo:** regenerate it (`mise run registry-sync`/`registry-build` if stale),
remove `registry.json` from `.gitignore`, commit + push it (it references the
already-committed `registry/default/*` source). **Verify:** `pnpm dlx shadcn@latest list
markmals/racket-ui` succeeds. (To prototype before that's fixed, serve the local build as
a stand-in: `cd racket-ui/dist && python3 -m http.server 8899`, components.json
`registries: {"@rac-ui": "http://localhost:8899/r/{name}.json"}`, `shadcn add @rac-ui/<item>`.)

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
   `@/lib/cva`, registry `markmals/racket-ui`.
6. **install script (phase 0, after pnpm install):**
   `pnpm dlx shadcn@latest add markmals/racket-ui/base markmals/racket-ui/button … --yes`
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
Verify representative combos green-on-arrival against the real registry once published:
node+none, cloudflare+convex (default), and one feature stacked.
