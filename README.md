# sops-config

A CLI that generates a single [SOPS](https://github.com/getsops/sops) `.sops.yaml`
from small `sops-config.yaml` files scattered throughout a directory tree.

## Purpose

Hand-maintaining `.sops.yaml` in a monorepo gets painful once different
subdirectories need to be encrypted for different sets of people: the file
becomes one long, centrally-owned list of regexes and keys that nobody near
the actual secrets wants to touch.

`sops-config` lets you instead drop a small `sops-config.yaml` next to the
secrets it governs. One is required at the root of the tree to define the
baseline set of users and rules; any subdirectory can add its own
`sops-config.yaml` to extend the rule set for just that subtree, without
being able to affect anything outside it. Running the tool merges every
`sops-config.yaml` it finds into one correctly-scoped `.sops.yaml` at the
root, in the format SOPS expects.

## Usage

### Install

Download a prebuilt binary from the
[latest release](https://github.com/Kreibich04/sops-config/releases/latest):
grab the archive matching your OS/arch (e.g.
`sops-config_vX.Y.Z_linux_amd64.tar.gz`, `..._darwin_arm64.tar.gz`,
`..._windows_amd64.zip`), extract it, and put the `sops-config` binary on
your `PATH`.

```sh
# example: Linux amd64
curl -L -o sops-config.tar.gz \
  https://github.com/Kreibich04/sops-config/releases/latest/download/sops-config_vX.Y.Z_linux_amd64.tar.gz
tar -xzf sops-config.tar.gz
sudo mv sops-config /usr/local/bin/
```

### Build from source (development)

Only needed if no release covers your platform, or you're working on
`sops-config` itself / need an unreleased change:

```sh
go build -o sops-config ./cmd/sops-config
```

### Write a `sops-config.yaml`

Every discovery root needs exactly one `sops-config.yaml`. It defines
`users` (each with group memberships and PGP/Age keys) and `rules` (each
naming a `path_regex`, an `encrypted_regex`, a `priority`, and the `groups`
allowed to decrypt matching files):

```yaml
users:
  - name: "Admin One"
    groups:
      - admin
    keys:
      pgp:
        - "AABB11"
      age:
        - "AABB11"

rules:
  - path_regex: .*/secrets/.*
    encrypted_regex: '^(data|stringData)$'
    comment: "default secrets rule"
    priority: 100
    groups:
      - admin
```

Any subdirectory may contain its own `sops-config.yaml` to add more users
and/or rules. A subdirectory rule's `path_regex` is relative to that
subdirectory, and matches the same way it would if you'd written it as a
root pattern scoped to just that subtree — see [Path scoping](#path-scoping)
below for exactly how it's combined with the directory prefix, and what a
leading `^` changes.

Users declared at the root are visible to every rule in the tree. Users
declared in a subdirectory are visible only to rules in that subdirectory
and its descendants — a subdirectory's `sops-config.yaml` can never change
who a rule outside its own subtree resolves to (see
[User visibility](#user-visibility) below).

### Generate `.sops.yaml`

```sh
sops-config generate --root .
```

Walks `--root` (default `.`), merges every `sops-config.yaml` found, and
writes `<root>/.sops.yaml`. Flags:

- `--root, -r` — directory to search (default `.`)
- `--output, -o` — output path (default `<root>/.sops.yaml`)
- `--force` — write `.sops.yaml` even if no rules resolved

If any error-level diagnostic is found, `generate` prints it and exits
non-zero **without writing** — it never produces a partial or wrong
`.sops.yaml`. The write itself is atomic (write to a temp file in the same
directory, then rename over the target), so an interrupted `generate` can
never leave a truncated `.sops.yaml` behind either.

### Validate without writing

```sh
sops-config validate --root .
```

Runs the exact same discovery/merge/validation pipeline as `generate`, but
never writes output. Exits non-zero if any error-level diagnostic is found,
or if zero rules resolved (same as `generate`, and overridable the same way
with `--force`) — so `validate` never passes something `generate` would
refuse. Intended for CI or a pre-commit hook, so a broken `sops-config.yaml`
is caught before it silently fails to grant (or worse, silently grants)
decrypt access.

## Technical details

### Package layout

```
cmd/sops-config/       entrypoint
internal/config/       sops-config.yaml types, loading, tree discovery
internal/merge/        user merge, path-regex scoping, rule resolution
internal/sopsyaml/     .sops.yaml rendering
internal/cli/          cobra commands (generate, validate)
```

`generate` and `validate` both call the same `internal/merge.Run(root)`
pipeline, so they can never drift in what they check.

### Merge pipeline

1. **Discovery** (`internal/config.Discover`): walks `--root`, finds every
   file named `sops-config.yaml`, in lexical order (root's own config
   first, then subdirectories in walk order). Errors if no config exists at
   the root. YAML is decoded with `KnownFields(true)`, so an unrecognized
   field (e.g. a typo'd `comnent:`) is rejected rather than silently
   ignored. Malformed YAML, an unknown field, or a missing/invalid field
   (see [Validation rules](#validation-rules)) aborts discovery
   immediately, reporting just that one file — fix it and rerun to see
   what's next. This stage is deliberately fail-fast: a config file that
   doesn't even parse can't be reasoned about well enough to keep walking
   past it.
2. **User merge** (`internal/merge.MergeUsers`): users from every discovered
   config are combined into a registry, keyed by `name`, that also tracks
   which directory(ies) declared each user. A name redeclared with
   identical groups/keys is a silent no-op that extends that user's
   visibility to the redeclaring directory too (lets a shared user be
   conveniently repeated across directories that need it). A name
   redeclared with *differing* groups or keys is an error — key-material
   conflicts are never resolved by silently picking one side.

   <a id="user-visibility"></a>User (and therefore group) visibility is
   scoped by directory ancestry, not global: a rule can only resolve a
   `groups` entry to users declared at the root, or declared in the rule's
   own directory or one of its ancestors. A user declared only in
   `foo/bar/sops-config.yaml` is invisible to rules in `foo/`, in a sibling
   `foo/baz/`, or at the root — so a subdirectory can never grant itself,
   or anyone else outside its own subtree, access to a rule it doesn't
   own. If a rule references a group that has no users visible from its
   directory, that's the same "group has no matching users" error as
   referencing a group that doesn't exist anywhere.
3. **Path scoping** (`internal/merge.ScopePathRegex`): a root-level rule's
   `path_regex` is used unmodified — SOPS matches it as an *unanchored
   substring search* over the whole repo, same as vanilla `.sops.yaml`. A
   subdirectory rule's `path_regex` gets the same substring-search
   treatment, just scoped to its own directory: the fragment matches
   whether it's directly inside that directory or nested arbitrarily
   deeper, exactly as if the author had run that same fragment as a root
   pattern against only their own subtree.

   `muc/sops-config.yaml` with `path_regex: secrets/.*` →
   `^muc/.*secrets/.*` (matches `muc/secrets/x` *and*
   `muc/anything/secrets/x`)

   Writing a leading `^` opts into anchoring the fragment to the top of the
   directory instead, the same way `^` anchors a root pattern to the top of
   the repo:

   `muc/sops-config.yaml` with `path_regex: ^secrets/.*` →
   `^muc/secrets/.*` (matches `muc/secrets/x` only, not
   `muc/anything/secrets/x`)

   The directory component is escaped with `regexp.QuoteMeta` so directory
   names containing regex metacharacters are matched literally, and the
   composed pattern is always anchored at the start with `^dir/...`:
   without that, a directory named e.g. `muc` appearing anywhere else in
   the tree could accidentally match a rule meant only for `muc/`. No
   trailing `$` is added — the author decides whether a rule covers one
   file or an entire subtree. Any `path_regex` (root or subdirectory) that
   doesn't start with `^` gets a warning, since it's easy to underestimate
   how broadly an unanchored substring search can match.
4. **Rule resolution** (`internal/merge.BuildRules`): for each rule, its
   `groups` are resolved to the union of matching *visible* users (see
   step 2), whose PGP/Age keys are flattened, deduplicated, and
   alpha-sorted (for deterministic, diff-friendly output). Rules are then
   stably sorted by `priority` ascending — SOPS evaluates `creation_rules`
   top-to-bottom and uses the first match, so a lower `priority` number
   means a rule is placed, and therefore tried, earlier. Because the
   pre-sort order is already root-then-subdirectory/in-file order,
   equal-priority rules keep a deterministic tie-break automatically.
5. **Rendering** (`internal/sopsyaml.Render`): maps resolved rules to
   `creation_rules` entries (`pgp`/`age` as comma-joined strings) and
   prepends a fixed, timestamp-free "generated file, do not edit" header —
   so re-running `generate` over unchanged input produces byte-identical
   output. `generate` writes the result atomically (temp file + rename).

### Validation rules

| Condition | Severity |
|---|---|
| Malformed `path_regex` / `encrypted_regex` | error |
| No `sops-config.yaml` at the root | error |
| Duplicate user name with conflicting groups/keys | error |
| Rule references a group with zero *visible* matching users | error |
| Rule resolves to no PGP and no Age keys | error |
| Duplicate `priority` across rules | warning |
| `path_regex` doesn't start with `^` (unanchored substring search) | warning |
| Zero rules resolved overall | error (both commands refuse unless `--force`) |

These are all raised during the **merge** stage, after discovery has already
succeeded, and are aggregated rather than fail-fast: a single run reports
every one of these found across every file in one pass.

Discovery-stage problems abort the run immediately, before merging starts,
so only the first one found is reported — fix it and rerun to uncover the
next. These include: malformed or unrecognized-field YAML; a missing
required field (`name`, `path_regex`, `encrypted_regex`, `priority`); no
`sops-config.yaml` at the root; an empty or duplicate group name; a user
with no `pgp`/`age` keys at all; a rule with no `groups`; and a `pgp`/`age`
key that doesn't look like a real key (an empty string, a duplicate within
the same user, or a value that doesn't match the expected shape — an
even-length hex string for `pgp`, an `age1...` recipient for `age`).

### Dependencies

- [`spf13/cobra`](https://github.com/spf13/cobra) — CLI framework
- [`gopkg.in/yaml.v3`](https://github.com/go-yaml/yaml) — YAML parsing and
  rendering

## Releasing

Changelog entries are managed with [towncrier](https://towncrier.readthedocs.io/).
Every change worth mentioning gets a fragment file in `changelog.d/`, named
`+<slug>.<type>.md` (e.g. `changelog.d/+scoped-regex.feature.md`), where
`<type>` is one of `feature`, `bugfix`, `doc`, `removal`, or `misc` (see
`towncrier.toml`). The fragment's contents become a bullet in the rendered
`CHANGELOG.md`.

To cut a release:

```sh
./release.sh X.Y.Z
```

Run from `main` with a clean working tree, this renders the accumulated
`changelog.d/` fragments into `CHANGELOG.md` via `towncrier build`, bumps the
`VERSION` file, commits both, pushes a `release/vX.Y.Z` branch, and opens a
PR via `gh pr create`. Requires `towncrier` (or `pipx`, which the script
falls back to) and an authenticated `gh` CLI.

Merging that PR is what actually ships the release:

1. The merge changes `VERSION` on `main`, which triggers
   [`tag-release.yml`](.github/workflows/tag-release.yml). It tags the merge
   commit `vX.Y.Z` and pushes the tag using the `RELEASE_TOKEN` secret
   rather than the default `GITHUB_TOKEN` — a tag pushed with the default
   token cannot trigger other workflows, so a real token is required for the
   next step to fire. `RELEASE_TOKEN` must be a PAT (fine-grained,
   "Contents: Read and write" on this repo) added as a repository secret.
2. That tag push triggers [`release.yml`](.github/workflows/release.yml),
   which cross-compiles and publishes the release binaries.
