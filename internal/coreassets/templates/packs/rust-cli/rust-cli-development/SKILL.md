---
name: rust-cli-development
description: Use when writing or modifying the Rust CLI target. Covers Clap + charmed_rust (bubbletea/bubbles/lipgloss/huh/glamour/harmonica/wish — a Rust port of the Charm ecosystem) + Diesel + reqwest + Progenitor (generated OpenAPI client) idioms with cargo test / rustfmt / clippy. Complementary to `implementing-a-spec` (process).
---

# Rust CLI Development

This skill covers **how to write the CLI when its stack is Rust**. The CLI is one target with a
choice of stack — Node, Rust, or Go — picked at setup; this is the **Rust** stack. For the _workflow_
of implementing a spec, see `implementing-a-spec`. For _what to build_, see the spec.

The web app is the **reference implementation**. This is a **terminal client** — it mirrors web
behavior idiomatically in a TUI. Read the web realization for context; the spec is authoritative.

The TUI layer is **[charmed_rust](https://github.com/dicklesworthstone/charmed_rust)** — a Rust port
of the Charm ecosystem (bubbletea · bubbles · lipgloss · huh · glamour · harmonica · wish), chosen
so this stack stays at parity with the Go CLI stack. It needs **Rust 1.85+ (edition 2024)**.

## Stack at a glance

| Concern            | Choice                                                                                   | First-party docs                                                                                        |
| ------------------ | ---------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| Language           | Rust                                                                                     | [rust-lang.org](https://www.rust-lang.org/)                                                             |
| Arg parser         | Clap (derive)                                                                            | [docs.rs/clap](https://docs.rs/clap)                                                                    |
| State + view       | charmed_rust `bubbletea` (Elm/TEA-style model · message · update · view)                 | [charmed_rust](https://github.com/dicklesworthstone/charmed_rust)                                       |
| Components         | charmed_rust `bubbles` (lists, inputs, tables, spinners, viewports, file pickers)        | [charmed_rust](https://github.com/dicklesworthstone/charmed_rust)                                       |
| Styling            | charmed_rust `lipgloss`                                                                  | [charmed_rust](https://github.com/dicklesworthstone/charmed_rust)                                       |
| Forms / prompts    | charmed_rust `huh`                                                                       | [charmed_rust](https://github.com/dicklesworthstone/charmed_rust)                                       |
| Markdown render    | charmed_rust `glamour`                                                                   | [charmed_rust](https://github.com/dicklesworthstone/charmed_rust)                                       |
| Animations         | charmed_rust `harmonica`                                                                 | [charmed_rust](https://github.com/dicklesworthstone/charmed_rust)                                       |
| SSH app framework  | charmed_rust `wish`                                                                      | [charmed_rust](https://github.com/dicklesworthstone/charmed_rust)                                       |
| On-device database | Diesel (SQLite)                                                                          | [diesel.rs](https://diesel.rs/)                                                                         |
| Networking         | reqwest                                                                                  | [docs.rs/reqwest](https://docs.rs/reqwest)                                                              |
| API client         | Progenitor (typed client generated from the OpenAPI document — don't hand-roll requests) | [github.com/oxidecomputer/progenitor](https://github.com/oxidecomputer/progenitor)                      |
| Tests              | cargo test (+ `insta` for snapshot assertions on rendered views)                         | [doc.rust-lang.org/cargo/commands/cargo-test](https://doc.rust-lang.org/cargo/commands/cargo-test.html) |
| Formatter          | rustfmt                                                                                  | [github.com/rust-lang/rustfmt](https://github.com/rust-lang/rustfmt)                                    |
| Linter             | Clippy                                                                                   | [doc.rust-lang.org/stable/clippy](https://doc.rust-lang.org/stable/clippy/)                             |
| Package / build    | Cargo (single-file release binary)                                                       | [crates.io](https://crates.io/)                                                                         |
| Auth               | Clerk (token attached to the generated client)                                           | [clerk.com/docs](https://clerk.com/docs)                                                                |

The Go CLI stack mirrors this exactly with the original Charm libraries — see `go-cli-development`.

## The Client layer depends on the backend mode

How the CLI reaches its data is set by the project's backend (see `specs/ARCHITECTURE.md` →
"Backend modes"):

- **OpenAPI** — a typed client generated by Progenitor over **reqwest**; never assemble requests by hand.
- **Convex** — Convex's Rust client.
- **No API** — no client at all; **Diesel/SQLite** is the source of truth.

In the remote modes, **Diesel/SQLite** is a local-first cache (not a second backend) and identity
flows through **Clerk**, whose token the client attaches.

## Idioms (read these before writing code)

### Clap derive for the CLI surface

Define commands and args declaratively with `#[derive(Parser)]`. Keep parsing thin — parse into a
struct, then hand off. No business logic in the arg layer.

```rust
use clap::{Parser, Subcommand};

#[derive(Parser)]
#[command(name = "app")]
struct Cli {
    #[command(subcommand)]
    command: Command,
}

#[derive(Subcommand)]
enum Command {
    /// List items
    List,
    /// Create an item
    Create { title: String },
}
```

### charmed_rust bubbletea: pure state + event loop, separate from rendering

Hold UI state in an **app-state model**. An `update` step folds a message into new state; a `view`
step renders a string from that state via `lipgloss`/`bubbles`. The **update layer is pure and
testable** — no IO, no terminal handles. This pure layer is the view-model analog and the **primary
spec target**. `bubbletea` (charmed_rust's Elm/TEA-style runtime) supplies the model · message ·
update · view loop — you write the model and the pure `update`; the runtime owns the event pump and
redraw scheduling.

```rust
// SPEC: vm.items.list
pub struct ItemsListModel {
    pub items: Vec<Item>,
    pub selected: usize,
}

pub enum Msg {
    Up,
    Down,
}

impl ItemsListModel {
    pub fn update(&mut self, msg: Msg) {
        match msg {
            Msg::Up => self.selected = self.selected.saturating_sub(1),
            Msg::Down => {
                if self.selected + 1 < self.items.len() {
                    self.selected += 1;
                }
            }
        }
    }
}
```

- The reverse pointer lives on the **model** (the pure update layer), not the view code.
- `view` reads state and renders with `lipgloss`/`bubbles`; it holds no logic. Tag it
  `// SPEC: manual` — it has no cross-target contract.
- IO (reqwest calls, Diesel reads, terminal events) lives at the edges and feeds messages into `update`.

### Tests at the logic layer

Follow the Rust convention: a `#[cfg(test)] mod` carrying `// SPEC: <id>`, with a
`// [scenario.<id>]` comment above each `#[test]` fn (Rust test names can't hold dots or brackets,
so the scenario sub-ID lives in the comment that drift tooling greps). The `specify verify <target>`
command runs the target's configured test command and joins scenarios to tests; format the test output
as `junit` so the engine can parse it.

```rust
#[cfg(test)]
mod tests {
    // SPEC: vm.items.list
    use super::*;

    // [scenario.items.list.move-down]
    #[test]
    fn moving_down_advances_selection() {
        let mut model = ItemsListModel {
            items: vec![Item::default(), Item::default()],
            selected: 0,
        };
        model.update(Msg::Down);
        assert_eq!(model.selected, 1);
    }
}
```

The pure `update` carries the behavioral coverage. When you need to pin the rendered frame, snapshot
the `view` string with `insta` — no real terminal required.

```rust
// [scenario.items.list.renders-title]
#[test]
fn renders_selected_title() {
    let model = ItemsListModel { items: vec![item("First item")], selected: 0 };
    insta::assert_snapshot!(model.view());
}
```

See `specs/CONVENTIONS.md` for the full spec ID format, scenario sub-ID scheme, and the
source-comment binding affordance (`// SPEC: <id>` reverse pointers).

### Error handling with `Result`

Return `Result` from anything fallible. Use `thiserror` for library-style typed errors at the domain
boundary; `anyhow` for the application top level where you just want context and a clean exit. **No
`unwrap()` or `expect()` on fallible IO in shipped paths** — propagate with `?` and surface failures,
don't swallow them. `unwrap()` is fine only on invariants that genuinely can't fail (and a
`// SAFETY`-style comment helps the next reader).

## File layout (within the target's source)

```
<source>/src/
├── main.rs            ← entry point; wires Clap → app → bubbletea runtime
├── cli.rs             ← Clap parser definitions (thin)
├── app/               ← models + pure update logic (the spec target)
├── ui/                ← bubbletea views: lipgloss styling + bubbles widgets (dumb; SPEC: manual)
├── api/               ← Progenitor client wrapper (reqwest transport)
└── db/                ← Diesel schema, models, local-first cache
```

## Verifying

- Run the target manually (e.g. `cargo run`) for checks against a live terminal.
- `cargo test` for the pure logic layer — this is where behavioral coverage lives.
- `specify verify <target>` runs the target's configured test command, joins scenarios to tests, and
  reports pass/fail against the spec. Run it and read the output before claiming success.
- `insta` snapshots to pin a rendered `view` string inside tests.
- `cargo clippy` and `cargo fmt --check` must be clean before you declare done.

Run the verifying command **in this turn** and read its output before claiming success — see
`verification-before-completion`.

## When to invoke a more specific skill

- About to write tests? → `test-driven-development`
- About to claim work is done? → `verification-before-completion`
- Debugging something unexpected? → `systematic-debugging`
- Implementing a spec end-to-end? → `implementing-a-spec`

Related: the other two CLI stacks are `go-cli-development` (the same Charm TUI in Go) and
`node-cli-development` (the Node/TS-Rest stack; owns the OpenAPI contract in OpenAPI mode).

## Commit

Land focused, atomic commits as the work hits natural boundaries — typically per spec ID, per model +
its tests, or per cohesive refactor. Use the scoped-commits convention (`specify gate scope`).

Rust-specific notes:

- **`Cargo.toml` / `Cargo.lock` bumps go alone.** A dependency change is its own commit
  (`chore: bump <crate>`); don't bundle it with feature code.
- **Generated Progenitor output rides with the contract/codegen commit** that produced it — never
  commit it out of sync with the OpenAPI document.
- **Don't commit build outputs.** The `target/` directory is gitignored.
