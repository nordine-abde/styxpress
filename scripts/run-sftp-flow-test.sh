#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUNTIME_DIR="$ROOT_DIR/.styxpress-test/admin"
SFTP_RUNTIME_DIR="$ROOT_DIR/.styxpress-test/sftp"
SERVER_LOG="$RUNTIME_DIR/server.log"
SFTP_PORT="${STYXPRESS_SFTP_TEST_PORT:-2222}"
SERVER_PID=""

cleanup() {
    if [ -n "$SERVER_PID" ]; then
        kill "$SERVER_PID" >/dev/null 2>&1 || true
        wait "$SERVER_PID" >/dev/null 2>&1 || true
    fi
}
trap cleanup EXIT

require_command() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo "$1 is required for this test." >&2
        exit 1
    fi
}

require_command curl
require_command jq
require_command ssh-keyscan
require_command docker
require_command go

"$ROOT_DIR/scripts/sftp-test-env.sh" reset

rm -rf "$RUNTIME_DIR"
mkdir -p "$RUNTIME_DIR"

XDG_CONFIG_HOME="$RUNTIME_DIR/xdg" GOTOOLCHAIN=auto go run ./cmd/styxpress-admin -addr 127.0.0.1:0 >"$SERVER_LOG" 2>&1 &
SERVER_PID=$!

BASE=""
TOKEN=""
for _ in $(seq 1 60); do
    if ! kill -0 "$SERVER_PID" >/dev/null 2>&1; then
        cat "$SERVER_LOG" >&2 || true
        exit 1
    fi
    BASE="$(sed -n 's/.*listening on \(http:\/\/[^[:space:]]*\).*/\1/p' "$SERVER_LOG" | tail -1)"
    TOKEN="$(sed -n 's/.*API session token: //p' "$SERVER_LOG" | tail -1)"
    if [ -n "$BASE" ] && [ -n "$TOKEN" ] && curl -fsS -H "X-Styxpress-Session: $TOKEN" "$BASE/api/health" >/dev/null 2>&1; then
        break
    fi
    sleep 1
done

if [ -z "$BASE" ] || [ -z "$TOKEN" ]; then
    cat "$SERVER_LOG" >&2 || true
    echo "admin server did not become reachable" >&2
    exit 1
fi

api() {
    local method="$1"
    local path="$2"
    local body="${3:-}"
    if [ -n "$body" ]; then
        curl -fsS -H "X-Styxpress-Session: $TOKEN" -H 'Content-Type: application/json' -X "$method" -d "$body" "$BASE$path"
    else
        curl -fsS -H "X-Styxpress-Session: $TOKEN" -X "$method" "$BASE$path"
    fi
}

content_dir="$RUNTIME_DIR/site-content"
public_dir="$RUNTIME_DIR/site-public"
known_hosts="$SFTP_RUNTIME_DIR/known_hosts"

create_body="$(jq -n --arg name "Docker SFTP Flow" --arg content "$content_dir" --arg public "$public_dir" '{name:$name,config:{contentDir:$content,publicDir:$public}}')"
api POST /api/sites "$create_body" >"$RUNTIME_DIR/create.json"
jq -e '.name == "Docker SFTP Flow" and .config.contentDir != "" and .config.publicDir != ""' "$RUNTIME_DIR/create.json" >/dev/null
test -f "$content_dir/site.toml"
test -f "$public_dir/index.html"

setup_body() {
    local secret="$1"
    local confirm="$2"
    jq -n \
        --arg content "$content_dir" \
        --arg public "$public_dir" \
        --arg known "$known_hosts" \
        --arg secret "$secret" \
        --argjson port "$SFTP_PORT" \
        --argjson confirm "$confirm" \
        '{
            config: {
                contentDir: $content,
                publicDir: $public,
                deploy: {
                    enabled: true,
                    sftp: {
                        host: "127.0.0.1",
                        port: $port,
                        user: "deploy",
                        remotePath: "/home/deploy/public_html",
                        knownHostsPath: $known
                    }
                }
            },
            secret: $secret,
            confirmRemoteOverwrite: $confirm
        }'
}

wrong_setup_code="$(curl -sS -o "$RUNTIME_DIR/wrong-setup.json" -w '%{http_code}' -H "X-Styxpress-Session: $TOKEN" -H 'Content-Type: application/json' -X POST -d "$(setup_body wrong-pass false)" "$BASE/api/deploy/setup")"
if [ "$wrong_setup_code" = "200" ]; then
    cat "$RUNTIME_DIR/wrong-setup.json" >&2
    echo "wrong setup password unexpectedly succeeded" >&2
    exit 1
fi
api GET /api/config >"$RUNTIME_DIR/config-after-wrong.json"
jq -e '.deploy.enabled == false' "$RUNTIME_DIR/config-after-wrong.json" >/dev/null

api POST /api/deploy/setup "$(setup_body correct-pass false)" >"$RUNTIME_DIR/setup-warning.json"
jq -e '.requiresConfirmation == true and .remoteFiles >= 2' "$RUNTIME_DIR/setup-warning.json" >/dev/null
api GET /api/config >"$RUNTIME_DIR/config-after-warning.json"
jq -e '.deploy.enabled == false' "$RUNTIME_DIR/config-after-warning.json" >/dev/null

api POST /api/deploy/setup "$(setup_body correct-pass true)" >"$RUNTIME_DIR/setup-confirm.json"
jq -e '(.requiresConfirmation != true) and .config.deploy.enabled == true and .secretSet == true and .summary.outOfSync == false and .summary.deleted >= 2 and ((.summary.uploaded + .summary.updated) > 0)' "$RUNTIME_DIR/setup-confirm.json" >/dev/null
docker exec styxpress-sftp-test sh -c 'test -f /home/deploy/public_html/index.html && test ! -f /home/deploy/public_html/remote-only.txt && test ! -f /home/deploy/public_html/stale/nested.txt'

api DELETE /api/deploy/secret >"$RUNTIME_DIR/clear-secret.json"
jq -e '.secretSet == false' "$RUNTIME_DIR/clear-secret.json" >/dev/null
wrong_secret_code="$(curl -sS -o "$RUNTIME_DIR/wrong-secret.json" -w '%{http_code}' -H "X-Styxpress-Session: $TOKEN" -H 'Content-Type: application/json' -X POST -d '{"secret":"wrong-pass"}' "$BASE/api/deploy/secret")"
if [ "$wrong_secret_code" = "200" ]; then
    cat "$RUNTIME_DIR/wrong-secret.json" >&2
    echo "wrong session password unexpectedly saved" >&2
    exit 1
fi
api GET /api/deploy/status >"$RUNTIME_DIR/status-after-wrong-secret.json"
jq -e '.secretSet == false' "$RUNTIME_DIR/status-after-wrong-secret.json" >/dev/null
api POST /api/deploy/secret '{"secret":"correct-pass"}' >"$RUNTIME_DIR/right-secret.json"
jq -e '.secretSet == true' "$RUNTIME_DIR/right-secret.json" >/dev/null

printf 'manual deploy file\n' >"$public_dir/manual.txt"
api GET /api/deploy/status >"$RUNTIME_DIR/status-outofsync.json"
jq -e '.outOfSync == true and .summary.uploaded >= 1' "$RUNTIME_DIR/status-outofsync.json" >/dev/null
api POST /api/deploy '{}' >"$RUNTIME_DIR/deploy-now.json"
jq -e '.outOfSync == false and .uploaded >= 1' "$RUNTIME_DIR/deploy-now.json" >/dev/null
docker exec styxpress-sftp-test sh -c 'test -f /home/deploy/public_html/manual.txt && grep -q "manual deploy file" /home/deploy/public_html/manual.txt'
api GET /api/deploy/status >"$RUNTIME_DIR/status-final.json"
jq -e '.outOfSync == false' "$RUNTIME_DIR/status-final.json" >/dev/null

cat <<EOF
SFTP flow passed.
  Admin: $BASE
  SFTP: 127.0.0.1:$SFTP_PORT
  Runtime: $RUNTIME_DIR
  wrong setup HTTP: $wrong_setup_code
  wrong secret HTTP: $wrong_secret_code
EOF
