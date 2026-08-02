## [0.0.1] - 2026-08-02

### Features

- Add the initial `sops-config` CLI (`generate` and `validate`) that merges `sops-config.yaml` files scattered across a directory tree into a single, correctly path-scoped `.sops.yaml`.

### Misc

- Add a tag-triggered release pipeline that cross-compiles binaries for Linux/macOS/Windows (amd64/arm64) and publishes them to a GitHub Release, plus a `--version` flag reporting the build's tag.
