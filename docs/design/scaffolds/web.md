# Web scaffold — tooling preview

**Status:** ⚠️ **Revised 2026-06-12 after inspecting the actual repos.** The earlier
"TanStack Start" approval was based on a shallow read (repo names + primary
language) and is **superseded**. The real, actively-developed web stack —
confirmed from `remix-3-templates` and `remix-3-contacts` source — is **Remix 3
on your own toolchain**. **Re-approval needed.**

## Real stack (inspected, not guessed)

| layer | what your repos actually use | evidence |
| --- | --- | --- |
| **framework** | **Remix 3** (`remix@3.0.0-beta.2`) — *not* React/TanStack (no `react` dep at all; Remix 3 is its own framework) | `remix-3-templates`, `remix-3-contacts`, 4× `remix-frame-*`, all pushed this month |
| **toolchain** | **vite-plus** — `vite` is aliased to `@voidzero-dev/vite-plus-core`; `@hiogawa/vite-plugin-fullstack` | `remix-3-templates/bun/package.json` |
| **runtime** | **Bun** and/or **Cloudflare Workers** (`wrangler`, `@cloudflare/vite-plugin`) | `bun/` + `cloudflare/` template variants |
| **TS / lint** | **tsgo** (`@typescript/native-preview`) + **oxlint** (`oxlint-tsgolint`, `oxc-parser`) | dev deps |
| **styling** | custom CSS (`app/styles/preflight.css`) — **not Tailwind** | template tree |
| **testing** | `remix-test.config.ts` + `*.test.ts` (vite-plus/vitest-compatible) + **Playwright** e2e | `remix-3-contacts` |
| **data** | SQL migrations (`db/migrations/*`) | `remix-3-templates/bun/db` |
| **layout** | `app/` (actions, components, data, routes.ts, entry.{browser,server}, middleware) | template tree |

You're also already doing SDD here — `remix-3-contacts` ships `.claude/skills/remix/`
and `docs/superpowers/specs/`.

## Implication: mirror your own template

SpecKit's `web` scaffold should **be your `remix-3-templates`** (the canonical
Remix 3 + vite-plus + Bun/Cloudflare setup you already maintain) with the SpecKit
**binding harness** layered on — not an invented TanStack project. And the
shipped `web` **pack** (`web-development`, which I ported from Workbench as
React/TanStack/Tailwind) is **wrong for you** → it should be rewritten to Remix 3
(you already have `.claude/skills/remix/` as the source of truth).

## The binding harness (makes `specify verify web` green on arrival)

- The test runner (vite-plus / `remix-test.config.ts`) configured with a
  **`junit` reporter** → the `report` path.
- One example story under `features/` + a bound `*.test.ts` using
  **`// [scenario.<id>]`** above the test, and a `// SPEC:` pointer on the unit.

```json
"web": {
  "stack": "web",
  "command": "vp test --run",
  "format": "junit",
  "report": "apps/web/junit.xml",
  "source": "apps/web/app"
}
```

(`command`/`report` flags pinned once I confirm vp test's junit output at build.)

## Your calls (re-approval)

1. **Runtime default** — Bun · Cloudflare Workers · ask each time. (Your templates ship both.)
2. **Mirror `remix-3-templates` as-is**, or a trimmed variant for scaffolding?
3. **Styling** — keep custom CSS (your default), or offer Tailwind as `--with`?
4. **Pack** — rewrite `web-development` to Remix 3 now, or point it at your existing `.claude/skills/remix/`?

## Method note

This correction came from reading the actual `package.json` + file trees. The
same **inspect-don't-guess** pass should precede every stack's preview (e.g. the
iOS apps, the TS libs, the VS Code extensions) before we build them.

Sources (mid-2026 currency): [Remix blog](https://remix.run/blog) ·
[Vite+ / VoidZero](https://voidzero.dev) — exact versions pinned at build time.
