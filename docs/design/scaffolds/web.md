# Web scaffold — tooling preview

**Status:** revised per detailed direction (2026-06-12), grounded in
`trove/apps/trove` + `tanstack-react-start-contacts`. For sign-off.

## The stack

Copy `trove/apps/trove`'s TanStack setup, with two deliberate changes: **Mise**
replaces `vite-plus` for tooling, and the **`foundation`** component library is
baked in.

| layer | choice |
| --- | --- |
| **framework** | **TanStack Start** (React 19 + **React Compiler**) — `react-start`, `react-router` (file routes), `react-query`/`form`/`table` |
| **tooling** | **Mise** for env + tasks + toolchain (node version), driving **raw** tools — **Vite**, **Vitest**, **Oxfmt**, **Oxlint**, **tsdown** (libs). **No vite-plus / `vp`.** |
| **styling** | **Tailwind v4** (`@tailwindcss/vite`, lightningcss) + `cva` (the `cx` helper) |
| **components** | **`foundation`** — the Catalyst-style set on **React Aria** (`react-aria-components`) + Tailwind + `motion` (badge, button, calendar, input, navbar, select, table, …). Shared across all web apps. |
| **data** | **`--data drizzle`** (`drizzle-orm` + `drizzle-kit`; trove's `main`) · **`--data convex`** (hosted; trove's `convex` branch) |
| **auth** *(optional)* | **Clerk** (`@clerk/tanstack-react-start`) via `--with clerk` |
| **TS** | **tsgo** (`@typescript/native-preview`) |
| **testing** | **Vitest** (raw) → `junit`, run via the Mise `test` task |

## SSR / server matrix (the new affordance)

TanStack Start can render server-side or not, and can ship with a server or not.
Two flags, three valid modes:

| `--ssr` | `--server` | mode | host |
| --- | --- | --- | --- |
| on | on | **SSR app** (server-rendered) | Cloudflare Workers / Node |
| off | on | **SPA + server** (client-rendered, server owns data/API — Trove's daemon pattern, `spa:{enabled:true}`) | Workers / a daemon |
| off | off | **Static SPA** (no server, no SSR) | Cloudflare Pages / any static host |

`--ssr --no-server` is rejected (SSR needs a server). Defaults: **`--ssr --server`**
(production), with **Cloudflare Workers** the default runtime. Static SPA pairs
naturally with `--data convex` (hosted backend, no server of your own).

## Tooling via Mise (replaces `vp` tasks)

The scaffold's `mise.toml` carries the toolchain + tasks; `package.json` carries
the deps. Tasks: `dev` (vite) · `test` (vitest → junit) · `build` (vite; tsdown
for any lib output) · `fmt` (oxfmt) · `lint` (oxlint) · the db tasks for Drizzle.

## The binding harness (green on `specify verify web`)

The `test` task runs Vitest with a **`junit` reporter** → the `report` path; one
example story under `features/` + a bound `*.test.ts` with **`// [scenario.<id>]`**
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

## Defaults (adjust at will)

`--ssr --server`, runtime **cloudflare**, **`--data convex`**, auth off, `foundation`
+ Tailwind + React Aria always in.

## References (inspected)

`trove/apps/trove` (TanStack Start + foundation + Drizzle, SPA+daemon) ·
`trove/apps/tangerine-dashboard` (SSR + Cloudflare + Clerk) ·
`tanstack-react-start-contacts` (SSR-on-Cloudflare config) · trove `main` (Drizzle)
vs `convex` branch (Convex).

## Method note

Same inspect-a-shipping-repo pass precedes every other stack's preview.
