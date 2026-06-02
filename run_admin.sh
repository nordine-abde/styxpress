#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ADDR="${STYXPRESS_ADMIN_ADDR:-127.0.0.1:8080}"

cd "$ROOT_DIR/admin/web"
if [ ! -d node_modules ]; then
    echo "Installing admin UI dependencies..."
    npm install
fi

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
