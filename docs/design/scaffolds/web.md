# Web scaffold — tooling preview

**Status:** ✅ approved (2026-06-12) with the corrections below baked in. Grounded
in `trove/apps/trove` + `tanstack-react-start-contacts`.

## The stack

| layer | choice |
| --- | --- |
| **framework** | **TanStack Start** (React 19 + **React Compiler**) — `react-start`, `react-router` with **virtual file routes**, `react-query`/`form`/`table` |
| **tooling** | **Mise** for env + tasks + toolchain, driving **raw** tools — **Vite**, **Vitest**, **Oxfmt**, **Oxlint**, **tsdown** (libs). **No vite-plus / `vp`.** |
| **styling** | **Tailwind v4** (`@tailwindcss/vite`, lightningcss) + `cva` |
| **components** | **Not bundled** — see *Components* below (Foundation is paid/closed-source). The scaffold encourages Tailwind-styled **React Aria** components. |
| **data** | **`--data drizzle`** (trove `main`) · **`--data convex`** (hosted; trove `convex` branch) |
| **auth** *(optional)* | **Clerk** (`@clerk/tanstack-react-start`) via `--with clerk` |
| **TS** | **tsgo** (`@typescript/native-preview`) |
| **testing** | **Vitest** (raw) → `junit`, run via the Mise `test` task |

## Why Mise (not vite-plus)

Mise lets a single-language app **grow into a polyglot monorepo** without
re-tooling: Node + Go + Swift toolchains side by side, per-platform `fmt`/`lint`
commands (oxfmt for TS, gofmt for Go, swift-format for Swift) behind one task
interface — and it works with **Astro** (which `vp` doesn't). It's also what
SpecKit itself uses.

**Monorepo layout** — a root config plus one per target:

```
mise.toml                 # root: shared tools/env + cross-target tasks
apps/web/mise.toml        # the web target's toolchain + tasks + env
…                         # apps/<go-daemon>/mise.toml, apps/<swift>/…, etc.
```

**The bare-binary trick** — so tasks call `vite`/`vitest`/`oxlint` directly, no `npx`:

```toml
[env]
_.path = ['{{config_root}}/node_modules/.bin']
```

Tasks: `dev` (vite) · `test` (vitest → junit) · `build` (vite; tsdown for lib
output) · `fmt` (oxfmt) · `lint` (oxlint) · Drizzle's db tasks.

## Folder layout & routing

- **`app/`** is the source dir — customized in the TanStack Start plugin
  (`tanstackStart({ srcDirectory: "app" })`), not the default `src/`.
- **Virtual file routes:** `app/routes.ts` declares the route tree →
  `routes.gen.ts` (TanStack Router's virtual-file-routes), not filesystem routing.

## SSR / server matrix

Two flags, three valid modes:

| `--ssr` | `--server` | mode | what it means / host |
| --- | --- | --- | --- |
| on | on | **SSR app** | server-renders JSX **and** runs middleware / server functions. CF Workers / Node. |
| off | on | **SPA + server** | client-renders the JSX (no SSR) **but keeps** middleware + server functions. CF Workers / Node. |
| off | off | **static SPA** | no server at all — a static host (CF Pages), or served by a non-JS daemon via `go:embed` (**Trove's pattern**). |

`--ssr --no-server` is rejected (SSR needs a server). Defaults: **`--ssr --server`**,
runtime **cloudflare**, **`--data convex`**.

## Components (no bundled library)

The `foundation` set in `trove` is **adapted from the paid, closed-source Catalyst
components — it can't ship in this MIT repo.** Instead the scaffold:

- ships **no** component library, and **encourages users to build their own** by
  styling **React Aria Components** with Tailwind (the same recipe `foundation`
  uses: `react-aria-components` + `cva` + `motion`);
- **Future direction:** a SpecKit **shadcn registry** that serves shadcn-style
  components which use **React Aria Components under the hood instead of Radix**,
  so users can `shadcn add` them. (Not in the first slice.)

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

## Defaults

`--ssr --server`, runtime **cloudflare**, **`--data convex`**, auth off; Tailwind +
React Aria wired, **no** bundled components.

## References (inspected)

`trove/apps/trove` (TanStack Start + Drizzle, static SPA via daemon) ·
`trove/apps/tangerine-dashboard` (SSR + Cloudflare + Clerk) ·
`tanstack-react-start-contacts` (SSR-on-Cloudflare config) ·
`trove/app/components/foundation` (the Catalyst-derived set — *pattern* only, not copied).

## Method note

Same inspect-a-shipping-repo pass precedes every other stack's preview.
