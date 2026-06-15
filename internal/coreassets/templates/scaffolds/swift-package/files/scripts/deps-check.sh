#!/usr/bin/env bash
# Local dependency-update gate.
#
# Runs Renovate against the working tree in local/dry-run mode — it NEVER opens
# PRs or writes anything; it just reports which dependencies (npm packages, Go
# modules, Swift packages, GitHub Actions, mise tools, …) have newer versions
# available. Wired into `mise run check` so updates surface as ongoing work
# instead of as autonomously-opened Dependabot PRs.
#
# Advisory only: always exits 0. Available updates are a heads-up, not a failure,
# and a Renovate/network hiccup must not break the rest of `check`.
set -uo pipefail

if ! command -v renovate >/dev/null 2>&1; then
    echo "deps: renovate not found — enable the gate with: mise use 'npm:renovate@latest' jq"
    exit 0
fi

# Renovate needs a github.com token to look up Actions / Go-module / Swift-package
# release versions (npm needs none). Reuse the user's existing gh auth when
# present; anonymous still works for npm, just rate-limited on github.com.
github_token="$(gh auth token 2>/dev/null || true)"

log="$(mktemp -t renovate.XXXXXX)"
trap 'rm -f "$log"' EXIT

GITHUB_COM_TOKEN="$github_token" \
RENOVATE_ONBOARDING=false \
LOG_LEVEL=debug \
LOG_FORMAT=json \
    renovate --platform=local --dry-run=full >"$log" 2>&1
status=$?

if [ "$status" -ne 0 ]; then
    echo "deps: renovate exited $status (skipping the update check this run)"
    grep -iE '"level":(50|60)|FATAL|ERROR' "$log" | head -5
    exit 0
fi

# All available updates live in the single "packageFiles with updates" debug
# record: walk every dependency that carries a non-empty updates[] array and
# emit one row per update — updateType, ecosystem, name, current → target.
# Everything Renovate found is surfaced; scope what it *looks at* in renovate.json.
updates="$(jq -r '
    select(.msg == "packageFiles with updates")
    | [.. | objects | select(.depName? and (.updates? | type == "array") and ((.updates | length) > 0))]
    | .[] as $dep
    | $dep.updates[]
    | "\(.updateType // "?")\t\($dep.datasource // "-")\t\($dep.depName)\t\($dep.currentValue // $dep.currentVersion // "-")\t\(.newVersion // .newValue)"
' "$log" | sort -u | sort -t"$(printf '\t')" -k2,2 -k1,1 -k3,3)"

if [ -z "$updates" ]; then
    echo "deps: ✓ all dependencies up to date"
    exit 0
fi

count="$(printf '%s\n' "$updates" | grep -c '')"
echo "deps: ⚠ $count update(s) available — local Renovate dry-run, no PRs opened"
echo
# Majors get their own flagged section (each is a breaking-change decision, not
# routine work); the in-range minor/patch updates follow, grouped by ecosystem.
printf '%s\n' "$updates" | awk -F'\t' '
    {
        type[NR] = $1; eco[NR] = $2; name[NR] = $3; cur[NR] = $4; tgt[NR] = $5
        if ($1 == "major") majors++
        else inrange[$2]++
    }
    END {
        if (majors > 0) {
            printf "  major — review before upgrading (%d)\n", majors
            for (i = 1; i <= NR; i++)
                if (type[i] == "major")
                    printf "    %-7s %-34s %s → %s\n", eco[i], name[i], cur[i], tgt[i]
            print ""
        }
        prev = ""
        for (i = 1; i <= NR; i++) {
            if (type[i] == "major") continue
            if (eco[i] != prev) {
                if (prev != "") print ""
                printf "  %s (%d)\n", eco[i], inrange[eco[i]]
                prev = eco[i]
            }
            printf "    %-6s %-34s %s → %s\n", type[i], name[i], cur[i], tgt[i]
        }
    }'
echo
echo "  In-range bumps: apply them with your stack's package manager, then re-run \`mise run deps\`."
echo "  Majors are listed separately — review each one's changelog before upgrading."
exit 0
