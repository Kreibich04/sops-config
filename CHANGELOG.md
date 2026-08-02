## [0.0.3] - 2026-08-02

### Features

- A subdirectory rule's `path_regex` now matches the same way it would as a root pattern scoped to just that subtree: unanchored by default (matches anywhere under the directory, not just directly inside it), or anchored to the top of the directory with a leading `^`. Any `path_regex` (root or subdirectory) that doesn't start with `^` now gets a warning, since it's easy to underestimate how broadly an unanchored substring search can match.
- `sops-config.yaml` is now decoded with strict (unknown-field-rejecting) YAML parsing, and validation now also rejects empty/duplicate group names, users with no `pgp`/`age` keys, rules with no `groups`, duplicate keys within a user, and `pgp`/`age` key values that don't look like real keys.

### Bug Fixes

- Add an explicit `permissions: contents: read` to `ci.yml` so the workflow's `GITHUB_TOKEN` isn't left with the broader default permissions (flagged by GitHub code scanning).
- Fix a privilege-escalation path where a `sops-config.yaml` in any subdirectory could declare a user in an existing group (e.g. `admin`) and have that user's keys silently added to root-level and sibling rules using that group. User/group visibility is now scoped by directory ancestry: root-declared users are visible everywhere, but a subdirectory's users are only visible to rules in that subdirectory and its descendants.
- Fixed `tag-release.yml` leaving a `GITHUB_TOKEN` credential override in local git config that silently overrode the `RELEASE_TOKEN`-authenticated tag push, which is why `release.yml` never fired automatically for the `v0.0.1`/`v0.0.2` tags.
- `encrypted_regex` is now optional on a rule, matching vanilla SOPS: omitting it encrypts the entire file, instead of `generate`/`validate` failing with "encrypted_regex is required". This is the common case for a directory holding nothing but secrets (e.g. loaded whole via `secretGenerator`).
- `generate` now writes `.sops.yaml` atomically (temp file in the same directory, then rename), so an interrupted or crashed run can no longer leave a truncated file behind.

### Documentation

- Documented `go install` and cobra's built-in shell completion in the README, and added a `checksums.txt` to each release so downloaded binaries can be verified.


## [0.0.2] - 2026-08-02

### Bug Fixes

- Release workflow can now be manually re-run via `workflow_dispatch`, so a release can be rebuilt if the tag push doesn't trigger it automatically.


## [0.0.1] - 2026-08-02

### Features

- Add the initial `sops-config` CLI (`generate` and `validate`) that merges `sops-config.yaml` files scattered across a directory tree into a single, correctly path-scoped `.sops.yaml`.

### Misc

- Add a tag-triggered release pipeline that cross-compiles binaries for Linux/macOS/Windows (amd64/arm64) and publishes them to a GitHub Release, plus a `--version` flag reporting the build's tag.
