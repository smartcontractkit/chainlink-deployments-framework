#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Create, push, and optionally publish a GitHub release for operations-gen.

Usage:
  ./tools/operations-gen/release-tag.sh v0.1.0 [--push] [--release] [--allow-dirty]

Examples:
  ./tools/operations-gen/release-tag.sh v0.1.0
  ./tools/operations-gen/release-tag.sh v0.1.0 --push
  ./tools/operations-gen/release-tag.sh v0.1.0 --release

Notes:
  - Tag format is always: tools/operations-gen/<version>
  - --release implies --push and uses GitHub CLI (gh)
  - Release notes are scoped to commits touching tools/operations-gen
  - Use this tag for go install:
    go install github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen@v0.1.0
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" || $# -lt 1 ]]; then
  usage
  exit 0
fi

version="$1"
shift

push_tag="false"
create_release="false"
allow_dirty="false"

for arg in "$@"; do
  case "$arg" in
    --push) push_tag="true" ;;
    --release) create_release="true" ;;
    --allow-dirty) allow_dirty="true" ;;
    *)
      echo "Unknown argument: $arg" >&2
      usage
      exit 1
      ;;
  esac
done

if [[ "$create_release" == "true" ]]; then
  push_tag="true"
fi

if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([\-+][0-9A-Za-z\.-]+)?$ ]]; then
  echo "Invalid version '$version'. Expected semver like v0.1.0" >&2
  exit 1
fi

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

if [[ "$allow_dirty" != "true" ]] && [[ -n "$(git status --porcelain)" ]]; then
  cat <<'EOF' >&2
Working tree is not clean.
Commit or stash changes first, or pass --allow-dirty if intentional.
EOF
  exit 1
fi

if [[ "$(git rev-parse --abbrev-ref HEAD)" != "main" ]]; then
  echo "You are not on main. Checkout main before tagging." >&2
  exit 1
fi

# Fetch only main; avoid syncing all tags because unrelated local tags can
# conflict with remote tags and fail the release flow.
git fetch --no-tags origin main

local_main_sha="$(git rev-parse HEAD)"
remote_main_sha="$(git rev-parse origin/main)"
if [[ "$local_main_sha" != "$remote_main_sha" ]]; then
  echo "Local main is not at origin/main. Pull/rebase before tagging." >&2
  exit 1
fi

tag="tools/operations-gen/$version"

if git rev-parse "$tag" >/dev/null 2>&1; then
  echo "Tag already exists locally: $tag" >&2
  exit 1
fi

if git ls-remote --tags origin "refs/tags/$tag" | grep -q "$tag"; then
  echo "Tag already exists on origin: $tag" >&2
  exit 1
fi

git tag -a "$tag" -m "operations-gen $version"
echo "Created tag: $tag"

if [[ "$push_tag" == "true" ]]; then
  git push origin "$tag"
  echo "Pushed tag: $tag"
fi

if [[ "$create_release" == "true" ]]; then
  if ! command -v gh >/dev/null 2>&1; then
    echo "GitHub CLI (gh) is required for --release." >&2
    exit 1
  fi

  if gh release view "$tag" >/dev/null 2>&1; then
    echo "GitHub release already exists for tag: $tag" >&2
    exit 1
  fi

  # Query tags from origin so changelog range does not depend on local tag state.
  # This still works when main was fetched with --no-tags.
  prev_opsgen_tag="$(
    (
      git ls-remote --tags --refs --sort='-v:refname' origin "refs/tags/tools/operations-gen/v*" \
        | awk -v current="$tag" '{sub("^refs/tags/", "", $2); if ($2 != current) print $2}' \
        | head -n 1
    ) || true
  )"

  release_range="$tag"
  if [[ -n "$prev_opsgen_tag" ]]; then
    release_range="$prev_opsgen_tag..$tag"
  fi

  notes_file="$(mktemp)"
  trap 'rm -f "$notes_file"' EXIT

  {
    echo "## operations-gen changes"
    if [[ -n "$prev_opsgen_tag" ]]; then
      echo
      echo "Changes since \`$prev_opsgen_tag\`."
    else
      echo
      echo "Initial release."
    fi
    echo

    if ! git log --no-merges --pretty=format:'- %s (%h)' "$release_range" -- tools/operations-gen; then
      true
    fi
    echo
    echo
    echo "## Install"
    echo
    echo "\`\`\`bash"
    echo "go install github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen@$version"
    echo "\`\`\`"
  } >"$notes_file"

  gh release create "$tag" \
    --title "operations-gen $version" \
    --notes-file "$notes_file"
  echo "Created GitHub release for tag: $tag"
fi

