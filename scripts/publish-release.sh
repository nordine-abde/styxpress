#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RELEASE_DIR="$ROOT_DIR/dist/releases"
TARGET="${STYXPRESS_RELEASE_TARGET:-all}"
TITLE=""
NOTES=""
NOTES_FILE=""
DRAFT=0
PRERELEASE=0
SKIP_BUILD=0
TAG=""

usage() {
    cat <<'USAGE'
Usage:
  ./scripts/publish-release.sh <tag> [options]

Examples:
  ./scripts/publish-release.sh v0.1.0
  ./scripts/publish-release.sh v0.1.0 --prerelease --notes-file RELEASE_NOTES.md
  ./scripts/publish-release.sh v0.1.0 --target linux/amd64

Options:
  --target <goos/goarch|all>  Build target passed to scripts/build-release.sh.
                             Defaults to all.
  --title <title>             GitHub release title. Defaults to "Styxpress <tag>".
  --notes <text>              GitHub release notes. Defaults to "Release <tag>.".
  --notes-file <path>         Read GitHub release notes from a file.
  --draft                     Create a draft GitHub release.
  --prerelease                Mark the GitHub release as a prerelease.
  --skip-build                Package and publish existing dist/releases output.
  -h, --help                  Show this help.

The script requires the GitHub CLI. Run `gh auth login` before publishing.
Set STYXPRESS_RELEASE_ALLOW_DIRTY=1 to bypass the clean working tree check.
USAGE
}

die() {
    echo "$*" >&2
    exit 1
}

require_command() {
    if ! command -v "$1" >/dev/null 2>&1; then
        die "$1 is required but was not found on PATH."
    fi
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --target)
            [ "$#" -ge 2 ] || die "--target requires a value."
            TARGET="$2"
            shift 2
            ;;
        --target=*)
            TARGET="${1#--target=}"
            shift
            ;;
        --title)
            [ "$#" -ge 2 ] || die "--title requires a value."
            TITLE="$2"
            shift 2
            ;;
        --title=*)
            TITLE="${1#--title=}"
            shift
            ;;
        --notes)
            [ "$#" -ge 2 ] || die "--notes requires a value."
            NOTES="$2"
            shift 2
            ;;
        --notes=*)
            NOTES="${1#--notes=}"
            shift
            ;;
        --notes-file)
            [ "$#" -ge 2 ] || die "--notes-file requires a value."
            NOTES_FILE="$2"
            shift 2
            ;;
        --notes-file=*)
            NOTES_FILE="${1#--notes-file=}"
            shift
            ;;
        --draft)
            DRAFT=1
            shift
            ;;
        --prerelease)
            PRERELEASE=1
            shift
            ;;
        --skip-build)
            SKIP_BUILD=1
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        -*)
            die "Unknown option: $1"
            ;;
        *)
            if [ -n "$TAG" ]; then
                die "Unexpected argument: $1"
            fi
            TAG="$1"
            shift
            ;;
    esac
done

[ -n "$TAG" ] || {
    usage
    exit 1
}

if [ -n "$NOTES" ] && [ -n "$NOTES_FILE" ]; then
    die "Use either --notes or --notes-file, not both."
fi

if [ -n "$NOTES_FILE" ] && [ ! -f "$NOTES_FILE" ]; then
    die "Release notes file does not exist: $NOTES_FILE"
fi

if [ -n "$NOTES_FILE" ]; then
    notes_dir="$(cd "$(dirname "$NOTES_FILE")" && pwd)"
    NOTES_FILE="$notes_dir/$(basename "$NOTES_FILE")"
fi

if [ -z "$TITLE" ]; then
    TITLE="Styxpress $TAG"
fi

if [ -z "$NOTES" ] && [ -z "$NOTES_FILE" ]; then
    NOTES="Release $TAG."
fi

require_command git
require_command gh
require_command tar
require_command sha256sum

cd "$ROOT_DIR"

if [ "${STYXPRESS_RELEASE_ALLOW_DIRTY:-}" != "1" ]; then
    if [ -n "$(git status --porcelain)" ]; then
        die "Working tree is not clean. Commit or stash changes before releasing."
    fi
fi

if ! gh auth status >/dev/null 2>&1; then
    die "GitHub CLI is not authenticated. Run: gh auth login"
fi

mkdir -p "$RELEASE_DIR"

if [ "$SKIP_BUILD" -eq 0 ]; then
    find "$RELEASE_DIR" -maxdepth 1 -type d -name 'styxpress-admin_*' -prune -exec rm -rf {} +
    ./scripts/build-release.sh "$TARGET"
fi

find "$RELEASE_DIR" -maxdepth 1 -type f \
    \( -name 'styxpress-admin_*.tar.gz' -o -name 'styxpress-admin_*.zip' -o -name 'SHA256SUMS' \) \
    -delete

mapfile -t release_dirs < <(find "$RELEASE_DIR" -maxdepth 1 -type d -name 'styxpress-admin_*' | sort)
[ "${#release_dirs[@]}" -gt 0 ] || die "No release directories found in $RELEASE_DIR."

archives=()
for dir in "${release_dirs[@]}"; do
    dir_name="$(basename "$dir")"

    if [[ "$dir_name" == *windows* ]]; then
        require_command zip
        archive="$RELEASE_DIR/$dir_name.zip"
        echo "Packaging $archive"
        (cd "$RELEASE_DIR" && zip -qr "$(basename "$archive")" "$dir_name")
    else
        archive="$RELEASE_DIR/$dir_name.tar.gz"
        echo "Packaging $archive"
        (cd "$RELEASE_DIR" && tar -czf "$(basename "$archive")" "$dir_name")
    fi

    archives+=("$archive")
done

archive_names=()
for archive in "${archives[@]}"; do
    archive_names+=("$(basename "$archive")")
done

echo "Writing $RELEASE_DIR/SHA256SUMS"
(cd "$RELEASE_DIR" && sha256sum "${archive_names[@]}" > SHA256SUMS)
artifacts=("${archives[@]}" "$RELEASE_DIR/SHA256SUMS")

local_tag_exists=0
remote_tag_exists=0

if git rev-parse -q --verify "refs/tags/$TAG" >/dev/null; then
    local_tag_exists=1
fi

if git ls-remote --exit-code --tags origin "refs/tags/$TAG" >/dev/null 2>&1; then
    remote_tag_exists=1
fi

if [ "$local_tag_exists" -eq 0 ] && [ "$remote_tag_exists" -eq 1 ]; then
    echo "Fetching existing remote tag $TAG"
    git fetch origin "refs/tags/$TAG:refs/tags/$TAG"
    local_tag_exists=1
fi

if [ "$local_tag_exists" -eq 0 ]; then
    echo "Creating tag $TAG"
    git tag -a "$TAG" -m "Styxpress $TAG"
fi

tag_commit="$(git rev-list -n 1 "$TAG")"
head_commit="$(git rev-parse HEAD)"
if [ "$tag_commit" != "$head_commit" ]; then
    die "Tag $TAG points to $tag_commit, but HEAD is $head_commit."
fi

if [ "$remote_tag_exists" -eq 0 ]; then
    echo "Pushing tag $TAG"
    git push origin "$TAG"
else
    remote_tag_commit="$(git ls-remote --tags origin "refs/tags/$TAG^{}" | awk 'NR == 1 { print $1 }')"
    if [ -z "$remote_tag_commit" ]; then
        remote_tag_commit="$(git ls-remote --tags origin "refs/tags/$TAG" | awk 'NR == 1 { print $1 }')"
    fi

    if [ -n "$remote_tag_commit" ] && [ "$remote_tag_commit" != "$head_commit" ]; then
        die "Remote tag $TAG points to $remote_tag_commit, but HEAD is $head_commit."
    fi
fi

if gh release view "$TAG" >/dev/null 2>&1; then
    echo "GitHub release $TAG already exists; replacing uploaded assets."
    gh release upload "$TAG" "${artifacts[@]}" --clobber
else
    release_args=("$TAG" "${artifacts[@]}" --title "$TITLE")

    if [ -n "$NOTES_FILE" ]; then
        release_args+=(--notes-file "$NOTES_FILE")
    else
        release_args+=(--notes "$NOTES")
    fi

    if [ "$DRAFT" -eq 1 ]; then
        release_args+=(--draft)
    fi

    if [ "$PRERELEASE" -eq 1 ]; then
        release_args+=(--prerelease)
    fi

    echo "Creating GitHub release $TAG"
    gh release create "${release_args[@]}"
fi

echo "Release $TAG published."
