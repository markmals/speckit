---
name: appkit-setup
description: Use when setting up a new Mac for this repo's Apple target, or when an appkit skill reports a missing prerequisite (xcodebuild/tuist not found, Xcode license not accepted, sdk-api/sdk-search missing). Installs and verifies Xcode + the macOS SDK, Command Line Tools, Homebrew, Tuist, swift-format, create-dmg, and the sdk-api/sdk-search grounding tools.
disable-model-invocation: true
---

# AppKit setup

Install and verify the machine prerequisites every other `appkit-*` skill assumes are present. This is the skill another skill points you to when a build, format, or grounding command fails because a tool is missing.

This skill is **user-invoked only** — run it explicitly with `/appkit-setup`. The agent won't load it on its own; if a later command fails for a missing prerequisite, it stops and asks you to run this. It is **idempotent**: every step checks first, skips if satisfied, and re-running on a set-up Mac is a fast no-op.

## What the scaffold needs

| Prerequisite | Why | Install |
| --- | --- | --- |
| Full Xcode (current macOS SDK) | `tuist`, the app build, and `sdk-api` all need the SDK — not just CLT | Mac App Store or `xcodes` (multi-GB — never auto-download) |
| Xcode license accepted | `xcodebuild` fails until it is | `sudo xcodebuild -license accept` |
| Command Line Tools | `swift`, `git`, `xcode-select` | `xcode-select --install` |
| Homebrew | installs the CLI tools below | [brew.sh](https://brew.sh) |
| swift-format | `mise run -C <dir> fmt`/`lint` (ships in the Xcode toolchain as `swift format`; standalone via brew) | `brew install swift-format` |
| create-dmg | DMG packaging (only if you ship one) | `brew install create-dmg` |
| `sdk-api`, `sdk-search` | the grounding tools the whole suite leans on (see below) | `mise run install` in apple-platform-tools |

Tuist itself is **pinned by mise** in each app's `mise.toml` (`[tools] tuist = "…"`) — `mise run -C <dir> generate` provisions it on first use; you do **not** `brew install tuist`. The generated `.xcodeproj`/`.xcworkspace`/`Derived/` are gitignored. See `appkit-setup`'s sibling `ios-development` for the layout (headless SwiftPM `Core` + the AppKit surface under `macOS/Sources/App`).

## Detect everything first

Batch the checks, show the full picture, then install only what's missing.

```bash
xcode-select -p                          # full Xcode, not /Library/Developer/CommandLineTools
xcodebuild -version                      # prints version; license-error if not accepted
command -v brew swift-format create-dmg  # CLI tools on PATH
command -v sdk-api sdk-search            # grounding tools in ~/.local/bin
mise --version                           # task runner the scaffold drives Tuist through
```

If `xcodebuild -version` emits a license error, the SDK is unusable until you accept:

```bash
sudo xcodebuild -license accept          # needs admin — ask the user before triggering sudo
```

## Install what's missing

**Xcode — do NOT auto-install.** A full Xcode (for the current macOS SDK) is several GB. If only CLT is present (`xcode-select -p` → `/Library/Developer/CommandLineTools`) or the version is too old, tell the user and offer options — App Store, or `xcodes` for a pinned version — then point selection at it:

```bash
brew install xcodesorg/made/xcodes && xcodes install 27   # or the SDK your scaffold targets
sudo xcode-select -s /Applications/Xcode.app              # adjust if versioned
```

**Homebrew + CLI tools** (only if missing, then upgrade — these ship breaking changes):

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
eval "$(/opt/homebrew/bin/brew shellenv)"   # don't skip — later brew calls fail otherwise
brew install swift-format create-dmg
brew upgrade swift-format create-dmg
```

`swift-format` is bundled with the Xcode toolchain (run as `swift format`), which is what the scaffold's `mise run -C <dir> fmt` and `lint` tasks call; install the standalone binary only for parity in CI or editor integrations. Both honor the committed `.swift-format` (lineLength 100, 4-space).

**Developer mode** (optional — lets the test runner/debugger attach without repeated auth prompts; needs admin, ask first):

```bash
sudo DevToolsSecurity -enable
```

## The grounding tools (required)

`sdk-api` and `sdk-search` are external CLIs built from the **apple-platform-tools** monorepo. They are the suite's grounding mandate, not a convenience — `appkit-design` and every AppKit-writing skill call them constantly. Build and install both:

```bash
mise run install   # in the apple-platform-tools checkout → builds + installs into ~/.local/bin
```

Confirm they answer (both should return JSON):

```bash
sdk-api check NSGlassEffectView          # verify a symbol exists + its macOS availability
sdk-search "source list sidebar"         # find the canonical AppKit pattern
```

**Never vendor these binaries into the project** — they live in `~/.local/bin`, owned by the apple-platform-tools repo. The discipline they enforce: before writing any AppKit code, verify every symbol and `@available` version with `sdk-api`, and search the canonical pattern with `sdk-search`. Never guess a symbol name or availability. (HIG guidance is the same workflow — read [developer.apple.com/design/human-interface-guidelines](https://developer.apple.com/design/human-interface-guidelines) and search patterns with `sdk-search`; HIG files are not vendored either.)

## Signing is out of scope here

A **Developer ID Application** certificate is needed only for signing/notarization, and creating it runs through the Apple Developer portal and your account — not machine setup. If a packaging step later reports no identity, point the user to the portal (Certificates → Developer ID Application).

## Verify the scaffold builds

Once the tools are in, prove the toolchain end to end against an app target:

```bash
mise run -C <dir> generate      # Tuist provisions itself + writes the Xcode project
mise run -C <dir> build         # build the macOS AppKit surface
mise run -C <dir> test          # the headless Core suite — what `specify verify` runs
```

`mise run -C <dir> test` writes the `swift` event-stream report (NDJSON) the engine joins by scenario id; see `ios-development` for the Swift Testing `.spec(...)`/`.scenario(...)` trait discipline.

## Things to NOT do

- **Don't auto-download Xcode** — it's multi-GB. Always ask; offer App Store / `xcodes`.
- **Don't trigger `sudo` without asking** — license acceptance and `DevToolsSecurity` both need admin; confirm before each.
- **Don't `brew install tuist`** — it's pinned per-app in `mise.toml` and provisioned by `mise run -C <dir> generate`.
- **Don't vendor `sdk-api`/`sdk-search`** (or HIG files) into the project — they belong in `~/.local/bin` via apple-platform-tools.
- **Don't create signing certificates** — that's an Apple-account flow, not setup.
- **Don't silently retry on failure** — if a `brew install` fails (no network, permissions), record it and move on so the user sees what broke.

## Related

- Xcode MCP for build/test/diagnostics is **per-machine** — see the MCP section of `ios-development` (user/local config, not committed). Not set up here.
- Writing AppKit code? → `appkit-design`. Driving the simulator? → `ios-simulator-control`.
- Process skills: `implementing-a-spec`, `test-driven-development`, `verification-before-completion`, `systematic-debugging`.
