---
name: node-cli-development
description: Use when writing or modifying the Node CLI target. Covers TS-Rest contracts + Bombshell (args/clack/tab) + Drizzle + plainjob + Evlog idioms and single-file-executable packaging with tsdown. Complementary to `implementing-a-spec` (process).
---

# Node CLI Development

This skill covers **how to write the CLI when its stack is Node**. The CLI is one target with a
choice of stack — Node, Rust, or Go — picked at setup; this is the **Node** stack. For the _workflow_
of implementing a spec, see `implementing-a-spec`. For _what to build_, see the spec.

The Node stack is distinctive among the CLI stacks: it is both a **client** (a headless/automation
client that consumes the API) and a **host of the API itself** (a TS-Rest web server). Hosting a
server isn't unique to Node — the Go stack can host via oapi-codegen — but it's the Node stack's
native shape. The web app is the reference implementation for behavior; in OpenAPI mode this stack
owns the wire contract every native client generates against.

## Stack at a glance

| Concern                | Choice                                             | First-party docs                                                                               |
| ---------------------- | -------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| Runtime                | Node single-file executable (SEA)                  | [single-executable-applications](https://nodejs.org/api/single-executable-applications.html)   |
| Exe bundler            | tsdown                                             | [tsdown.dev/options/exe](https://tsdown.dev/options/exe)                                       |
| Arg parser             | Bombshell Args                                     | [github.com/bombshell-dev/args](https://github.com/bombshell-dev/args)                         |
| Prompts                | Bombshell Clack                                    | [github.com/bombshell-dev/clack](https://github.com/bombshell-dev/clack)                       |
| Shell completions      | Bombshell Tab                                      | [github.com/bombshell-dev/tab](https://github.com/bombshell-dev/tab)                           |
| Server / RPC / OpenAPI | TS-Rest                                            | [ts-rest.com](https://ts-rest.com)                                                             |
| Database               | Drizzle (node:sqlite)                              | [orm.drizzle.team/docs/connect-node-sqlite](https://orm.drizzle.team/docs/connect-node-sqlite) |
| Background jobs        | plainjob                                           | [github.com/justplainstuff/plainjob](https://github.com/justplainstuff/plainjob)               |
| Networking             | fetch                                              | [MDN Fetch API](https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API)                    |
| Logging                | Evlog                                              | [evlog.dev](https://www.evlog.dev/)                                                            |
| Tests                  | Vitest                                             | [llms.txt](https://vitest.dev/llms.txt)                                                        |
| Linter / formatter     | Oxlint + Oxfmt                                     | [llms.txt](https://oxc.rs/llms.txt)                                                            |
| Type checker           | tsgo                                               | —                                                                                              |
| Package manager        | pnpm                                               | [pnpm.io](https://pnpm.io/)                                                                    |
| Production             | Single-file executable; Railway VPS for hosted API | [railway.com](https://railway.com/)                                                            |

When you need to look something up, fetch the relevant doc with WebFetch. For Vitest and Oxc the
`/llms.txt` is the index — it routes you to the page, it is not the content.

## This target owns the contract

> This section applies in **OpenAPI** backend mode (see `specs/ARCHITECTURE.md` → "Backend modes").
> In **Convex** mode the CLI uses the Convex client instead of hosting a contract; in **No API**
> mode it's a standalone tool with only local state.

The **TS-Rest router/contract is the single source of truth for the HTTP surface.** It is the
canonical description of every endpoint: path, method, inputs, outputs, errors. From it, TS-Rest
emits the **OpenAPI document** that every native client (Apple, Android, Windows, Linux, the Rust
CLI) generates its typed client against. Change the contract and you change every downstream client.

Because of that:

- **Keep the contract in its own module/package** (`src/contract/`) so both the server _and_ clients
  can import it. The server implements it; this target consumes it; the OpenAPI emission reads it.
- **The server implements the contract; it never defines a second one.** Handlers satisfy the
  contract's shapes; they don't invent their own.
- **This target is also a client.** It calls the API through the TS-Rest fetch client built from the
  same contract — it does not hand-roll requests against the OpenAPI surface.
- **Reverse-point contract endpoints with `// SPEC: protocol.<area>.<op>`** on the contract
  definition, matching the `protocol.*` IDs Convex functions use on web. The server handler that
  implements an endpoint carries the same ID.

No client mirrors the protocol by hand. The contract is the only thing that crosses the wire boundary.

## Idioms (read these before writing code)

### Commands: define with Args, prompt with Clack

Command definitions are **declarative** — describe the command, its arguments, and its flags with
Bombshell Args. Interactive input uses Clack. Shell completions come from Tab. Keep the wiring thin:
the command file parses, prompts, and delegates to pure logic.

```ts
// SPEC: manual (CLI wiring for vm.items.create)
import { command } from "@bombsh/args";
import * as clack from "@clack/prompts";
import { createItem } from "../lib/items.create";

export const createCommand = command("create")
    .option("name", { type: "string" })
    .action(async ({ name }) => {
        const chosenName = name ?? (await clack.text({ message: "Item name" }));
        const result = createItem({ name: String(chosenName) });
        process.stdout.write(`${result.id}\n`);
    });
```

The reverse pointer for behavior lives on the **pure logic**, not the command wiring. Command files
carry `// SPEC: manual` — they have no cross-target contract.

### Separate pure command logic from I/O

The work a command performs is a **pure, testable function** in `src/lib/`. Argument parsing,
prompting, printing, and network/DB calls live at the edges. Tests target the pure logic, not the
parsed-args plumbing.

```ts
// SPEC: vm.items.create
export interface CreateItemInput {
    name: string;
}

export function createItem(input: CreateItemInput): { id: string; name: string } {
    const name = input.name.trim();
    if (name.length === 0) throw new Error("Item name must not be empty");
    return { id: crypto.randomUUID(), name };
}
```

### TS-Rest is the boundary

Server handlers implement the contract; the target consumes it. Validate inputs **at the contract**
— TS-Rest carries the request/response schemas, so a malformed input is rejected at the boundary, not
deep in a handler. Internal code downstream of the contract trusts the validated shape.

```ts
// SPEC: protocol.items.create
import { s } from "./contract"; // the shared TS-Rest contract
import { createItem } from "../lib/items.create";

export const itemsHandlers = {
    createItem: async ({ body }) => {
        const item = createItem(body);
        return { status: 201, body: item };
    },
};
```

### Drizzle for SQL; plainjob for background work

Local/server state goes through **Drizzle over `node:sqlite`** — schema in `src/db/`, explicit
queries, no ORM magic in handlers. Queued or deferred work goes through **plainjob** (`src/jobs/`),
not ad-hoc `setTimeout` or fire-and-forget promises. Structured logging goes through **Evlog**, not
`console.log`, in anything that ships. `process.stdout`/`process.stderr` is for the CLI's own
user-facing output; `console.*` is fine for throwaway local debugging — remove it before committing.

### Single-file exe packaging via tsdown

The release artifact is a **single-file executable** built by tsdown's exe target
([tsdown.dev/options/exe](https://tsdown.dev/options/exe)), which wraps Node's SEA. Don't ship a
`node_modules` tree or a loose entrypoint as the deliverable. The hosted API runs the same codebase
on a Railway VPS.

## File layout (within the target's source)

```
<source>/src/
├── commands/          ← Bombshell command definitions (Args + Clack + Tab wiring)
├── contract/          ← TS-Rest contract + OpenAPI emission (the wire source of truth)
├── server/            ← TS-Rest handlers implementing the contract
├── db/                ← Drizzle schema and queries (node:sqlite)
├── jobs/              ← plainjob queues and workers
└── lib/               ← pure command logic (the spec target; tests live here)
```

## Verifying

The verify-iterate loop is **build → run command → assert output → fix**.

- **CLI commands:** build the binary, run it, and assert on stdout/stderr and exit code.
    ```sh
    node ./dist/cli.js items create --name Foo || echo "exit:$?"
    ```
    Check the printed output and the exit code together — a command that prints an error but exits `0`
    is a bug.
- **Command logic:** Vitest against the pure functions in `src/lib/`. Place a `// [scenario.<id>]`
  comment above each `it(...)` block (keep the ID out of the test name itself).

    ```ts
    import { describe, it, expect } from "vitest";
    import { createItem } from "./items.create";

    describe("vm.items.create", () => {
        // [scenario.items.create.empty-name]
        it("rejects a blank name", () => {
            expect(() => createItem({ name: "  " })).toThrow();
        });
    });
    ```

- **Server / contract:** test handlers against the TS-Rest contract — assert each handler satisfies
  the contract's response shape. Tag contract tests with `describe("protocol.<area>.<op>")`.

- `specify verify <target>` runs the target's configured test command, joins scenarios to tests, and
  reports pass/fail against the spec. Run it and read the output before claiming success.

See `specs/CONVENTIONS.md` for the full spec ID format, scenario sub-ID scheme, and the
source-comment binding affordance (`// SPEC: <id>` reverse pointers).

Run the verifying command **in this turn** and read its output before claiming success — see
`verification-before-completion`.

## When to invoke a more specific skill

- About to write tests? → `test-driven-development`
- About to claim work is done? → `verification-before-completion`
- Debugging something unexpected? → `systematic-debugging`
- Implementing a spec end-to-end? → `implementing-a-spec`
- Building the CLI in Rust or Go instead? → `rust-cli-development` / `go-cli-development`

## Commit

Land focused, atomic commits as the work hits natural boundaries — typically per spec ID, per command
+ its pure-logic tests, or per cohesive refactor. Use the scoped-commits convention
(`specify gate scope`).

Node-CLI-specific notes:

- **Contract changes go alone.** An edit to `src/contract/` ripples to every generated client — it
  belongs in its own commit (`feat: add protocol.items.create endpoint`) so the downstream
  regeneration is reviewable in isolation. Don't bundle it with handler or command code.
- **Drizzle schema/migration changes are separate** from the queries that consume them.
- **Don't bundle dependency bumps** (Bombshell, TS-Rest, tsdown) with feature work. Separate
  `chore:` commit.
