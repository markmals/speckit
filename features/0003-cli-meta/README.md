# 0003 — CLI meta commands

The small, always-present commands: `version` and `check`. Specced from the
oracle (D14) with the fork's divergences baked in — `--json` on every command
(D2) and no "GitHub Spec Kit" banner (D1).

| Spec | Capability |
| --- | --- |
| [`story.cli.version`](stories/cli.version.md) | Report the binary version, plain and `--json`. |
| [`story.cli.check`](stories/cli.check.md) | Report required-tool availability, plain and `--json`. |
| [`story.cli.upgrade`](stories/cli.upgrade.md) | `self upgrade` — in-place, checksum-verified, fail-safe. |
