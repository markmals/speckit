---
name: appkit-packaging
description: Use when distributing the macOS AppKit app outside the Mac App Store — Developer ID signing (codesign), notarization (notarytool), stapling, and shipping a signed .dmg / .pkg. The direct-download / Sparkle side; the TestFlight + Mac App Store CI path is the `app-store-connect` deploy kind (`specify deploy add app-store-connect`).
---

# AppKit Packaging (Developer ID)

Ship the macOS app **outside the App Store**: sign it with your **Developer ID Application** certificate, **notarize** it with Apple, **staple** the ticket, and wrap it in a signed `.dmg` or `.pkg`. This is the direct-download / Sparkle channel.

The App Store / TestFlight pipeline is a **different beast** — different certs, App Sandbox mandatory, App Review instead of notarization, no Developer ID. That path is the **`app-store-connect` deploy kind**: run `specify deploy add app-store-connect`, which wires the App Store Connect API-key flow into CI. Don't sign a store build with Developer ID or vice-versa — the two signings are mutually exclusive.

This skill is the **release surface**, not the inner loop. To build/run the app while developing, see `appkit-dev-workflow`. To prove behavior, run the headless `Core` suite (`ios-development`, `test-driven-development`). Before claiming a release works, see `verification-before-completion`.

## What you build from

Notarization wants a **Release** build of the `.app`, produced by Tuist (the generated `.xcodeproj` / `Derived` are gitignored — regenerate, never commit). From the AppKit app dir:

```bash
mise run -C macOS build      # Release product lands under macOS/Derived/Build/Products/Release/<AppName>.app
```

Everything below operates on that `.app`. You sign the bundle, not the source.

## The one-time setup

- **Developer ID Application certificate** in your login keychain. List identities:
  ```bash
  security find-identity -v -p codesigning
  ```
  If you see only Apple Development / Apple Distribution, you don't have a Developer ID cert yet — create one in the Apple Developer portal (Certificates → **Developer ID Application**) and download it. `appkit-setup` points you there when a packaging step reports no identity. `TEAMID` is your 10-character team identifier.
- **Notary credentials**, cached once so later submits carry no secrets. Create an **app-specific password** at appleid.apple.com (or use an App Store Connect API key), then:
  ```bash
  xcrun notarytool store-credentials "AC_NOTARY" \
    --apple-id "you@example.com" --team-id "TEAMID" \
    --password "abcd-efgh-ijkl-mnop"   # app-specific password (or --key/--key-id/--issuer)
  ```

## The chain: sign → notarize → staple → verify → package

**Sign, inside-out, hardened runtime, timestamped.** Notarization *requires* the hardened runtime (`--options runtime`) and a secure timestamp (`--timestamp`) — miss either and Apple returns `Invalid`. Sign **nested code first** (frameworks, helpers, XPC services), the `.app` bundle **last**. Apple discourages `--deep` for production — sign each nested Mach-O explicitly.

```bash
xattr -cr <AppName>.app            # strip quarantine xattrs first
IDENTITY="Developer ID Application: Your Name (TEAMID)"

find <AppName>.app/Contents/Frameworks \( -name "*.framework" -o -name "*.dylib" \) | while read -r item; do
  codesign --force --options runtime --timestamp --sign "$IDENTITY" "$item"
done
codesign --force --options runtime --timestamp \
  --entitlements <AppName>.entitlements --sign "$IDENTITY" <AppName>.app
codesign --verify --deep --strict --verbose=2 <AppName>.app   # 0 = good
```

Keep entitlements minimal — only add a hardened-runtime exception (JIT, disable-library-validation for plug-ins) when a real crash demands it, then re-sign.

**Notarize and wait.** notarytool wants an archive — zip with `ditto` (never Finder's "Compress" or plain `zip`; they mangle bundle symlinks and fail), or submit the `.dmg`/`.pkg` directly.

```bash
ditto -c -k --keepParent <AppName>.app <AppName>.zip
xcrun notarytool submit <AppName>.zip --keychain-profile "AC_NOTARY" --wait
```

`--wait` blocks until `Accepted` or `Invalid`. On `Invalid`, pull the log — it names the exact unsigned/un-hardened binary:

```bash
xcrun notarytool log <submission-id> --keychain-profile "AC_NOTARY" log.json
```

**Staple** the ticket so Gatekeeper passes **offline**, then **verify like a first-run user on another Mac would**:

```bash
xcrun stapler staple <AppName>.app          # staple the artifact you ship (.app, then repackage — or the dmg/pkg)
spctl -a -vvv -t install <AppName>.app      # expect: accepted, source=Notarized Developer ID
xcrun stapler validate <AppName>.app
```

(You can't staple a `.zip` — staple the `.app` inside it, or the `.dmg` / `.pkg`.)

**Package — and sign the wrapper too.** A `.dmg` (drag-to-Applications) for most apps; a `.pkg` when you need an installer (privileged step, login item):

```bash
# DMG
create-dmg --volname "<AppName>" --app-drop-link 450 160 <AppName>.dmg <AppName>.app  # brew install create-dmg
codesign --force --timestamp --sign "$IDENTITY" <AppName>.dmg
xcrun stapler staple <AppName>.dmg

# PKG (installer cert is a DIFFERENT cert: Developer ID Installer)
pkgbuild --component <AppName>.app --install-location /Applications <AppName>-comp.pkg
productbuild --package <AppName>-comp.pkg --sign "Developer ID Installer: Your Name (TEAMID)" <AppName>.pkg
xcrun notarytool submit <AppName>.pkg --keychain-profile "AC_NOTARY" --wait
xcrun stapler staple <AppName>.pkg
```

The download must be trusted **as one unit** — sign (and for a dmg, staple) the wrapper, not just the app inside it.

## Key rules

- **Hardened runtime + timestamp are mandatory** for notarization (`--options runtime --timestamp`).
- **Sign inside-out**, bundle last; avoid `--deep` in production.
- **Zip for notarization with `ditto -c -k --keepParent`** — anything else corrupts the bundle.
- **Staple the artifact you actually ship.** A stapled ticket is what lets Gatekeeper pass with no network.
- **`.dmg` uses the Application cert; `.pkg` uses the Developer ID *Installer* cert** — different certs.
- **Never tell users to `xattr -d` your download or disable Gatekeeper** as a "fix" — that masks a real signing/notarization failure.

## Troubleshooting

| Symptom | Fix |
|---|---|
| `errSecInternalComponent` on codesign | Identity missing or keychain locked — `security unlock-keychain`; confirm with `find-identity`. |
| notarytool returns `Invalid` | `notarytool log <id>` — almost always an unsigned/un-hardened nested binary; re-sign it inside-out. |
| `not signed with a valid Developer ID certificate` | You signed with Apple Development, not Developer ID Application. |
| `code object is not signed at all` (nested) | A framework/helper was missed — sign every nested Mach-O before the bundle. |
| "App is damaged and can't be opened" on another Mac | Not notarized/stapled, or zipped with Finder — notarize + staple, repackage with `ditto` / `create-dmg`. |
| `spctl` says `source=Unnotarized Developer ID` | Notarization didn't complete or wasn't stapled — finish the staple step. |
| `stapler staple` → `Error 65 / could not find ticket` | Ticket hasn't propagated, or you stapled before `Accepted` — wait, re-staple. |

## CI

A tag-triggered **GitHub Actions** release runs the same chain on a `macos-15` runner: import the Developer ID `.p12` (base64 repo secret) into a **temporary keychain** at job start, build via `mise run -C macOS build`, then sign → notarize → staple → build the `.dmg`, and `security delete-keychain` on `if: always()`. Authenticate notarization with an **App Store Connect API key** (`--key`/`--key-id`/`--issuer`) — no human MFA, revocable — rather than an Apple-ID password. Attach the stapled `.dmg` to the GitHub Release.

The store/TestFlight CI lane is separate — that's `specify deploy add app-store-connect`, not this Developer-ID job.

## A grounding note (entitlements / availability)

Don't guess an entitlement key or an `@available` gate. Verify symbols and availability with the external `sdk-api` CLI and search canonical patterns with `sdk-search` (built into `~/.local/bin` via `mise run install` — referenced, never vendored), e.g. `sdk-api check NSGlassEffectView` before relying on a version. `appkit-design` covers the in-app non-negotiables (semantic colors/typography, accessibility identifiers, content-derived sizing, explicit Liquid Glass) that the build you sign must already satisfy.

## When to step out

- Private APIs in the build you're shipping → `appkit-private-apis` (Developer ID is the escape hatch from App Store review; you own OS-update breakage).
- Build won't produce a clean Release `.app` → `appkit-dev-workflow`, then `systematic-debugging`.
- About to call the release done → `verification-before-completion` (run the Core suite, then notarize-verify on a clean Mac, not just your dev box).

## Commit

The release pipeline lives in CI config and packaging scripts — keep those in their own commit, separate from feature code. Never commit certs, `.p12`, `.p8`, or notary passwords; CI reads them from repo secrets. Use the scoped-commits convention (`specify gate scope`).
