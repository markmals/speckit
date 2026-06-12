---
name: windows-app-control
description: Use to drive a running WinUI 3 app on Windows for visual verification, UI debugging, and behavioral checks via the winapp CLI (winapp run + winapp ui). Trigger when verifying Windows target UI changes, screenshotting an app state, asserting element values, or simulating clicks/typing in a tight verify-iterate loop. Windows host only — inert on a macOS-hosted agent.
---

# Windows App Control

Recipes for driving a running WinUI 3 app from the `winapp` CLI on Windows. Use these the same way you'd use the `chrome-devtools` CLI on the web side or `ios-simulator-control` on Apple: tight loops of "make change → run → screenshot/assert → verify".

> **Windows host only.** `winapp` is a Windows `winget` package and packaged WinUI apps activate only on Windows. On a macOS-hosted agent this skill is **inert** — there, spec compliance is proven by MSTest at the view-model layer (see `windows-development` → "Verifying"), and GUI / visual / accessibility-runtime checks punt to a human on Windows. Do not load this skill to "verify" Windows behavior you cannot actually run.

> **This is the verification loop, not the spec-test layer.** It is the analog of simulator screenshotting: post-implementation behavioral and visual confirmation. The spec's contract is still proven by `// SPEC:`-tagged MSTest view-model tests. Don't treat a `winapp ui` script as a spec's test of record — it's evidence for `verification-before-completion`, not the thing `specify verify` and `specify scan` read.

## Prerequisites (one-time, Windows)

`winapp`'s skills assume a real toolchain. If a command fails because something's missing, install it — don't work around it:

- **.NET SDK ≥ 8** (`dotnet --list-sdks`) — already managed for build/test.
- **`winapp` CLI** (`winget install Microsoft.WinAppCLI`; keep it current — it's pre-1.0 and ships breaking changes between minors).
- **WinUI 3 templates** (`dotnet new install Microsoft.WindowsAppSDK.WinUI.CSharp.Templates`).
- **Developer Mode** enabled (Settings → System → For developers). Packaged apps won't install without it.

## Run the app and watch it

```powershell
# Build (MSBuild/dotnet), then launch the packaged app and stream debug output.
# --debug-output surfaces first-chance exceptions and crashes back to you —
# without it a crash on launch looks like silence.
winapp run --debug-output
```

The launch prints the running PID (`launched (PID: 12345)`). Capture it — every `winapp ui` call targets the app by `-a <PID>`. Launch in a backgrounded `Bash` call (`run_in_background: true`) so the stream stays attached while you drive the UI in subsequent calls.

**Never** launch the bare `.exe` — a packaged WinUI app must activate through its package identity, or it silently exits.

## Drive and assert: `winapp ui` verbs

`status`, `inspect`, `search`, `get-property`, `get-value`, `screenshot`, `invoke`, `click`, `set-value`, `focus`, `scroll`, `scroll-into-view`, `wait-for`, `list-windows`, `get-focused`. Run `winapp ui --cli-schema` for the full JSON structure, or `winapp ui <verb> --help` for one verb.

`wait-for --value` is the primary assertion — it auto-detects the right UIA pattern per control type (TextBlock→Name, TextBox→Value, ComboBox→Selection, Toggle/CheckBox→Toggle state):

| Assertion         | Command                                                              |
| ----------------- | -------------------------------------------------------------------- |
| Element exists    | `winapp ui wait-for "Id" -a PID -t 3000`                             |
| Exact value       | `winapp ui wait-for "Id" -a PID --value "expected" -t 3000`          |
| Value contains    | `winapp ui wait-for "Id" -a PID --value "words" --contains -t 3000`  |
| Element gone      | `winapp ui wait-for "Id" -a PID --gone -t 3000`                      |
| Specific property | `winapp ui wait-for "Id" -a PID -p IsEnabled --value "True" -t 3000` |
| Click / activate  | `winapp ui invoke "Id" -a PID` (exit 0 = success)                    |
| Set then verify   | `winapp ui set-value "Id" "text" -a PID` then `wait-for --value`     |
| Screenshot        | `winapp ui screenshot -a PID -o path.png`                            |
| Right-click menu  | `winapp ui click "Id" -a PID --right` then `wait-for` the menu item  |

If you wrote the code, you already know the `AutomationId`s from the XAML — skip discovery. If you're verifying code you didn't write, `winapp ui inspect -a <PID> --interactive --json` enumerates the visible tree; read the XAML for `AutomationId`s on flyouts/dialogs/lazy content that `inspect` misses while collapsed.

## Scripted batch verification

For anything beyond a spot check, generate one `ui-tests.ps1` that exercises every requirement in a single pass and writes structured results — faster and repeatable versus interactive poking:

```powershell
param([Parameter(Mandatory)][int]$AppPid)   # NOT $Pid — it's read-only in PowerShell
$pass = 0; $fail = 0; $results = @()
function Test-UI {
    param([string]$Name, [scriptblock]$Script)   # use 'throw' inside $Script to fail, never 'exit'
    try {
        & $Script 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) { $script:pass++; $results += @{ name=$Name; status="PASS" } }
        else { $script:fail++; $results += @{ name=$Name; status="FAIL" } }
    } catch { $script:fail++; $results += @{ name=$Name; status="FAIL"; detail="$_" } }
}

Test-UI "Nav to Settings"   { winapp ui invoke  "NavSettings" -a $AppPid }
Test-UI "Settings loaded"   { winapp ui wait-for "TxtUserName" -a $AppPid -t 3000 }
Test-UI "Set username"      { winapp ui set-value "TxtUserName" "TestUser" -a $AppPid }
Test-UI "Save commits"      { winapp ui invoke  "BtnSave" -a $AppPid }   # commits the TextBox binding
Test-UI "Username persisted"{ winapp ui wait-for "TxtUserName" -a $AppPid --value "TestUser" -t 2000 }

New-Item -ItemType Directory -Force -Path screenshots | Out-Null
winapp ui screenshot -a $AppPid -o screenshots/01-settings.png
Write-Host "Passed: $pass | Failed: $fail"
$results | ConvertTo-Json | Out-File test-results.json
if ($fail -gt 0) { exit 1 } else { exit 0 }
```

Run it with the captured PID; read `test-results.json`. **Cap fix-and-rerun at ~2 cycles** — if the same checks keep failing, report them as known issues rather than thrashing.

### AutomationId coverage gate

Every interactive control needs an `AutomationId` (it's also an accessibility and testability requirement — see `windows-development`). Audit it in the same pass, scoping to the app window so OS pickers don't pollute the result:

```powershell
$els = (winapp ui inspect -a $AppPid --interactive --json | ConvertFrom-Json).elements
$missing = $els | Where-Object { $_.type -match 'Button|TextBox|ComboBox|CheckBox|ToggleSwitch|TabItem' -and -not $_.automationId }
if ($missing) { "Missing AutomationId: " + ($missing.type -join ', ') }
```

### Screenshot review is not optional

UIA assertions pass while the app is visually broken — clipping, overlap, wrong theme, controls bleeding past their container all return green. Screenshot each meaningful state and **look at the PNG**. Fail the run on any of: unintended scrollbars, unexpected text truncation (`…`), sliced hero elements, controls past the right edge, overlapping rows, content not using the available width, wrong theme (Light/Dark/HighContrast) vs. what was asked, broken focus/hover/error states. A failed visual check is a bug — window too small? size it in the `MainWindow` ctor (see `windows-development` → Fluent design idioms, runtime-only).

## Gotchas

- **`set-value` doesn't commit a default `TextBox` binding.** WinUI `x:Bind TwoWay` on `TextBox.Text` updates the view model on `LostFocus`. UIA `set-value` changes the text without firing focus events. Fix the _app_ with `UpdateSourceTrigger=PropertyChanged` (see `windows-development`); otherwise `invoke` a button or `focus` another control after `set-value` to force the commit.
- **File pickers run in a separate `PickerHost` process** — `-a PID` won't find them. `winapp ui list-windows -a PID --json` to get the picker HWND, then target it with `-w <HWND>`.
- **Flyouts/menus appear asynchronously** — a short `Start-Sleep` after the trigger before asserting on items.
- **Verify persistence via the data file, not a relaunch** — killing/relaunching a packaged app from a script is fragile (MSIX registration timing). Read the on-disk data file and assert on its contents.
- **Use `$AppPid`, never `$Pid`** (read-only automatic variable in PowerShell).

## When NOT to use this skill

- **A macOS-hosted agent** — it can't run any of this; verify at the MSTest view-model layer instead (`specify verify windows`).
- **Pure unit / view-model tests** — run those via `dotnet test`; they're the spec-bound layer that `specify verify` reads.
- **One-off state lookups** unrelated to verifying a change.
