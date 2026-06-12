# Web scaffold — tooling preview

**Status:** ✅ **Approved 2026-06-12.** Framework **TanStack Start** (React 19) ·
toolchain **raw Vite 8** (no Vite+) · Convex **optional (`--with convex`)** ·
extras off by default · RSC off by default. Exact versions pinned at build time.
Ready to build.

## Proposed stack (bleeding edge, mid-2026)

| layer | proposed | current state / why |
| --- | --- | --- |
| **framework** | **TanStack Start** (React 19) | RC, API stable (v1.167+, ~6M wk downloads); RSC support landed Apr 2026 (experimental, client-first React Flight). Matches SpecKit's `web` pack. |
| **toolchain** | **Vite 8** (Rolldown + Oxc + Lightning CSS) | single Rust pipeline since Mar 2026 — 4–20× faster, no dev/prod divergence. *Or* **Vite+ (`vp`)** — your own tooling (you ship the homebrew formula; `create-sprinkles` requires it). |
| **styling** | **Tailwind CSS v4.3** | CSS-first config; v4.3 May 2026. |
| **testing** | **Vitest 4** (Browser Mode, Playwright-driven) | Browser Mode + visual regression now stable; its `junit` reporter feeds the engine, and the binding harness rides on it. |
| **data** *(optional)* | **Convex** via `--with convex` | your backend of choice; ties into the deferred `contracts` story. |
| **extras** *(optional)* | Clerk (auth), Motion (animation) via `--with` | from the `web` pack; off by default. |

## The binding harness (makes `specify verify web` green on arrival)

- Vitest configured with the **`junit` reporter** → the `report` path.
- One example story under `features/` + a bound test using **`// [scenario.<id>]`**
  above `it(…)`, and a `// SPEC:` pointer on the unit it covers.
- Result: `specify scan` and `specify verify web` both pass on the fresh target —
  the agent extends a green loop instead of wiring one.

## Resulting `specs.json` target

```json
"web": {
  "stack": "web",
  "command": "pnpm -C apps/web test --run",
  "format": "junit",
  "report": "apps/web/junit.xml",
  "source": "apps/web/src"
}
```

## Decided (2026-06-12)

1. **Framework** — **TanStack Start** (React 19). RSC off by default.
2. **Toolchain** — **raw Vite 8** (Rolldown + Oxc + Lightning CSS); no Vite+.
3. **Convex** — **optional `--with convex`**; the base scaffold stays lean.
4. **Extras** — Clerk / Motion as `--with` add-ons; off by default.

## Notes

- **RSC stays off by default** — it's experimental in TanStack Start RC; opt-in.
- React 19 + the React Compiler are assumed (stable).
- Exact versions are pinned when we build, not now.

Sources: [TanStack Start RSC](https://tanstack.com/blog/react-server-components) ·
[Vite 8 / Rolldown](https://vite.dev/blog/announcing-vite7) ·
[Tailwind v4.3](https://tailwindcss.com/blog/tailwindcss-v4-3) ·
[Vitest 4 Browser Mode](https://voidzero.dev/posts/announcing-vitest-4).
