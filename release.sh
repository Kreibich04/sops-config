#!/usr/bin/env bash
# Render the changelog with towncrier and open a release PR.
#
# Usage: ./release.sh X.Y.Z
#
# Merging the resulting PR into main updates the tracked VERSION file, which
# .github/workflows/tag-release.yml picks up to tag the merge commit. That
# tag push (authenticated as the RELEASE_TOKEN secret, not GITHUB_TOKEN, so
# it can trigger other workflows) fires .github/workflows/release.yml, which
# builds and publishes the cross-platform release binaries.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$repo_root"

if [ $# -ne 1 ]; then
  echo "usage: $0 X.Y.Z" >&2
  exit 1
fi
version="$1"

if ! [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "error: version must be in X.Y.Z format, got '$version'" >&2
  exit 1
fi
tag="v$version"
branch="release/$tag"

if [ ! -f towncrier.toml ]; then
  echo "error: towncrier.toml not found; run this script from the repo root" >&2
  exit 1
fi

if command -v towncrier >/dev/null 2>&1; then
  towncrier_cmd=(towncrier)
elif command -v pipx >/dev/null 2>&1; then
  towncrier_cmd=(pipx run towncrier)
else
  echo "error: towncrier not found. Install it with 'pipx install towncrier' (or 'pip install --user towncrier')." >&2
  exit 1
fi

command -v gh >/dev/null 2>&1 || {
  echo "error: gh CLI not found. Install from https://cli.github.com/ and run 'gh auth login'." >&2
  exit 1
}
gh auth status >/dev/null 2>&1 || {
  echo "error: gh CLI is not authenticated. Run 'gh auth login' first." >&2
  exit 1
}

if [ -n "$(git status --porcelain)" ]; then
  echo "error: working tree is not clean. Commit or stash changes first." >&2
  exit 1
fi

current_branch="$(git rev-parse --abbrev-ref HEAD)"
if [ "$current_branch" != "main" ]; then
  echo "error: must be run from the main branch (currently on '$current_branch')" >&2
  exit 1
fi

git fetch origin main --quiet
if [ "$(git rev-parse HEAD)" != "$(git rev-parse origin/main)" ]; then
  echo "error: local main is not in sync with origin/main. Pull or push first." >&2
  exit 1
fi

if git rev-parse -q --verify "refs/tags/$tag" >/dev/null ||
  git ls-remote --exit-code --tags origin "refs/tags/$tag" >/dev/null 2>&1; then
  echo "error: tag $tag already exists" >&2
  exit 1
fi

if ! find changelog.d -maxdepth 1 -type f ! -name '.gitkeep' | grep -q .; then
  echo "error: no changelog fragments found in changelog.d/. Add at least one before releasing." >&2
  exit 1
fi

echo "==> creating branch $branch"
git checkout -b "$branch"

echo "==> bumping VERSION to $version"
echo "$version" >VERSION

echo "==> rendering changelog with towncrier"
"${towncrier_cmd[@]}" build --version "$version" --yes

git add VERSION CHANGELOG.md
git add -A changelog.d
git commit -m "Release $tag"

echo "==> pushing $branch"
git push -u origin "$branch"

pr_body_file="$(mktemp)"
trap 'rm -f "$pr_body_file"' EXIT
awk -v hdr="## [$version]" '
  $0 ~ "^## \\[" {
    if (capture) exit
    if (index($0, hdr) == 1) { capture = 1; next }
    next
  }
  capture { print }
' CHANGELOG.md >"$pr_body_file"

echo "==> opening release PR"
gh pr create \
  --base main \
  --head "$branch" \
  --title "Release $tag" \
  --body-file "$pr_body_file"

echo "==> done. Merging that PR will trigger the release pipeline (see .github/workflows/tag-release.yml)."
