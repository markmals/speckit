---
description: The proven recipe to replace the web scaffold's hand-rolled foundation with rac-ui via the official shadcn CLI — gated on publishing rac-ui to GitHub.
---

# rac-ui via the shadcn CLI (web foundation replacement)

**Goal (Mark's call):** retire the web scaffold's ad-hoc `foundation/` (React-Aria +
cva recipe) and instead consume **rac-ui** — a shadcn/ui-compatible registry rebuilt
on React Aria Components + Tailwind v4 + cva + Tabler — via the **official shadcn
CLI**. rac-ui lives at `~/Developer/Libraries/rac-ui` (registry items under
`registry/default/*`, built to `dist/r/*.json`); its GitHub remote is
**`github.com/markmals/rac-ui`**.

## ⛔ Gate: rac-ui must be published first

Decided: **Mark publishes `github.com/markmals/rac-ui` (public) first**, THEN this is
templatized + verified end-to-end against the real GitHub registry + merged. As of
2026-06-13 the public registry (`rac-ui.malstrom.me`) is **down** and the GitHub repo
isn't public — so the change **cannot merge** (every `target add web` would fail) and
can only be verified against a **local stand-in** (`cd rac-ui/dist && python3 -m
http.server 8899`, components.json `registries: {"@rac-ui": "http://localhost:8899/r/{name}.json"}`).
Shadcn GitHub-registry form: `shadcn add markmals/rac-ui/<item>`.

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
   `@/lib/cva`, registry `markmals/rac-ui`.
6. **install script (phase 0, after pnpm install):**
   `pnpm dlx shadcn@latest add markmals/rac-ui/base markmals/rac-ui/button … --yes`
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
