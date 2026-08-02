# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A CLI (`sops-config`) that generates a single [SOPS](https://github.com/getsops/sops)
`.sops.yaml` from small `sops-config.yaml` files scattered throughout a
directory tree, so subdirectories in a monorepo can each own the encryption
rules for their own secrets without a centrally-maintained regex list. See
README.md for the full user-facing spec (config format, CLI flags, path
scoping rules, validation table) — it's kept accurate and detailed, read it
before making behavioral changes.

## Commands

```sh
go build -o sops-config ./cmd/sops-config   # build
go test ./...                               # run all tests
go test ./internal/merge/... -run TestName  # run a single test
go vet ./...
gofmt -l .                                  # CI fails if this prints anything; use `gofmt -w .` to fix
```

CI (`.github/workflows/ci.yml`) runs gofmt check, vet, test (with coverage),
then build, in that order, on every PR and push to `main`.

## Architecture

```
cmd/sops-config/       entrypoint
internal/config/       sops-config.yaml types, loading, tree discovery
internal/merge/        user merge, path-regex scoping, rule resolution
internal/sopsyaml/     .sops.yaml rendering
internal/cli/          cobra commands (generate, validate)
```

`generate` and `validate` both call the same `internal/merge.Run(root)`
pipeline (see `internal/merge/pipeline.go`) — this is the key invariant to
preserve: the two commands must never be able to drift in what they check.
`validate` is meant to be safe to run in CI/pre-commit, so it must reject
everything `generate` would reject, and only that.

The pipeline runs in five stages — discovery, user merge, path scoping, rule
resolution, rendering — each in its own package/file as listed above.
Discovery-stage problems (malformed YAML, missing required field, no root
config) are fail-fast: abort immediately, report only the first one found.
Everything from user-merge onward is aggregated: every diagnostic across
every file is collected and reported together in one run. This split is
deliberate (a config that doesn't parse can't be reasoned about further) —
don't collapse the two error-handling strategies into one.

Path scoping (`internal/merge/pathregex.go`) prefixes subdirectory rules
with `^<dir>/`, escaping the directory component via `regexp.QuoteMeta` —
that outer `^dir/` anchor is always load-bearing, since SOPS matches
`path_regex` as an unanchored substring search and omitting it would let a
directory name matching elsewhere in the tree accidentally trigger a rule
scoped to a different subtree. What comes after the `dir/` prefix depends
on whether the author's own fragment starts with `^`: if it does, the `^`
is stripped and the fragment is glued on directly (anchored to the top of
the directory, e.g. `^secrets/.*` in `muc/` → `^muc/secrets/.*`, matching
only `muc/secrets/...`); if it doesn't, a `.*` is inserted before it so the
fragment is an unanchored search *within* the directory (e.g. `secrets/.*`
in `muc/` → `^muc/.*secrets/.*`, matching `muc/secrets/...` and
`muc/anything/secrets/...`). This makes a subdirectory fragment behave the
same way it would if the author had run it directly as a root pattern
against just their own subtree — root and subdirectory patterns share the
same "unanchored unless you write `^`" semantics, they're just scoped
differently. `internal/merge/rules.go` warns (not errors) on any
`path_regex`, root or subdirectory, that doesn't start with `^`.

User/group visibility (`internal/merge/registry.go`) is scoped by directory
ancestry, not global. `ResolvedUser.Dirs` tracks every directory a user was
(identically) declared in, and `UserRegistry.UsersInGroup(group, fromDir)`
only returns users visible from `fromDir` — declared at the root, or
declared in `fromDir` or one of its ancestors. This is the fix for a
privilege-escalation path where a `sops-config.yaml` in an unrelated
subdirectory could declare a user in an existing group (e.g. `admin`) and
have that user's keys silently added to every rule using that group,
including root-level and sibling rules. **Do not go back to a single global
byGroup map** — group resolution must always be filtered by the querying
rule's directory.

Rendering (`internal/sopsyaml/render.go`) must stay deterministic and
timestamp-free — re-running `generate` over unchanged input is expected to
produce byte-identical output (keys sorted, rules stably sorted by
priority).

## Releasing

Full mechanics are in README.md's "Releasing" section. Key points if asked
to cut a release:

- Changelog fragments go in `changelog.d/` (towncrier), named
  `+<slug>.<type>.md`; `release.sh` refuses to run with none present.
- `release.sh X.Y.Z` must run from `main` with a clean tree; it opens a
  release PR, it does not merge it.
- Merging that PR bumps `VERSION`, which `tag-release.yml` picks up to tag
  the merge commit and push the tag using the `RELEASE_TOKEN` secret (a
  push authenticated with the default `GITHUB_TOKEN` cannot trigger other
  workflows, which is why a PAT is required here).
- That tag push is *supposed* to trigger `release.yml` (cross-compile +
  publish binaries), but this has been unreliable in practice — both
  `v0.0.1` and `v0.0.2` tag pushes via `RELEASE_TOKEN` completed
  successfully without ever triggering `release.yml` (confirmed via
  `gh run list --workflow=release.yml` showing zero runs). Root cause is
  unconfirmed; likely `RELEASE_TOKEN`'s scopes/repo access. If a release
  goes out without binaries, don't assume automation will catch up —
  check `gh run list --workflow=release.yml`, and if empty, dispatch it
  manually: `gh workflow run release.yml --ref vX.Y.Z` (works because
  `release.yml` has a `workflow_dispatch` trigger as of `v0.0.2`; tags cut
  before that don't have it baked in and can't be dispatched directly).
