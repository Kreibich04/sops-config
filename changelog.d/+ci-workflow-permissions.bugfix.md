Add an explicit `permissions: contents: read` to `ci.yml` so the workflow's `GITHUB_TOKEN` isn't left with the broader default permissions (flagged by GitHub code scanning).
