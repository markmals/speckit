---
description: The npm-package scaffold — the node-family single-TS-library twin of swift-package; its scoped-binding choice, the source-read junit binding, and the toolchain gotchas (tsdown extension, oxfmt chain).
---

# npm-package scaffold (internal/coreassets/templates/scaffolds/npm-package)

A single TypeScript library — the **node-family twin of [[apple-scaffold|swift-package]]**
(built 2026-06-27). `family: node`, `memberDir: packages`. Verify target = the package's
own Vitest suite (`mise //packages/<name>:test` → junit), green-on-arrival with just the
node toolchain. Render test: `internal/scaffold/npm_package_test.go`. Engine: zero changes
— it reuses the junit path web already proved.

## The decided calls

- **`bindings: scoped`, NOT web's strict/empty.** A library mixes scenario tests with
  plain unit/property tests; scoped leaves untagged tests out of scope instead of failing
  `verify` on them. This matches the swift library twins — web is strict because it's an
  *app*. Proven: adding an untagged `*.test.ts` to `src/` keeps verify green.
- **The junit binding is read from SOURCE, never the report.** `vitestBindRe`
  (`internal/engine/verify.go`) scans the `it("[scenario.<id>] …")` title in `src/*.ts`;
  the report carries only test identity + pass/fail. "Report-carried" is a misnomer — the
  difference from swift is only *where the id sits* (visible it() title vs a `.scenario()`
  trait), both source-read.
- **`nameRule: "npm"`.** `src/` is name-agnostic (the module is named by package.json
  `name: {{.Name}}`), so it needs no swift-style *identifier* rule. But the member name *is*
  the npm package name, so the `npm` rule (`scaffold.go ValidateName` + the `npmReserved`
  map) rejects capitals, the blacklist (`node_modules`/`favicon.ico`), Node core-module
  names, and >214 chars at `target add` — before they'd fail at `npm publish`. The base slug
  check runs first (excludes spaces/`@`/`/`/leading-dot), so the rule only adds what a valid
  slug still slips past npm. The `npmReserved` list is a fail-open snapshot (stale → misses a
  new builtin, never a false reject).
- **fmt/lint scoped to `src/`** (`oxfmt src` / `oxlint src`), like web's `app/`. So
  package.json + the configs at member root are NOT format-gated — only the three
  `src/*.ts` files must be oxfmt/oxlint-clean. This is what lets `pnpm add` rewrite
  package.json freely without breaking `fmt:check`.
- **Example bound to `story.slug.create`** (a `slugify`), chosen DISTINCT from web
  (`welcome.greet`) and swift-package (`version.compare`) so two stacks in one repo never
  collide on `features/`. `kind: story` (not `kind: library` — that taxonomy is still
  pending across all library stacks).

## Toolchain gotchas that cost real time

- **tsdown defaults to `.mjs`/`.d.mts`** — mismatches package.json `exports` (`./dist/
  index.js`). Fix in `tsdown.config.ts`: `outExtensions: () => ({ js: ".js", dts: ".d.ts" })`.
- **oxfmt breaks a 3+ method chain across lines.** `slugify.ts`'s `.toLowerCase().replace()
  .replace()` must be pre-formatted multi-line in the template — the scaffold runs no
  formatter, so the emitted bytes must already be oxfmt-clean (verify by rendering + `oxfmt
  --check src`). Same green-on-arrival discipline as [[web-scaffold]].
- **The node drift test couples only `test` + `typecheck`** (`TestNodeFamilyMatchesNpmPackageInline`)
  — their bodies are byte-identical to node.toml so they promote to `extends` when
  npm-package is the 2nd node member; build/fmt/lint are src-scoped, differ from web's
  app-scoped templates, and stay inline. See [[mise-monorepo]].
- **Test each library stack in its OWN fresh repo** (the first-target-seeds-example gotcha
  — [[apple-scaffold]]).

## `npm-package` replaced `ts-lib` (Mark's call, 2026-06-27)

The roster name for the TS-library stack used to be `ts-lib` (a planned, scaffold-LESS
"register an existing library" name). `npm-package` **retired and replaced it** everywhere:
the `config.go` Stack enum, `stack-scaffolding.md`, `library-products.md`, `BACKLOG.md`, and
`HANDOFF.md` now say `npm-package`. The canonical **scaffold-less** example in the register
code/tests (`register_test.go`, `main.go`/`monorepo.go` comments) moved to **`go-cli`**
(`node-cli` as the second example). You can still `register` an existing TS library as
`npm-package` without scaffolding — `register` seeds wiring from the scaffold manifest, or
takes explicit `--format/--source/--bindings` flags. Stacks with no scaffold dir today:
`go-cli`, `node-cli`, `website`, `android`, `vscode-extension`.
