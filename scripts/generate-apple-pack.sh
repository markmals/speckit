#!/usr/bin/env bash
set -euo pipefail

# Regenerates the GENERATED slice of the embedded apple pack from the mac-dev-skills
# appkit plugin source, retargeted for SpecKit. Generated: the deep skills appkit-design,
# appkit-private-apis, appkit-app-inspector, and the offline apple-hig HIG corpus, plus the
# appkit-dev stack agent. NOT touched: the other (concise, hand-maintained) appkit skills and
# the iOS skills. So the committed pack is a deliberate hybrid; this regenerates only the
# generated slice. Needs a mac-dev-skills checkout ($MAC_DEV_SKILLS_ROOT or ../skills/mac-dev-skills).
# CI runs `--check` against markmals/mac-dev-skills pinned to a commit (see .github/workflows/
# ci-go.yml); to pick up new source content, bump that ref, run this script, and commit both.
usage() {
  cat <<'EOF'
Usage: scripts/generate-apple-pack.sh [--check] [--source PATH]

Regenerates the GENERATED slice of templates/packs/apple from the mac-dev-skills appkit
plugin source: the deep appkit-design/private-apis/app-inspector skills, the offline
apple-hig HIG corpus, and the appkit-dev agent. The other (concise, hand-maintained) appkit
skills and the iOS skills are left in place.

Options:
  --check        regenerate in a temp dir and fail if the generated slice has drifted
  --source PATH mac-dev-skills checkout (default: $MAC_DEV_SKILLS_ROOT or ../skills/mac-dev-skills)
EOF
}

mode="write"
source_root="${MAC_DEV_SKILLS_ROOT:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --check)
      mode="check"
      shift
      ;;
    --source)
      source_root="${2:-}"
      if [[ -z "$source_root" ]]; then
        echo "--source requires a path" >&2
        exit 2
      fi
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
if [[ -z "$source_root" ]]; then
  source_root="$(cd "$repo_root/../skills/mac-dev-skills" 2>/dev/null && pwd || true)"
fi
if [[ -z "$source_root" || ! -d "$source_root/plugins/appkit" ]]; then
  cat >&2 <<EOF
mac-dev-skills source checkout not found.

Set MAC_DEV_SKILLS_ROOT or pass --source:
  MAC_DEV_SKILLS_ROOT=/path/to/mac-dev-skills scripts/generate-apple-pack.sh
EOF
  exit 1
fi

pack_root="$repo_root/internal/coreassets/templates/packs/apple"

copy_skill() {
  local dest_name="$1"
  local src_name="${2:-$1}"
  local fallback_name="${3:-}"
  local src="$source_root/plugins/appkit/skills/$src_name"
  if [[ ! -d "$src" && -n "$fallback_name" ]]; then
    src="$source_root/plugins/appkit/skills/$fallback_name"
  fi
  if [[ ! -d "$src" ]]; then
    echo "missing source skill: $src_name" >&2
    exit 1
  fi
  rm -rf "$pack_root/$dest_name"
  mkdir -p "$pack_root"
  cp -R "$src" "$pack_root/$dest_name"
}

write_speckit_agent() {
  local source_agent="$source_root/plugins/appkit/agents/appkit-dev.agent.md"
  if [[ ! -f "$source_agent" ]]; then
    echo "missing source agent: $source_agent" >&2
    exit 1
  fi
  mkdir -p "$pack_root/agents"
  cat >"$pack_root/agents/appkit-dev.md" <<'EOF'
---
name: appkit-dev
description: Builds native macOS AppKit apps in a SpecKit project — Swift 6, Tuist, the macOS 26/27 SDK, behaviour proven by the headless Core. Use for creating an apple target's UI, adding features, designing with Liquid Glass, migrating from UIKit/Catalyst/Objective-C, or any AppKit/Cocoa Swift task. Examples — <example>user: "Build the settings window for the apple target" assistant: "Dispatching appkit-dev to design + implement it against the Core."</example> <example>user: "Make this list use a view-based NSTableView" assistant: "Sending appkit-dev to ground it in appkit-design and wire it."</example>
tools: Read, Write, Edit, Bash, Grep, Glob
---

You build and ship native macOS AppKit code in a **SpecKit** project. You own the loop: spec → design → implement → build & run → **`specify verify`**. The spec-provable behaviour lives in the target's **headless `Core` package** (`@Observable` view models + domain), which `swift test` proves with no Tuist/Xcode/simulator; the AppKit surface (`macOS/`) sits on top.

## Default skills

For the build/run loop and the Apple idioms (view models, `Observations`-driven AppKit controllers, the OpenAPI client, SwiftData), load **ios-development**. For UI/design — control selection, layout, semantic color/typography, Liquid Glass, window sizing, accessibility, **wired to the `sdk-search` / `sdk-api` tools** — load **appkit-design**, and **apple-hig** for the Human Interface Guidelines authority it implements against. For runtime/static reverse-engineering of *other* apps, **appkit-app-inspector** (uitool) and **appkit-private-apis** (headerdump/redump). For process: **implementing-a-spec**, **test-driven-development**, **verification-before-completion**.

## Grounded tools — never guess

- Verify **every** symbol and its macOS availability with **`sdk-api`** before you write it (`sdk-api check NSGlassEffectView.effectIsInteractive`; `sdk-api availability <symbol>`). Don't guess symbol names or `@available` versions.
- For canonical patterns ("how do I build X in AppKit"), query **`sdk-search`** before writing from scratch.
- These CLIs come from the `apple-platform-tools` monorepo (`mise run install` → `~/.local/bin`); they're not pinned per-project. If they're absent, say so and fall back to Apple's first-party docs — never to memory.

## Non-negotiables

- **Swift 6** language mode, strict concurrency. Respect `@MainActor`; AppKit UI is main-actor.
- **Spec binding:** behaviour goes in the `Core` package and is bound with the Swift Testing traits — `@Suite(.spec("<id>"))` / `@Test(.scenario("<sub-id>"))`, the dotted id in the trait, a raw-identifier function name for the description. View controllers carry `// SPEC: manual` or `// SPEC: <vm-id> (deviates: <ui reason>)`. `specify verify` is the gate, and it targets the Core.
- **Accessibility:** an accessibility identifier on every interactive control. Not optional.
- **No hardcoded colors/fonts:** semantic `NSColor`/`NSFont` (`preferredFont(forTextStyle:)`); support light/dark/high-contrast.
- **Gate new APIs** behind `@available` / `if #available` when the deployment target is below the API's introduction.

## Distribution awareness

Release is SpecKit's **deploy subsystem** (`specify deploy add app-store-connect <target>`): Mac App Store / TestFlight need public APIs + App Sandbox; Developer ID (web/Sparkle) is the escape hatch. Private APIs, swizzling, and techniques discovered by inspecting other apps at runtime (**appkit-app-inspector**) *may* fail App Store review case-by-case — surface the trade-off and the Developer-ID alternative; never refuse to implement them.

## Efficiency

Batch file creates/edits in one pass. Don't re-read files you just wrote. Chain dependent shell commands with `&&`. Build and run before claiming done; report what `specify verify` (and any app build) actually passed.
EOF
}

retarget_for_speckit() {
  local files=(
    "$pack_root/appkit-design"
    "$pack_root/appkit-private-apis"
    "$pack_root/appkit-app-inspector"
    "$pack_root/apple-hig"
  )
  find "${files[@]}" -type f -name '*.md' -print0 |
    xargs -0 perl -0pi -e '
      s/^name: appkit-hig$/name: apple-hig/mg;
      s/`appkit-hig`/`apple-hig`/g;
      s/appkit-ui-testing/XCUITest/g;
      s/→ `appkit-modern-input`; `gesture-recognizers-basics`/Use the standard event plumbing, not raw `mouseDown:` overrides/g;
      s/See \*\*`appkit-packaging`\*\* for the full distribution picture\./See **your project'\''s deploy setup (`specify deploy`)** for the full distribution picture./g;
      s/See \*\*`appkit-private-apis`\*\* and \*\*`appkit-packaging`\*\*\./See **`appkit-private-apis`** and **your project'\''s deploy setup (`specify deploy`)**./g;
      s/and `appkit-packaging` for the distribution end/and your project'\''s deploy setup (`specify deploy`) for the distribution end/g;
      s/the `appkit-packaging` cross-reference/the SpecKit deploy cross-reference/g;
      s/The build\/sign\/notarize\/staple mechanics live in \*\*`appkit-packaging`\*\*/The build\/sign\/notarize\/staple mechanics live in **your project'\''s deploy setup (`specify deploy`)**/g;
      s/`appkit-setup` handles this if the monorepo is present/Your environment setup handles this if the apple-platform-tools monorepo is present/g;
      s/^HIG: Typography — (https:\/\/developer\.apple\.com\/design\/human-interface-guidelines\/typography) /HIG: Typography — <$1> /mg;
    '

  perl -0pi -e 's/\n```\n(headerdump \[<options>\])/\n```text\n$1/g' \
    "$pack_root/appkit-private-apis/references/header-dumper.md"
  perl -0pi -e 's/\n```\n(class ~ '\''NSTextField'\'')/\n```text\n$1/g' \
    "$pack_root/appkit-app-inspector/references/filter-drill-and-selectors.md"
}

generate() {
  rm -rf "$pack_root/appkit-hig"
  copy_skill appkit-design
  copy_skill appkit-private-apis
  copy_skill appkit-app-inspector
  copy_skill apple-hig apple-hig appkit-hig
  write_speckit_agent
  retarget_for_speckit
}

if [[ "$mode" == "check" ]]; then
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT
  tmp_pack="$tmp/apple"
  cp -R "$pack_root" "$tmp_pack"
  original_pack="$pack_root"
  pack_root="$tmp_pack"
  generate
  if ! diff -qr "$original_pack" "$tmp_pack"; then
    cat >&2 <<EOF

apple pack drift detected.
Run:
  MAC_DEV_SKILLS_ROOT=$source_root scripts/generate-apple-pack.sh
EOF
    exit 1
  fi
else
  generate
fi
