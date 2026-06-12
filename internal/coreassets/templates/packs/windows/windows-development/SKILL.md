---
name: windows-development
description: Use when writing or modifying Windows target code. Covers C# + WinUI 3 + XAML + MVVM Toolkit + EF Core idioms, Fluent Design correctness, the Kiota-generated OpenAPI client, the RoslynMcpExtension MCP bridge, and the winapp run/ui verification loop. Complementary to `implementing-a-spec` (process) and `windows-app-control` (driving the running app on Windows).
---

# Windows Development

This skill covers **how to write Windows code** in this repo. For the _workflow_ of implementing a spec, see `implementing-a-spec`. For _what to build_, see the spec. For _driving the running app on Windows_, see `windows-app-control`.

The Windows target is a **WinUI 3 desktop app** that mirrors the reference implementation's behavior idiomatically. The reference implementation is the source of truth for behavior; code does not need to converge.

## Stack at a glance

| Concern             | Choice                                                            | First-party docs                                                                                                                                                 |
| ------------------- | ----------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Language            | C#                                                                | [learn.microsoft.com/dotnet/csharp](https://learn.microsoft.com/en-us/dotnet/csharp/)                                                                            |
| Concurrency         | async/await                                                       | [learn.microsoft.com/dotnet/csharp/asynchronous-programming](https://learn.microsoft.com/en-us/dotnet/csharp/asynchronous-programming/)                          |
| State management    | MVVM Toolkit (`CommunityToolkit.Mvvm`)                            | [learn.microsoft.com/dotnet/communitytoolkit/mvvm](https://learn.microsoft.com/en-us/dotnet/communitytoolkit/mvvm/)                                              |
| UI                  | WinUI 3 + XAML                                                    | [learn.microsoft.com/windows/apps/winui/winui3](https://learn.microsoft.com/en-us/windows/apps/winui/winui3/)                                                    |
| Design language     | Fluent Design                                                     | [learn.microsoft.com/windows/apps/design](https://learn.microsoft.com/en-us/windows/apps/design/)                                                                |
| On-device database  | EF Core (SQLite)                                                  | [learn.microsoft.com/ef/core](https://learn.microsoft.com/en-us/ef/core/)                                                                                        |
| Networking          | `HttpClient`                                                      | [learn.microsoft.com/dotnet/fundamentals/networking/http/httpclient](https://learn.microsoft.com/en-us/dotnet/fundamentals/networking/http/httpclient)           |
| API client          | Kiota (typed client over `HttpClient` — don't hand-roll requests) | [learn.microsoft.com/openapi/kiota](https://learn.microsoft.com/en-us/openapi/kiota/)                                                                            |
| Tests               | MSTest                                                            | [learn.microsoft.com/dotnet/core/testing/unit-testing-csharp-with-mstest](https://learn.microsoft.com/en-us/dotnet/core/testing/unit-testing-csharp-with-mstest) |
| UI automation / run | `winapp` CLI (`winapp run` + `winapp ui`)                         | [github.com/microsoft/winappcli](https://github.com/microsoft/winappcli) — Windows host only; see `windows-app-control`                                          |
| Formatter           | `dotnet format`                                                   | [learn.microsoft.com/dotnet/core/tools/dotnet-format](https://learn.microsoft.com/en-us/dotnet/core/tools/dotnet-format)                                         |
| Linter              | StyleCop Analyzers                                                | [github.com/DotNetAnalyzers/StyleCopAnalyzers](https://github.com/DotNetAnalyzers/StyleCopAnalyzers)                                                             |
| Package manager     | NuGet                                                             | [nuget.org](https://www.nuget.org/)                                                                                                                              |
| Auth                | Clerk (identity / token)                                          | [clerk.com/docs](https://clerk.com/docs)                                                                                                                         |

Microsoft doesn't publish a `/llms.txt` for any of these. Use WebFetch against the canonical doc URLs above when you need to look something up. For canonical control samples, fetch the [WinUI Gallery](https://github.com/microsoft/WinUI-Gallery) and [Windows Community Toolkit](https://github.com/CommunityToolkit/Windows) source directly rather than guessing property names.

## The Client layer depends on the backend mode

How the app reaches its data is set by the project's backend (see `specs/ARCHITECTURE.md` → "Backend modes"):

- **OpenAPI** — a typed client generated by Kiota over **`HttpClient`**; never assemble requests or model the wire protocol by hand.
- **Convex** — Convex's client SDK where one exists, otherwise its HTTP API (there is no first-party C# Convex SDK — OpenAPI mode is usually the better fit on Windows).
- **No API** — no client at all; **EF Core (SQLite)** is the source of truth.

In the remote modes, **EF Core** is a local-first cache (not a second backend) and identity flows through **Clerk**, whose token the client attaches. See the target's source directory for the wrapper that exposes idiomatic call sites.

## Idioms (read these before writing code)

### MVVM Toolkit view models

The view model is the primary spec target. Use `ObservableObject` partial classes with `[ObservableProperty]` and `[RelayCommand]` — the MVVM Toolkit source generators produce the change-notification and command boilerplate at compile time.

```cs
// SPEC: vm.items.list
public sealed partial class ItemsListViewModel : ObservableObject
{
    private readonly IApiClient client;

    [ObservableProperty]
    public partial IReadOnlyList<Item> Items { get; set; } = [];

    [ObservableProperty]
    public partial LoadStatus Status { get; set; } = LoadStatus.Idle;

    [ObservableProperty]
    public partial string? ErrorMessage { get; set; }

    public ItemsListViewModel(IApiClient client) => this.client = client;

    [RelayCommand]
    private async Task LoadAsync()
    {
        Status = LoadStatus.Loading;
        try
        {
            Items = await client.ListItemsAsync();
            Status = LoadStatus.Loaded;
        }
        catch (Exception error)
        {
            Status = LoadStatus.Error;
            ErrorMessage = error.Message;
        }
    }
}
```

- **Use the partial-_property_ form** (`[ObservableProperty] public partial T Name { get; set; }`), not the older field form (`[ObservableProperty] private T name;`). The field form emits `MVVMTK0045` under the AOT/trimming settings WinUI uses. The partial-property form needs **C# 13** (the default on the .NET 9+ SDK — no `LangVersion` override needed at the template's .NET 10 target) and **CommunityToolkit.Mvvm 8.4+**. Inline initializers are supported (`… { get; set; } = [];`); use them for defaults rather than the constructor.
- `IApiClient` is the thin wrapper over the Kiota-generated output — it exposes idiomatic async methods (`ListItemsAsync()`), not the generated request-builder chain. Dependencies arrive via the constructor; no service locator, no static singletons.
- `[RelayCommand]` generates an `ICommand` (`LoadCommand`) — never hand-roll `ICommand`. Don't replace an `ObservableCollection<T>`; mutate it (`.Clear()` + re-add).

### XAML views are dumb; no logic in code-behind

Views bind to the view model via `{x:Bind}` (compiled, type-checked, AOT-safe; preferred) or `{Binding}` (reflection-based — avoid). The code-behind stays empty except for wiring — `InitializeComponent()` and DI of the view model. No business logic in `.xaml.cs`.

```xml
<!-- SPEC: manual -->
<ListView ItemsSource="{x:Bind ViewModel.Items, Mode=OneWay}" />
```

Mark XAML and code-behind `// SPEC: manual` (no cross-target behavioral contract), or `// SPEC: <vm-id> (deviates: <ui reason>)` for a UI-level deviation. The reverse pointer for behavior lives on the **view model**, never the view.

### async/await for all I/O

Every network and database call is `async`/`await`. **Never** `.Result` or `.Wait()` (or `.GetAwaiter().GetResult()`) — they deadlock the UI thread. There are no synchronous fallbacks; if an API offers only a callback, wrap it once with `TaskCompletionSource` and forget about it. `async Task` for async methods; `async void` only for event handlers.

### WinUI framework footguns

- **Never add `<UseWPF>true</UseWPF>` or `<WindowsPackageType>None</WindowsPackageType>` to `*.csproj`** — both silently corrupt the build. `PresentationCore` / `System.Windows.Media.Imaging` crash the WinUI XAML compiler (as of the current Windows App SDK); use `Microsoft.UI.Xaml.Media.Imaging.BitmapImage` instead. (Link the [migration docs](https://learn.microsoft.com/en-us/windows/apps/windows-app-sdk/migrate-to-windows-app-sdk/migrate-to-windows-app-sdk-ovw) rather than asserting the crash as a permanent fact.)
- **Marshal to the UI thread with `DispatcherQueue.TryEnqueue(...)`** — there is no `Application.Current.Dispatcher` in WinUI 3.
- **For P/Invoke, prefer CsWin32** (source-generated bindings from a `NativeMethods.txt`) over hand-rolled `[DllImport]` — it gives you the correct, AOT-safe signatures.

### Tests at the view-model layer

Per `specs/CONVENTIONS.md`, MSTest bindings use `[TestProperty]` attributes — metadata on the class and method, not encoded in the method name. The spec ID stays out of the test description.

```cs
[TestClass]
[TestProperty("spec", "vm.items.list")]
public class ItemsListViewModelTests
{
    [TestMethod]
    [TestProperty("scenario", "scenario.items.list.empty")]
    public async Task ShowsEmptyStateWhenNoItemsExist()
    {
        var client = new FakeApiClient(returning: []);
        var viewModel = new ItemsListViewModel(client);

        await viewModel.LoadCommand.ExecuteAsync(null);

        Assert.AreEqual(0, viewModel.Items.Count);
        Assert.AreEqual(LoadStatus.Loaded, viewModel.Status);
    }
}
```

- `[TestProperty("spec", "<id>")]` on the class carries the spec ID — the engine reads this for the scenario join.
- `[TestProperty("scenario", "<sub-id>")]` on the method carries the scenario sub-ID. The sub-ID lives in the attribute, not in the method name or description.
- Test the view model directly — invoke `[RelayCommand]`-generated commands (`LoadCommand`) rather than the private method.
- Run with `format junit` so `specify verify windows` can parse the report.

### EF Core for the local cache; Kiota for the network

EF Core (SQLite) is the local-first cache. A `DbContext` owns the on-device tables; reads serve from it for offline and fast first paint, then reconcile against the backend. The Kiota client is the only thing that crosses the wire — don't route persistence through it, and don't treat the local database as a second source of truth.

## Fluent design idioms (static — verifiable on any host)

`specs/DESIGN_SYSTEM.md` owns the cross-target token / component / icon parity (the WinUI brush names, Segoe Fluent Icons, `AutomationProperties` roles). This section adds the WinUI-specific _correctness mechanics_ that the parity table doesn't cover. Everything below is checkable by **reading C#/XAML**, so it pays off even on a macOS-hosted agent. Runtime-only concerns are fenced at the end. Do not build a rival design rulebook here — when in doubt about a design choice, read [Fluent Design](https://learn.microsoft.com/en-us/windows/apps/design/) and check `DESIGN_SYSTEM.md`.

- **Theming.** Use `{ThemeResource Brush}` at usage sites (it updates on theme switch); use `{StaticResource}` for the redirect _inside_ a theme dictionary. A `ResourceKey` must end in `Brush` (target the `SolidColorBrush`, not the `Color`). Define all three variants — `Light`, `Dark`, `HighContrast` — never `Default`. No hardcoded colors (`#FF0000`, `Color="Blue"`).
- **High Contrast.** Only the 8 `SystemColor*Brush` pairs belong in HC dictionaries; no opacity, no accent colors, no regular WinUI brushes. Use an empty HC dict when the WinUI defaults already suffice.
- **Typography.** Use the built-in styles (`Caption` / `Body` / `BodyStrong` / `Subtitle` / `Title` / `TitleLarge` / `Display` `TextBlockStyle`), never a raw `FontSize`. `SemiBold`, never `Bold`; 12px floor.
- **Spacing.** Multiples of the 4px grid (4, 8, 12, 16, 24, 32, 48). `ControlCornerRadius` / `OverlayCornerRadius`, never hardcoded radii. `RowSpacing` / `ColumnSpacing` over spacer elements.
- **Data binding.** `{x:Bind}` over `{Binding}`. **Set `Mode` explicitly** — `x:Bind` defaults to `OneTime`, which silently never updates dynamic data (the usual "blank screen after load" cause). `x:DataType` on every `DataTemplate`. Prefer `x:Bind` function-converters over `IValueConverter`; **never `Converter={x:Null}`** — it throws at runtime. A `TextBox` `TwoWay` binding needs `UpdateSourceTrigger=PropertyChanged`, or the view model updates only on `LostFocus` and UIA `set-value` won't commit.
- **Accessibility.** `AutomationProperties.AutomationId` on every interactive control (Button, TextBox, ComboBox, ToggleSwitch, ListView, NavigationViewItem); `AutomationProperties.Name` on icon-only controls. Use semantic controls (`Button`, `HyperlinkButton`), not a clickable `Border`/`TextBlock`. Set attached properties via the **static setter** — `AutomationProperties.SetAutomationId(btn, "BtnSave")` — never object-initializer syntax (`new Button { AutomationProperties = { … } }` does not compile).

When reviewing WinUI code (the code-quality stage of `implementing-a-spec`), the net-new static checks beyond the idioms above are: missing `x:DataType` on a `DataTemplate`, an `x:Bind` to dynamic data without `Mode=OneWay`, hardcoded user-facing strings instead of `x:Uid` / `ResourceLoader`, and information conveyed by color alone. Where i18n/RTL implies a product behavior, that's a spec-level decision (write a spec), not a lint rule.

**Runtime-only (human-on-Windows; a macOS-hosted agent cannot verify these):** window sizing (WinUI 3 has no `SizeToContent` — size in the `MainWindow` ctor; `AppWindow.Resize` takes _physical pixels_, so multiply by the monitor's DPI scale), Mica/Acrylic backdrops, `ThemeShadow`, connected animations, live theme/HighContrast switching, and screen-reader (Narrator/NVDA) behavior. Screenshot-verify these on Windows via `windows-app-control`. Cosmetic gaps the cross-target spec doesn't cover → open a defect spec for the windows target.

## File layout (within the target's source)

The configured `source` dir for the windows target follows this structure:

```
src/
├── App/                            ← App entry, MainWindow, root navigation, DI setup
├── Features/<Feature>/             ← Feature-scoped: ViewModel + its View (.xaml/.xaml.cs)
├── Domain/                         ← Plain C# types and validation (one file per domain.* spec)
├── Data/                           ← EF Core DbContext, entities, local-first cache
└── Client/                         ← generated Kiota client + idiomatic wrapper (HttpClient transport)
```

## Driving the C# toolchain from the agent (MCP)

The **RoslynMcpExtension** exposes the Roslyn compiler API over MCP — the C# code model, diagnostics, and symbol search — so the agent reasons about the project structurally (types, references, errors) instead of by text search.

- Repo: [github.com/sailro/RoslynMcpExtension](https://github.com/sailro/RoslynMcpExtension).
- It is **per-machine** — it requires a Visual Studio / Roslyn host running on the machine — so it belongs in your **user/local MCP config (`~/.claude/` or `.mcp.local.json`)**, not the committed `.mcp.json`, which must work for everyone (including contributors who don't run it).
- Once connected, prefer it for symbol lookup, reference search, and diagnostics over `rg`/grep. Fall back to `dotnet build` and `dotnet test` when the bridge isn't available — a macOS-hosted agent won't have it.

## Verifying

Two execution contexts, and which one is load-bearing depends on where the agent runs.

**Any host (including a macOS-hosted agent) — load-bearing.** `dotnet build` then `dotnet test` exercises view-model behavior; it runs anywhere .NET runs and is the verification of record for spec compliance. Run `specify verify windows` to have the engine run the target's configured test command and join scenarios to test results. Before declaring done, run `dotnet format --verify-no-changes` and confirm StyleCop is clean (it surfaces as build warnings/errors). The static C#/XAML checks above — binding `Mode`, `x:DataType`, hardcoded colors/strings, `.Result`/`.Wait`, missing `AutomationId`, UI types leaking into view models — are all reviewable by reading the source; do them here regardless of host.

**On Windows — the GUI/visual loop.** Active Windows development is expected to happen on Windows. There, launch and drive the running app with the **`winapp` CLI**: `winapp run --debug-output` starts the packaged app and streams crashes/exceptions back so you can see them instead of staring at silence; `winapp ui` asserts element state, drives controls, and screenshots. This is covered by the **`windows-app-control`** skill. A macOS-hosted agent cannot run it — there, GUI / visual / accessibility-runtime checks punt to a human on Windows. **UIA assertions pass while the app is visually broken** (clipping, overlap, wrong theme), so a screenshot review is part of the loop, not optional.

**Packaged-execution invariants (Windows).** A packaged WinUI app launches through its packaged identity — never run the bare `.exe`, never set `<WindowsPackageType>None</WindowsPackageType>`, never delete `Package.appxmanifest`. Target `x64`/`ARM64`, never `AnyCPU` (`0x8007000B` is a platform mismatch). Developer Mode must be enabled. Quick build-error reads: blank window after launch → an `x:Bind` defaulted to `OneTime`, add `Mode=OneWay`; `XLS0414` → missing `xmlns`; `XDG0062` → an `x:Bind` path not found on the view model.

> Distribution (MSIX packaging, code signing, Microsoft Store) is a per-product, human-gated, Windows-only operation outside the spec→test→impl loop; this template ships no ship-stage runbook. A few durable facts if you reach it: the certificate `Publisher` must match the manifest `Identity.Publisher`; sign with `--timestamp` or signatures expire with the cert; Store submission is browser-based (no first-party CLI submit). See [winappcli](https://github.com/microsoft/winappcli).

## When to invoke a more specific skill

- About to write tests? → `test-driven-development`
- About to claim work is done? → `verification-before-completion`
- Verifying Windows UI **on Windows** (run, screenshot, drive controls)? → `windows-app-control`
- Debugging something unexpected? → `systematic-debugging`
- Implementing a spec end-to-end? → `implementing-a-spec`

## Commit

Land focused, atomic commits as the work hits natural boundaries — typically per spec ID, per view-model + its tests, or per cohesive refactor. Follow the scoped-commits convention (`specify gate scope`).

Windows-specific notes:

- **`.csproj` and NuGet changes go alone.** Edits to `.csproj`, `Directory.Packages.props`, or package references belong in their own commit (`chore: add <package>` or `chore: bump <package>`). Don't bundle with feature code.
- **Generated Kiota output rides with the codegen commit.** When you regenerate the client, commit the generated `Client/` output together with the OpenAPI/codegen change that produced it — never hand-edit it.
- **Don't commit build outputs.** `**/bin/` and `**/obj/` are gitignored.
