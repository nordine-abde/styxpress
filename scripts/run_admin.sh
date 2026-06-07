#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ADDR="${STYXPRESS_ADMIN_ADDR:-127.0.0.1:8080}"

source "$ROOT_DIR/scripts/go-toolchain.sh"
styxpress_require_supported_go_toolchain

cd "$ROOT_DIR/admin/web"
echo "Installing admin UI dependencies from lockfile..."
npm ci

echo "Auditing admin UI production dependencies..."
npm audit --omit=dev

echo "Building admin UI..."
npm run build

cd "$ROOT_DIR"
echo "Building styxpress-admin..."
go build -o styxpress-admin ./cmd/styxpress-admin

if [ "$#" -gt 0 ]; then
    echo "Starting styxpress-admin with provided arguments: $*"
    exec ./styxpress-admin "$@"
fi

echo "Starting styxpress-admin at http://$ADDR"
exec ./styxpress-admin -addr "$ADDR"
