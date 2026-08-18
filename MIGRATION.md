# Migrating to the stack-agnostic SpecKit

SpecKit is no longer a project scaffolder. It is a spec engine: it verifies
behavior against a spec library, and it adopts into a project that already
exists. This document migrates a repo that was set up under the old layout —
stacks, scaffolds, packs, deploy manifests, GitHub-only work tracking — onto
the current surface. Everything here is concrete; follow it top to bottom.

## If you only do one thing

Replace the old register flow with one `specify target add`. Where you used to
run something like:

```sh
# old — gone
specify target register api --stack go-service --dir cmd/api
```

now spell the wiring out explicitly (nothing is seeded from a stack anymore):

```sh
specify target add api --dir cmd/api --format gotest \
  --report .speckit/api.gotest.json \
  --source cmd/api --source internal \
  --command "go test -json ./cmd/... ./internal/... > .speckit/api.gotest.json" \
  --bindings scoped
```

`target add` writes one entry into `.speckit/specs.json` and nothing else — no
files rendered, no installs, no platform question. `--format`, `--report`, and
`--source` are required; `--command` is optional when the report already exists;
`--bindings scoped` is what a project with pre-existing plain unit tests almost
always wants; `--reference` marks the reference target in a multi-target repo.
Worked examples per report format: [docs/adopting.md](docs/adopting.md).

## Config: `.speckit/specs.json`

**You don't have to touch the file to keep working.** An unmigrated config
loads fine: retired keys are ignored with a single printed notice (never a
failure), and the file is normalized to the current schema the next time
SpecKit writes it. Migrating by hand just silences the notice.

Per target:

- **Drop `stack`.** It selected a scaffold and a platform skill pack; both are
  gone. The key is now ignored — it does nothing.
- **Drop `deploy`.** The deploy manifest (kinds, `op://` secret references) is
  gone. The key is now ignored — it does nothing.
- **`dir` is new.** The target's root, relative to the project root.
  `specify target add --dir` writes it; a target that omits it is treated as
  rooted at the project root. Informational — nothing is generated into it —
  but it records what the target *is* now that no platform label does.

Top level:

- **`version` is now `2`.** Any write normalizes the file to `2`; an older
  version loads with a notice.
- **`reference_target` is new and optional.** Names the target other targets
  match when a spec is ambiguous across them. Set it with
  `specify target add <name> … --reference`. When unset, no target is
  privileged.
- **`work` is new and optional.** Selects the work-tracking provider
  (`markdown` · `beads` · `github-projects` · `none`); absent means the
  `markdown` provider on `WORK.md`. See
  [docs/work-providers.md](docs/work-providers.md).

Everything else (`agent`, `paths`, and the targets' `command`/`format`/
`report`/`source`/`bindings`/`product`) is unchanged. Full schema:
[docs/config.md](docs/config.md).

## Removed commands, and what replaces each

| Removed | Replacement |
| --- | --- |
| `specify target add --stack …` (the scaffolding form) | Nothing — SpecKit no longer generates code. Bring your own project and register it with the new `specify target add` (above). |
| `specify target register` | `specify target add` — the same job (record existing code as a target, write nothing), without the platform question. All wiring is explicit flags. |
| `specify packs` | Nothing. Skills that prescribe a stack are the adopting project's business; SpecKit ships no platform skill packs. Delete the projected ones (next section). |
| `specify deploy` | Your project's own task runner / CI. SpecKit no longer writes deploy workflows or records deploy manifests. |
| `specify secrets` | Your project's own secret tooling. SpecKit no longer reads 1Password `op://` references and no longer writes GitHub Actions secrets. |
| `specify protect` | The manual `gh` branch-protection recipe (below). |
| `specify issues create` | `specify work create "<title>" --type defect` |
| `specify issues list` / `specify issues close <n>` | `specify work list` / `specify work move <id> done` |
| `specify work discover` | `specify work create` (with `--spec <spec-id>` to record what it advances) |

The old `specify work` subcommands were GitHub-Projects-only; the new `specify
work` (`ready` / `create` / `claim` / `move` / `list`) drives whichever
provider the `work` block selects, with `markdown` — a committed `WORK.md` —
as the zero-setup default. The GitHub Projects board is still available as the
`github-projects` provider.

### Branch protection without `specify protect`

`specify protect` provisioned a ruleset; do it once with `gh` instead. The
required status-check context is **`verify / verify`** when your `ci.yml` calls
the reusable gate workflow from a job named `verify` (caller job / reusable
job); add your project's own quality-job contexts alongside it if you have
them:

```sh
gh api --method POST repos/{owner}/{repo}/rulesets --input - <<'JSON'
{
  "name": "speckit-gate",
  "target": "branch",
  "enforcement": "active",
  "conditions": { "ref_name": { "include": ["~DEFAULT_BRANCH"], "exclude": [] } },
  "rules": [
    { "type": "pull_request",
      "parameters": {
        "required_approving_review_count": 0,
        "dismiss_stale_reviews_on_push": false,
        "require_code_owner_review": false,
        "require_last_push_approval": false,
        "required_review_thread_resolution": false
      } },
    { "type": "required_status_checks",
      "parameters": {
        "strict_required_status_checks_policy": true,
        "required_status_checks": [
          { "context": "verify / verify" }
        ]
      } },
    { "type": "non_fast_forward" }
  ]
}
JSON
```

`gh api` fills in `{owner}/{repo}` from the current repo; `~DEFAULT_BRANCH`
targets the repo's default branch; adjust `required_approving_review_count` to
taste. Re-running it on an existing same-named ruleset fails — edit the ruleset
in Settings → Rules → Rulesets, or `gh api --method PUT
repos/{owner}/{repo}/rulesets/<id>` with the same body. The UI path and the
full gate breakdown are in [docs/ci-gating.md](docs/ci-gating.md).

## Delete the projected packs

`specify packs` projected platform skill packs (and, for Claude, per-stack
agents) into your agent's directories. Nothing removes them for you — delete
them from an already-projected repo by hand. The projection locations, per
agent integration:

| Agent | Skills landed in | Per-stack agents landed in |
| --- | --- | --- |
| `claude` | `.claude/skills/<skill>/` | `.claude/agents/` |
| `codex` / `generic` | `.agents/skills/<skill>/` | — (never projected) |
| `copilot` | `.github/skills/<skill>/` | — (never projected) |

Delete the pack skill directories for whichever stacks you had (each is a
directory named after the skill, containing `SKILL.md` and possibly
`references/`):

- **web:** `web-development`, `web-verification`
- **website:** `website-development`
- **apple:** `ios-development`, `ios-simulator-control`, `apple-hig`,
  `appkit-design`, `appkit-setup`, `appkit-dev-workflow`, `appkit-code-review`,
  `appkit-ui-testing`, `appkit-packaging`, `appkit-migration`,
  `appkit-private-apis`, `appkit-app-inspector`, `appkit-modern-input`,
  `appkit-launch-continuity`, `appkit-liquid-glass`, `appkit-session-report`
- **android:** `android-development`, `android-emulator-control`
- **go-cli:** `go-cli-development`
- **node-cli:** `node-cli-development`

And the one per-stack agent (Claude only): `.claude/agents/appkit-dev.md`.

Leave everything else in the skills dir alone — the `speckit-*` command skills
and the process-discipline skills (`test-driven-development`,
`verification-before-completion`, `adversarial-review`, `systematic-debugging`,
`implementing-a-spec`, `brainstorming-feature`, `writing-user-stories`,
`managing-memory`) are still projected by `init` and still current. The five
review subagents in `.claude/agents/` (`spec-reviewer`, `test-gap-finder`,
`drift-hunter`, `handoff-builder`, `visual-verifier`) also stay.

## Correct the projected prose (the reference target)

Older projections hardcoded **"Web is the reference target"** in the rules and
skills prose. That sentence is wrong now: the reference target is the
`reference_target` key in `.speckit/specs.json`, and when it's unset **no
target is privileged**. Fix an already-projected repo by re-projecting:

```sh
specify target add <your-reference-target> … --reference   # set the key first
specify init --here --integration <agent>                  # re-project rules/skills/commands
```

Re-projection overwrites the generated rules, skills, and command files with
prose that reads `reference_target` instead of naming a platform (re-running
`init` never clobbers accumulated agent memory — the seed `MEMORY.md` is
written skip-if-exists). If you maintain hand-edited copies of any projected
file, grep them for the hardcoded sentence and reword against the config key.

## Replace the scaffolded CI workflow

The old `target add` dropped a `.github/workflows/ci.yml` (a `quality` job
running the scaffold's task-runner checks + a `verify` job) into your repo, and
`deploy add` dropped `deploy.yml`. Neither is written anymore, and the gate no
longer sets up any toolchain. Your CI is yours; the stack-neutral replacement
shape is:

- **The gate job** — one line, calling the reusable workflow:

  ```yaml
  # .github/workflows/ci.yml — yours
  on: { pull_request: {} }
  jobs:
    verify:
      uses: markmals/speckit/.github/workflows/gate.yml@v1
      with:
        target: web
  ```

  It runs `specify scan` → the test-edit firewall → `specify verify <target>`
  (which runs your target's configured `command`) → `specify parity --gate`,
  each annotating failures inline in the PR. If your target's `command` needs
  toolchain or dependency setup beyond what the runner ships, run the composite
  action `markmals/speckit/gate@v1` as a step inside your own job, after your
  own checkout (`fetch-depth: 0`) and setup steps.

- **Your quality checks** (formatting, linting, type checks) — your own jobs,
  written by you, required alongside the gate if you want them gating.

- **Deploys** — your own workflows, on your own triggers. SpecKit has no
  opinion.

Full CI shapes and the branch-protection recipe:
[docs/ci-gating.md](docs/ci-gating.md).

## The scaffolded `.github/` defect surface

The old `target add` also dropped `ISSUE_TEMPLATE/defect.yml`,
`ISSUE_TEMPLATE/config.yml`, `PULL_REQUEST_TEMPLATE.md`, and `CODEOWNERS` into
your repo. They are plain files you now own: keep them if they serve you,
delete them if not. Defect intake through SpecKit is
`specify work create "<title>" --type defect` on whatever work provider you
configure.
