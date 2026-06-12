# Node CLI scaffold — design

**Status:** stack defined by Mark (2026-06-12); for review.

> A `node-cli` target shares the **Node toolchain** with `web` (Mise, tsdown,
> Vitest, oxc, Drizzle, …); the CLI-specific pieces are the **Bombshell** UX kit,
> **TS-Rest**, **plainjob**, and **single-file-executable** distribution.

## The stack (reference)

| concern | choice |
| --- | --- |
| single-file executables | **Node** (`node:sea`) |
| bundler | **tsdown** |
| argument parser | **Bombshell Args** (`@bomb.sh/args`) |
| prompts | **Bombshell Clack** |
| completions | **Bombshell Tab** |
| server · RPC · OpenAPI | **TS-Rest** |
| database | **Drizzle** + `node:sqlite` |
| networking | **fetch** |
| logging | **Evlog** |
| background jobs | **plainjob** |
| distribution | **Homebrew** · **Mise** · **apt** · **winget** |

## Shared with `web` (the Node toolchain)

**Mise** (env + tasks + toolchain, monorepo) · **pnpm** · **tsdown** · **Vitest** ·
**Oxfmt/Prettier** · **Oxlint/ESLint** · **tsgo** · `node:sqlite` + Drizzle ·
fetch · **Evlog** · **Varlock** · **GitHub Actions**. (Same Mise monorepo + the
`_.path = node_modules/.bin` trick as `web`.)

## What the scaffold wires

**Default — green-on-arrival starter:**
- A tsdown-bundled Node CLI with **Bombshell Args** + an example command, the Mise
  tool chain, **Vitest**, a **SEA** build task, and a **GitHub Actions** release workflow.

**Optional `--with`:** `prompts` (Bombshell Clack) · `completions` (Bombshell Tab) ·
`ts-rest` (server / RPC / OpenAPI) · `drizzle` (`node:sqlite`) · `jobs` (plainjob) ·
`dist` (the Homebrew / Mise / apt / winget release wiring).

## Distribution

The CLI builds a **single executable** (`node:sea`) and ships through Homebrew,
Mise, apt, and winget — the **same channels SpecKit itself uses** (a goreleaser-style
release + a Homebrew tap formula + a Mise plugin). The `--with dist` feature can
scaffold the release workflow + a formula template, mirroring `packaging/` here.

## Layout

`bin/` (the executable entry) · `src/` (commands, lib) · `src/**/*.test.ts`
(Vitest). tsdown → `dist/`; SEA wraps it into one binary.

## Product kind

`node-cli` → **`kind: app`** by default (the CLI's user is a human actor, so
`writing-user-stories` applies as-is), overridable to `library` for an SDK-shaped
CLI. See [library-products.md](library-products.md).

## Binding harness (green on `specify verify <name>`)

Vitest with a **`junit` reporter** → the `report` path; an example story under
`features/` + a bound `*.test.ts` with **`// [scenario.<id>]`** + a `// SPEC:` pointer.

```json
"cli": {
  "stack": "node-cli",
  "command": "mise run -C apps/cli test",
  "format": "junit",
  "report": "apps/cli/junit.xml",
  "source": "apps/cli/src"
}
```

## Method note

Before building, inspect a real node CLI repo (`downpour-js`, `create-sprinkles` —
adapted off vite-plus) to ground the `bin/`/`src/` layout + the Bombshell + SEA
wiring.
