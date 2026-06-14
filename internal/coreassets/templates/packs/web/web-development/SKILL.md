---
name: web-development
description: Use when writing or modifying web target code. Covers React + TanStack Start/Router/Query/DB/Form + Convex + Clerk + Tailwind v4 + rac-ui (React Aria via shadcn) + Motion + Zod idioms, and points at /llms.txt endpoints for first-party docs. Complementary to `implementing-a-spec` (process) and `web-verification` (visual verification loop).
---

# Web Development

This skill covers **how to write web-app code** for the web target. For the _workflow_ of implementing a spec, see `implementing-a-spec`. For the _verify-iterate loop_ in a browser, see `web-verification`. For _what to build_, see the spec. For the marketing/content site, see `website-development` (Astro, not TanStack Start).

The web target is the **reference implementation**. Build features here first; every other target uses the web realization as a worked example.

> **Backend mode.** This skill documents the **Convex** backend (the template default). If your project chose `OpenAPI` mode at `/speckit.implement`, the data layer swaps the Convex client for the TS-Rest typed client; if `No API`, for local Drizzle / TanStack DB. The view-model and component idioms below are identical regardless of mode — only the Client layer changes. See `specs/ARCHITECTURE.md` → "Backend modes".

## Stack at a glance

| Concern              | Choice                                     | First-party docs                                                               |
| -------------------- | ------------------------------------------ | ------------------------------------------------------------------------------ |
| Framework            | TanStack Start                             | [llms.txt](https://tanstack.com/llms.txt)                                      |
| Components / React   | React + React Compiler (optimizer)         | [react.dev/llms.txt](https://react.dev/llms.txt)                               |
| Router               | TanStack Router (via Start)                | [llms.txt](https://tanstack.com/llms.txt)                                      |
| Server state         | TanStack Query + `@convex-dev/react-query` | (covered by Convex docs below)                                                 |
| Local-first store    | TanStack DB                                | [llms.txt](https://tanstack.com/llms.txt)                                      |
| Tables / Forms       | TanStack Table · TanStack Form             | [llms.txt](https://tanstack.com/llms.txt)                                      |
| Hotkeys              | TanStack Hotkeys                           | [llms.txt](https://tanstack.com/llms.txt)                                      |
| Backend              | Convex                                     | [llms.txt](https://docs.convex.dev/llms.txt)                                   |
| Auth                 | Clerk                                      | [llms.txt](https://clerk.com/docs/llms.txt)                                    |
| UI components        | rac-ui (React Aria via the shadcn CLI)     | [racket-ui](https://github.com/markmals/racket-ui) · [react-spectrum](https://react-spectrum.adobe.com/llms.txt) |
| Styling              | Tailwind v4 + cva (rac-ui)                 | [tailwindcss.com/docs](https://tailwindcss.com/docs) _(no /llms.txt yet)_      |
| Animation            | Motion                                     | [motion.dev/docs](https://motion.dev/docs)                                     |
| Validation           | Zod                                        | [zod.dev](https://zod.dev/)                                                    |
| Relational / edge DB | Drizzle (`node:sqlite` or Cloudflare D1)   | [orm.drizzle.team](https://orm.drizzle.team/docs)                              |
| Logging              | Evlog                                      | [evlog.dev](https://www.evlog.dev/)                                            |
| Build tool           | Vite                                       | [llms.txt](https://vitejs.dev/llms.txt)                                        |
| Library bundler      | tsdown (only for shared libs / exes)       | [tsdown.dev](https://tsdown.dev/)                                              |
| Tests                | Vitest                                     | [llms.txt](https://vitest.dev/llms.txt)                                        |
| Linter / formatter   | Oxlint + Oxfmt                             | [llms.txt](https://oxc.rs/llms.txt)                                            |
| Type checker         | tsgo (`@typescript/native-preview`)        | [tsdown.dev](https://tsdown.dev/)                                              |
| Dev tools            | TanStack DevTools                          | [tanstack.com/devtools](https://tanstack.com/devtools/latest)                  |
| Package manager      | pnpm                                       | [pnpm.io](https://pnpm.io/)                                                    |
| Production runtime   | Cloudflare Workers                         | [developers.cloudflare.com/workers](https://developers.cloudflare.com/workers) |
| Desktop packaging    | Electron (wraps this same app)             | [electronjs.org/docs](https://www.electronjs.org/docs/latest)                  |

When you need to look something up: fetch the relevant `/llms.txt` with WebFetch and let it route you to the specific page. The `/llms.txt` is the index, not the content — it tells you which URLs to fetch next.

## Idioms (read these before writing code)

### View models are query options + helpers, not classes

A view model in this stack is rarely a class. It's typically:

- A `queryOptions` factory exposing the read state
- A handful of selectors (pure functions of the query data)
- One or more mutation wrappers exposing user actions

```ts
// SPEC: vm.items.list
import { queryOptions } from "@tanstack/react-query";
import { convexQuery } from "@convex-dev/react-query";
import { api } from "convex/_generated/api";

export const itemsListQueryOptions = queryOptions(convexQuery(api.items.list, {}));

export function selectItemCount(items: Item[] | undefined): number {
    return items?.length ?? 0;
}
```

Tests target the queryOptions key and the selectors — not the rendered DOM. Behavior tests live at the view-model layer; component tests are usually limited to wiring smoke tests.

The Vitest binding affordance: place a `// [scenario.<sub-id>]` comment directly above each `it(...)` block to declare the scenario join, keeping the test title clean. See `specs/CONVENTIONS.md` for the full binding table and scenario sub-ID conventions.

### Components stay dumb

Components consume view models and render. They do **not** call mutations directly, do **not** read from the Convex client directly, do **not** contain branching business logic.

```tsx
// SPEC: manual (this component renders vm.items.list)
import { useSuspenseQuery } from "@tanstack/react-query";
import { itemsListQueryOptions } from "./items.list.vm";

export function ItemsList() {
    const { data: items } = useSuspenseQuery(itemsListQueryOptions);
    if (items.length === 0) return <EmptyState />;
    return (
        <ul>
            {items.map((c) => (
                <ItemRow key={c._id} item={c} />
            ))}
        </ul>
    );
}
```

The reverse pointer lives on the **view model**, not on the component. Components carry `// SPEC: manual` because they have no cross-platform behavioral contract.

### Server functions are the call site for non-data work

Anything that isn't a Convex query/mutation but needs to run server-side goes in a TanStack Start server function (`createServerFn`). Sending an email, calling a third-party API, computing a sensitive value — server function. Don't put server logic in route components.

### No global state library

TanStack Query + Convex's reactive cache _is_ the server state. Don't reach for Redux, Zustand, Jotai, or Pinia. Local UI state (open/closed, hover, transient input) stays in components via `useState`. For genuinely **local-first** views — optimistic, offline-tolerant, or derived across queries — use **TanStack DB** collections backed by the Convex query, not a bespoke cache. Most reads stay plain `convexQuery`; reach for TanStack DB only when the view actually needs it.

### Auth is Clerk

Identity, sessions, and the signed-in user come from **Clerk** — not hand-rolled auth state. Gate routes with Clerk, read the user from Clerk's hooks/components, and let Convex validate the Clerk-issued identity server-side. There is no separate user table you own for authentication.

### Forms: TanStack Form + Zod

**TanStack Form** owns form state; **Zod** owns the schema. Define the Zod schema once and share it between the form's validators and the Convex mutation's argument validation, so client and server can't disagree about what's valid. Don't hand-roll `onChange` validation.

### Animation is Motion

Use **Motion** for transitions and micro-interactions. Honor `prefers-reduced-motion`, and match the `motion.*` duration/easing tokens in `DESIGN_SYSTEM.md` rather than inventing per-component timings.

### Rich text is TipTap

Rich-text editing (comments, descriptions, document bodies) uses **TipTap**. Keep the editor a dumb component that emits structured content up to a view model; don't scatter editor commands through business logic. Persist the document model, not rendered HTML, and share the schema with the backend the same way forms share their Zod schema.

### Logging is Evlog

Structured logging goes through **Evlog**, not `console.log`, in anything that ships. `console.*` is fine for throwaway local debugging — remove it before committing.

### Tailwind v4 conventions

- Use the `@theme` directive to declare design tokens; reference them via `var(--color-...)` in components when you need to bind dynamically.
- Prefer utility classes over `@apply`. Reach for `@apply` only for repeated patterns that are genuinely component-level (focus rings, button bases).
- Tailwind v4 reads the config from CSS, not a JS config file. Live with it — don't shim a v3-style `tailwind.config.js`.

### UI components come from rac-ui (React Aria under the hood)

The scaffold installs [rac-ui](https://github.com/markmals/racket-ui) — a shadcn/ui-compatible registry built on **React Aria Components** + Tailwind v4 + cva + Tabler — via the shadcn CLI. Add a component with `pnpm dlx shadcn@latest add @racket-ui/<name>` (e.g. `card`, `dialog`, `input`); it lands in `components/ui/` (member root) and imports the `cva`/`cx` helpers from `@/lib/cva`. Restyle and compose the generated components in place — don't hand-roll `<button>` + manual ARIA (rac-ui/React Aria handles focus, keyboard, and screen-reader semantics), and don't vendor a paid set (Tailwind Plus / Catalyst).

### Oxlint and Oxfmt

These are the linter and formatter. Don't ship ESLint or Prettier config files. If a rule is too strict, change the rule in `.oxlintrc.json` — don't disable it inline unless you have a real reason.

## Convex specifics

- **Schema is the protocol.** Every entity lives in `services/convex/schema.ts` with explicit indexes. Add `// SPEC: domain.<entity>` to each table block.
- **Functions are thin.** Queries return the precise shape the client needs; mutations validate inputs and return the affected entity. Heavy logic lives in `services/convex/lib/`.
- **Use actions for I/O only.** Anything calling an external API goes in an action, not a mutation.
- **Identity comes from Clerk.** Convex functions read the authenticated identity via the Clerk integration (`ctx.auth.getUserIdentity()`), not a user table you manage for authentication.
- **Convex is the primary store; Drizzle is the exception.** Reach for Drizzle (e.g. Cloudflare D1) only for data that genuinely belongs at the edge or in a relational shape Convex doesn't fit. Default to Convex.
- **Run the Convex codegen command after schema changes**, then run the target's typecheck command (tsgo) to surface breaks.

## TanStack Start specifics

- **File-based routing.** Route files under the target's `src/routes/` map to URL paths. The conventions are documented at the TanStack Start docs (see /llms.txt above).
- **Data loaders go in `loader`.** Route-scoped data fetching uses the loader API — not `useEffect`. The loader runs on the server during SSR.
- **Cloudflare Workers as the deploy target.** Avoid Node-only APIs (`fs`, `child_process`, etc.) in code that runs in the worker. Use Web Standard APIs (`fetch`, `Request`, `Response`, `crypto.subtle`).

## File layout (within the target's source)

The target's configured `source` dir contains:

```
src/
├── routes/             ← file-based routes
├── features/<slug>/    ← feature-scoped: <view>.vm.ts, <view>.vm.test.ts, components
├── components/         ← shared dumb components
├── domain/             ← domain types (one file per domain.* spec)
├── client/             ← Convex client wrapper (rarely edited)
└── styles/             ← Tailwind layers and tokens
```

## When to invoke a more specific skill

- About to write tests? → `test-driven-development`
- About to claim work is done? → `verification-before-completion`
- Verifying visually in a browser? → `web-verification`
- Debugging something unexpected? → `systematic-debugging`
- Implementing a spec end-to-end? → `implementing-a-spec` (this skill supports that workflow with idiom knowledge)

## Commit

Land focused, atomic commits as the work hits natural boundaries — typically per spec ID, per view-model + its tests, or per cohesive refactor. Follow the scoped-commits convention (`specify gate scope`).

Web-specific notes:

- **Schema changes go alone.** A Convex schema edit (with its codegen output) belongs in its own commit — `feat: add <table> schema`. Don't bundle with view-model code that consumes it.
- **Generated files (`_generated/`) ride with the schema commit** that produced them; never commit them out of sync.
- **Don't bundle dependency bumps** (e.g. Tailwind, TanStack) with feature work. Separate `chore:` commit.
