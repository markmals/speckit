---
name: web-verification
description: Use to drive the web target via the `chrome-devtools` CLI for visual verification, UI debugging, and behavioral checks. Trigger when verifying web UI changes, screenshotting an app state, simulating interaction, or inspecting console/network in a tight verify-iterate loop. Analog of `ios-simulator-control` and `android-emulator-control` for the web target.
---

# Web Verification

Drive the web target from Claude Code via the **`chrome-devtools` CLI** — a background daemon plus one subcommand per browser action. Tight verify-iterate loops: change code → reload → snapshot → assert → fix → repeat.

The CLI is the same `chrome-devtools-mcp` engine the template used to register as an MCP server, driven over the shell instead. It's pinned in the repo's **root `mise.toml`** (`"npm:chrome-devtools-mcp"`), so `mise install` puts the `chrome-devtools` command on your PATH — no separate install step. (Working in a copy that hasn't pinned it? `mise use npm:chrome-devtools-mcp@latest`.)

Every browser action is a `chrome-devtools <command>` call — run them with the `Bash` tool. The daemon keeps one Chromium instance alive across calls, so there's no per-call launch cost; you're scripting a long-lived browser.

## Daemon lifecycle

`start` boots (or restarts) the background daemon and decides which browser it drives. Point it at **Chromium** (not Chrome), isolated profile — matching the template's reproducibility intent:

```sh
chrome-devtools start \
  --executablePath /Applications/Chromium.app/Contents/MacOS/Chromium \
  --isolated
```

- `--isolated` uses a throwaway user-data-dir cleaned up on stop (the default when no `--userDataDir` is given; passed here for intent).
- Add `--no-headless` if you want to watch the browser drive itself; headless (the default) screenshots identically.
- `chrome-devtools status` — is the daemon up, and with what args.
- `chrome-devtools stop` — kill it (and the isolated profile) when done.

Commands auto-start a daemon if none is running, but with the **default** browser (system Chrome) — so run `start` with the Chromium `--executablePath` first, or you'll be driving the wrong browser. `start` is idempotent ("start or restart").

## Output format

Every command prints Markdown by default; pass `--output-format json` when you want to parse the result (page lists, network requests, console dumps):

```sh
chrome-devtools list_pages --output-format json
```

## Open a page

```sh
chrome-devtools new_page "http://localhost:5173/"
```

(Default dev URL is `http://localhost:5173/` for Vite; check the dev-server output for the real port.)

Navigate an existing tab:

```sh
chrome-devtools navigate_page --type url --url "http://localhost:5173/items"
```

Reload (e.g. after a change that didn't HMR):

```sh
chrome-devtools navigate_page --type reload --ignoreCache
```

Multiple tabs: `list_pages`, then `select_page <pageId>` to choose the context, `close_page <pageId>` to drop one.

## Screenshot

```sh
chrome-devtools take_screenshot \
  --filePath .tmp/screenshots/items-list.png \
  --fullPage          # omit for just the viewport
```

`--filePath` is absolute, or relative to the directory you run the command from. Save under a gitignored path inside the target's source (e.g. `.tmp/screenshots/`) so before/after shots don't pollute the tree. Then `Read` the file to view it. Screenshot a single element with `--uid <uid from snapshot>` (incompatible with `--fullPage`).

Take screenshots aggressively when verifying visual changes; name them descriptively (`items-list-empty.png`, `item-create-error.png`) so comparison is easy.

## Inspect what's on screen

Accessibility snapshot — text representation of the DOM with stable element UIDs:

```sh
chrome-devtools take_snapshot
```

Use the returned UIDs to drive interactions or assert structure. Prefer a snapshot over a screenshot when checking _content_ rather than _visuals_. (`--filePath` saves it; `--verbose` includes the full a11y tree.)

## Interact

```sh
chrome-devtools click  <uid>                 # --dblClick for double-click
chrome-devtools fill   <uid> "Jane Doe"      # text input, textarea, or <select>
chrome-devtools hover  <uid>
chrome-devtools press_key Escape
chrome-devtools type_text "Hello"            # into the focused element
```

UIDs come from the latest `take_snapshot` — re-snapshot after anything that re-renders. Add `--includeSnapshot` to `click`/`fill` to get the post-action snapshot in the same call (saves a round-trip in loops). There's no `fill_form` command — fill a form with one `fill` per field.

## Read console messages

```sh
chrome-devtools list_console_messages --types error warn
```

Check the console **first** after a change before deciding it worked — silent JS errors are the usual cause of "looks blank, no obvious problem." Drop `--types` to see everything; `get_console_message <msgid>` drills into one.

## Inspect network

```sh
chrome-devtools list_network_requests --resourceTypes document fetch xhr
chrome-devtools get_network_request --reqid <number from list>
```

Useful for catching unexpected requests (a stale cache, or a Convex call firing twice).

## Evaluate JavaScript in the page

The escape hatch for everything else — pass a function declaration as the positional arg:

```sh
chrome-devtools evaluate_script '() => ({
  title: document.title,
  hasItemList: !!document.querySelector("[data-testid=items-list]"),
  itemCount: document.querySelectorAll("[data-testid=item-row]").length,
})'
```

Use it to confirm the app mounted (`document.getElementById("app").children.length > 0`), read computed styles when a visual check is ambiguous, or inspect page data. Returned values must be JSON-serializable.

This is also how you **wait** — there's no `wait_for` command. Poll a condition before asserting:

```sh
chrome-devtools evaluate_script 'async () => {
  for (let i = 0; i < 50; i++) {
    if (document.querySelector("[data-testid=items-list]")) return "ready";
    await new Promise(r => setTimeout(r, 100));
  }
  return "timeout";
}'
```

For a coarse pause, `sleep 0.5` between commands is fine.

## Lighthouse + performance

```sh
chrome-devtools lighthouse_audit --device desktop            # a11y / SEO / best-practices
chrome-devtools performance_start_trace --reload --autoStop  # then performance_stop_trace
```

Lighthouse excludes performance — use the perf trace for Core Web Vitals (LCP, INP, CLS).

## Verify-iterate loop pattern

```
1. Make a code change in the target's source.
2. Dev server hot-reloads. If you suspect it didn't:
   chrome-devtools navigate_page --type reload --ignoreCache
3. Drive the UI to the state you want (navigate → click → fill → …).
4. If the action is async, poll with evaluate_script (above) before asserting.
5. take_screenshot OR take_snapshot OR list_console_messages.
6. Decide: did it work? Yes → move on. No → fix and repeat.
```

Because each step is a shell command, you can chain a whole loop iteration in one `Bash` call. If you run the same sequence repeatedly, script it as a named task in the target's tooling.

## Prerequisites

- The target's dev server running (default `http://localhost:5173/` for Vite; confirm the port).
- Any backend dev process (e.g. `services/convex dev`) running in parallel if the page touches live data.
- The CLI on PATH — pinned in the root `mise.toml`, so `mise install` provides it (or `mise use npm:chrome-devtools-mcp@latest` in a copy that lacks the pin).
- Chromium installed at the `--executablePath` you pass to `start` (default macOS: `/Applications/Chromium.app/Contents/MacOS/Chromium`). If yours is elsewhere, pass the right path.

## Running the behavioral suite

To run the full behavioral suite for the web target:

```sh
specify verify web
```

The engine runs the target's configured test command and joins scenarios to tests. For non-test tooling (lint, build, typecheck) use the target's own commands directly — refer to the target's README or tooling config.

## Common gotchas

- **404 for `/favicon.ico`** is harmless — the browser's default request. Our favicon is at `/favicon.svg`.
- **`Content-Type: text/html`** on what should be a static asset means Vite is serving its SPA shell as a fallback because the URL didn't match a known asset. Check `public/` resolution.
- **Empty `#app`** = React didn't mount. Almost always a runtime error in the entry script — `list_console_messages` with no `--types` filter.
- **Stale data after a Convex mutation** = the query cache wasn't invalidated. Check the mutation's `optimisticUpdate` / cache config in the view model.
- **"No daemon running" or driving the wrong browser** = re-run `chrome-devtools start …` with the Chromium `--executablePath`.

## When NOT to use this skill

- Pure unit tests of view models or selectors — run `specify verify web`, no browser needed.
- One-off lookups of page state not tied to a code change.
- Anything with a faster path through Vitest + jsdom.

## Related skills

- `web-development` — how to _write_ the code being verified.
- `implementing-a-spec` — the workflow this verification supports.
- `test-driven-development` — verification at the unit level (where most coverage lives).
- `verification-before-completion` — never claim a visual change works without doing this loop first.
