#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET="${STYXPRESS_RELEASE_TARGET:-linux/amd64}"
VERSION="${STYXPRESS_RELEASE_VERSION:-}"
TARGET_SET=0

usage() {
    cat <<'USAGE'
Usage:
  ./scripts/build-release.sh [target] [options]

Examples:
  ./scripts/build-release.sh
  ./scripts/build-release.sh all
  ./scripts/build-release.sh all --version v0.1.0
  ./scripts/build-release.sh linux/amd64 --version v0.1.0

Options:
  --version <version>  Append a version to release directory and binary names.
  -h, --help          Show this help.
USAGE
}

die() {
    echo "$*" >&2
    exit 1
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --version)
            [ "$#" -ge 2 ] || die "--version requires a value."
            VERSION="$2"
            shift 2
            ;;
        --version=*)
            VERSION="${1#--version=}"
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
            if [ "$TARGET_SET" -eq 1 ]; then
                die "Unexpected argument: $1"
            fi
            TARGET="$1"
            TARGET_SET=1
            shift
            ;;
    esac
done

if [ -n "$VERSION" ] && [[ ! "$VERSION" =~ ^[A-Za-z0-9._+-]+$ ]]; then
    die "Version may only contain letters, numbers, dots, underscores, plus signs, and hyphens."
fi

source "$ROOT_DIR/scripts/go-toolchain.sh"
styxpress_require_supported_go_toolchain

targets=()
if [ "$TARGET" = "all" ]; then
    targets=(
        "linux/amd64"
        "linux/arm64"
        "darwin/amd64"
        "darwin/arm64"
        "windows/amd64"
    )
else
    targets=("$TARGET")
fi

cd "$ROOT_DIR/admin/web"
echo "Installing admin UI dependencies from lockfile..."
npm ci

echo "Auditing admin UI production dependencies..."
npm audit --omit=dev

echo "Building admin UI..."
npm run build

cd "$ROOT_DIR"

for target in "${targets[@]}"; do
    IFS=/ read -r goos goarch <<< "$target"
    if [ -z "${goos:-}" ] || [ -z "${goarch:-}" ]; then
        echo "Invalid target '$target'. Use GOOS/GOARCH, for example linux/amd64." >&2
        exit 1
    fi

    if [ -n "$VERSION" ]; then
        dir="dist/releases/styxpress-admin_${VERSION}_${goos}_${goarch}"
        binary="styxpress-admin_${VERSION}"
    else
        dir="dist/releases/styxpress-admin_${goos}_${goarch}"
        binary="styxpress-admin"
    fi

    if [ "$goos" = "windows" ]; then
        binary="$binary.exe"
    fi

    output="$dir/$binary"
    mkdir -p "$dir"

    echo "Building $target -> $output"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -trimpath -o "$output" ./cmd/styxpress-admin

    if go version -m "$output" | grep -q "httpmuxgo121=1"; then
        echo "Refusing release with httpmuxgo121=1; API method routes would return 404." >&2
        exit 1
    fi
done

echo "Release build complete."
