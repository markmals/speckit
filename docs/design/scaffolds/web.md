# Web scaffold — tooling preview

**Status:** proposal, grounded in your real product repos (`trove/apps/trove`,
`trove/apps/tangerine-dashboard`) + your stated intent. Supersedes both earlier
drafts (the TanStack-off-metadata one and the Remix-3 overcorrection). For sign-off.

> **What went wrong twice, so it's on record:** I first proposed TanStack off repo
> *names*; then "corrected" to Remix 3 off your *most-pushed* repos — but those are
> you working **on** the Remix framework, not your **product** stack. Reading the
> actual `package.json` of a shipping app settled it.

## The stack (inspected from `trove`)

| layer | choice | from |
| --- | --- | --- |
| **framework** | **TanStack Start** (React 19 + **React Compiler**) — `@tanstack/react-start`, `react-router`, `react-query`/`form`/`table` | both trove apps |
| **toolchain** | **vite-plus** (`defineConfig` from `vite-plus`; `@vitejs/plugin-react` + `@rolldown/plugin-babel` reactCompiler; **lightningcss**) | `apps/trove/vite.config.ts` |
| **styling** | **Tailwind v4** (`@tailwindcss/vite`) + `cva` + `tailwind-merge` + `react-aria-components` | both apps |
| **TS / lint** | **tsgo** (`@typescript/native-preview`) + `eslint-plugin-{perfectionist,prefer-let}` | both apps |
| **data** | **Convex** (hosted) by default; *or* **Drizzle** + `@hey-api/openapi-ts` against an OpenAPI backend (Trove's Go-daemon pattern) | your intent + trove |
| **auth** *(optional)* | **Clerk** (`@clerk/tanstack-react-start`) via `--with clerk` | tangerine-dashboard |
| **runtime** | **Cloudflare Workers** (SSR, prod default — `@cloudflare/vite-plugin` + `wrangler`) · **Node-local** (Trove's daemon-served SPA, `spa: { enabled: true }`) | both apps |
| **testing** | `*.test.ts` via **`vp test`** (vitest under vite-plus) → `junit` | trove tests |

Layout mirrors your apps: `app/` (routes.ts virtual routes → `routes.gen.ts`,
`entry.*`, `components/`, `styles/`, `lib/`).

## Proposed defaults (adjust at will)

- **Runtime:** `--runtime cloudflare` (prod) | `node` (Trove pattern). Default **cloudflare**.
- **Data:** `--data convex` (default, hosted) | `drizzle` (with an OpenAPI backend).
- **Auth:** off; `--with clerk` adds it.

## The binding harness (green on `specify verify web`)

`vp test` configured with a **`junit` reporter** → the `report` path; one example
story under `features/` + a bound `*.test.ts` using **`// [scenario.<id>]`** + a
`// SPEC:` pointer.

```json
"web": {
  "stack": "web",
  "command": "vp test --run",
  "format": "junit",
  "report": "apps/web/junit.xml",
  "source": "apps/web/app"
}
```

## Consequence for the pack

The shipped `web-development` pack (ported from Workbench) is close in spirit
(React/TanStack/Tailwind) but should be refreshed to *this* exact stack
(React Compiler, vite-plus, Convex/Drizzle, Clerk) — a follow-up, not a blocker.

## Method note

Every remaining stack gets the same treatment: read a real shipping repo's
manifest before its preview, not the name/language. (Confirmed by inspection so
far: `remctl` = Swift `.executable` CLI; `Reactivity` = Swift `.library`;
`SafariInjector` = a Theos jailbreak tweak, not an extension.)
