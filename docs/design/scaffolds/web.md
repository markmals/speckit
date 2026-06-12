# Web scaffold — design

**Status:** stack defined by Mark (2026-06-12). Scaffold spec below; for review
before code.

> The **full stack** is captured here as reference context. The **scaffold** wires
> a sensible *default subset* (green on `specify verify` on arrival); the rest is
> reachable via `--with`, `--data`, or is simply documented. "Sites" is a separate
> stack — it becomes the **`website`** scaffold (its own preview later).

---

## The stack (reference)

### Tooling

| concern | choice |
| --- | --- |
| orchestration | **Mise** — env + tasks + toolchain (monorepo: a root config + one per target) |
| dev runtime | **Node** |
| package manager | **pnpm** |
| dev server | **Vite** |
| library bundler | **tsdown** |
| test runner | **Vitest** |
| formatter | **Oxfmt** (web app) · Prettier + `@prettier/plugin-oxc` (Astro `website` only — oxc can't format `.astro`) |
| linter | **Oxlint** (web app) · ESLint + `eslint-plugin-oxlint` (Astro `website` only) |
| type checker | **tsgo** (`@typescript/native-preview`) |
| dev tools | **TanStack DevTools** |

### Apps (this scaffold)

| concern | choice |
| --- | --- |
| components | **React** + **React Compiler** |
| router / framework | **TanStack Router** (virtual file routes) / **TanStack Start** |
| async state | **TanStack Query** |
| local-first | **TanStack DB** |
| tables / forms / hotkeys | **TanStack Table** / **Form** / **Hotkeys** |
| styles | **Tailwind CSS** (component styles: **Tailwind Plus** — paid; see *Components*) |
| unstyled components | **React Aria** |
| animations | **Motion** |
| validation | **Zod** |
| rich text | **TipTap** |
| data | **Convex** (default) · **Drizzle** + `node:sqlite`/**D1** |
| networking | **fetch** |
| logging | **Evlog** |
| desktop | **Electron** |

### Platform (Convex-centric)

| concern | choice |
| --- | --- |
| functions · database · file storage · search | **Convex** |
| cron · queues · realtime | Convex **Scheduling** · **Workpool** · **ProseMirror sync** |
| authentication | **Clerk** |
| email | **Resend** + **React Email** |
| payments | **Stripe** |

### Deployment

| concern | choice |
| --- | --- |
| domains · DNS · CDN · image opt · observability · bot protection · static · edge | **Cloudflare** |
| VPS | **Railway** |

### Project / DevOps

| concern | choice |
| --- | --- |
| agent · IDE | **Claude Code** · **Visual Studio Code** |
| toolchain · tasks · shell env | **Mise** |
| environment variables | **Varlock** |
| CI/CD | **GitHub Actions** |
| error tracking & crash reporting | **Sentry** |
| feature flags · analytics | **PostHog** |

### Sites — the separate `website` scaffold (Astro)

Captured for context; this is the **`website`** stack, scaffolded separately:
**Astro** + React + React Compiler + Tailwind (+ Tailwind Plus) + React Aria,
animations via **View Transitions**, Zod, **Astro** i18n, Drizzle +
`node:sqlite`/D1, **content collections** for markdown, fetch, Evlog. Formatter
**Prettier + `@prettier/plugin-oxc`** and linter **ESLint + `eslint-plugin-oxlint`**
(oxc can't handle `.astro`).

---

## What the web-app scaffold wires

**Default — the green-on-arrival starter:**
- **TanStack Start** + Router (virtual routes) + Query; React 19 + React Compiler; TanStack DevTools.
- **Mise** (monorepo) driving pnpm · Vite · Vitest · tsdown · **Oxfmt** · **Oxlint** · tsgo. (Prettier/ESLint belong to the Astro `website` stack, since oxc can't handle `.astro`.)
- **Tailwind CSS** + **React Aria**; **Motion**; **Zod**.
- **Data:** `--data convex` (default) · `drizzle` (`node:sqlite`/D1).
- **Runtime/deploy:** Cloudflare (default) · Node-local; the SSR/server matrix below.
- **Project glue:** **Varlock** for env vars, a **GitHub Actions** CI workflow (runs the
  Mise `test`/`lint`/`check` tasks), `.vscode` settings.

**Optional `--with`:** `clerk` (auth) · `stripe` (payments) · `email` (Resend + React
Email) · `tiptap` (rich text) · `tanstack-db` (local-first) · `electron` (desktop) ·
`sentry` (errors) · `posthog` (flags + analytics).

**Documented, not wired:** the Convex platform features (scheduling / workpool /
sync — added as a feature needs them) and the Cloudflare/Railway deploy specifics.

## Why Mise (not vite-plus)

Mise lets a single-language app **grow into a polyglot monorepo** without
re-tooling: Node + Go + Swift toolchains side by side, per-platform `fmt`/`lint`
behind one task interface — and it works with **Astro** (which `vp` doesn't). It's
what SpecKit itself uses.

**Monorepo layout** — a root config plus one per target:

```
mise.toml                 # root: shared tools/env + cross-target tasks
apps/web/mise.toml        # the web target's toolchain + tasks + env
…                         # apps/<go-daemon>/mise.toml, apps/<swift>/…
```

**The bare-binary trick** — so tasks call `vite`/`vitest`/`oxlint` directly, no `npx`:

```toml
[env]
_.path = ['{{config_root}}/node_modules/.bin']
```

Tasks: `dev` (vite) · `test` (vitest → junit) · `build` (vite; tsdown for libs) ·
`fmt` (oxfmt) · `lint` (oxlint) · `check` (tsgo) · Drizzle db tasks.

## Folder layout & routing

- **`app/`** is the source dir — set in the TanStack Start plugin
  (`tanstackStart({ srcDirectory: "app" })`), not the default `src/`.
- **Virtual file routes:** `app/routes.ts` declares the tree → `routes.gen.ts`
  (TanStack Router's virtual-file-routes), not filesystem routing.

## SSR / server matrix

| `--ssr` | `--server` | mode | what it means / host |
| --- | --- | --- | --- |
| on | on | **SSR app** | server-renders JSX **and** runs middleware / server functions. CF Workers / Node. |
| off | on | **SPA + server** | client-renders (no SSR) **but keeps** middleware + server functions. CF Workers / Node. |
| off | off | **static SPA** | no server — a static host (CF Pages), or served by a non-JS daemon via `go:embed` (**Trove's pattern**). |

`--ssr --no-server` is rejected. Defaults: **`--ssr --server`**, runtime
**cloudflare**, **`--data convex`**.

## Components (no bundled library)

The app stack's component styles are **Tailwind Plus** (paid; the closed-source
"Catalyst" set `trove`'s `foundation` adapts). **It can't ship in this MIT repo.**
So the scaffold:

- ships **no** component library, and **encourages users to build their own** by
  styling **React Aria Components** with Tailwind (the `foundation` recipe:
  `react-aria-components` + `cva` + `motion`);
- **Future direction:** a SpecKit **shadcn registry** serving shadcn-style
  components that use **React Aria under the hood instead of Radix**, so users can
  `shadcn add` them. (Not in the first slice.)

## The binding harness (green on `specify verify web`)

The Mise `test` task runs Vitest with a **`junit` reporter** → the `report` path;
one example story under `features/` + a bound `*.test.ts` with **`// [scenario.<id>]`**
and a `// SPEC:` pointer.

```json
"web": {
  "stack": "web",
  "command": "mise run -C apps/web test",
  "format": "junit",
  "report": "apps/web/junit.xml",
  "source": "apps/web/app"
}
```

(exact mise monorepo task invocation pinned at build.)

## Defaults

`--ssr --server`, runtime **cloudflare**, **`--data convex`**, auth off; Tailwind +
React Aria + Motion + Zod + the TanStack kit + the full tool chain wired; **no**
bundled components; platform/email/payments/rich-text via `--with`.

## References (inspected)

`trove/apps/trove` · `trove/apps/tangerine-dashboard` (Clerk + Cloudflare) ·
`tanstack-react-start-contacts` · trove `main` (Drizzle) vs `convex` branch.

## Method note

Same inspect-a-shipping-repo pass precedes every other stack's preview.
