#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET="${1:-linux/amd64}"

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

    dir="dist/releases/styxpress-admin_${goos}_${goarch}"
    binary="styxpress-admin"
    if [ "$goos" = "windows" ]; then
        binary="styxpress-admin.exe"
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
