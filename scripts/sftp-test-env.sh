#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/docker-compose.sftp-test.yml"
RUNTIME_DIR="$ROOT_DIR/.styxpress-test/sftp"
KNOWN_HOSTS="$RUNTIME_DIR/known_hosts"
PORT="${STYXPRESS_SFTP_TEST_PORT:-2222}"

usage() {
    cat <<EOF
Usage: $0 up|down|reset|known-hosts|logs

Starts a local SFTP test server for Styxpress deploy tests.

Connection:
  host: 127.0.0.1
  port: ${PORT}
  user: deploy
  password: ${STYXPRESS_SFTP_PASSWORD:-correct-pass}
  remote path: /home/deploy/public_html
  known_hosts: ${KNOWN_HOSTS}
EOF
}

wait_for_sftp() {
    mkdir -p "$RUNTIME_DIR"
    : >"$KNOWN_HOSTS"
    for _ in $(seq 1 60); do
        if ssh-keyscan -p "$PORT" 127.0.0.1 >"$KNOWN_HOSTS" 2>/dev/null && [ -s "$KNOWN_HOSTS" ]; then
            return 0
        fi
        sleep 1
    done
    echo "SFTP test container did not become reachable on 127.0.0.1:${PORT}." >&2
    docker compose -f "$COMPOSE_FILE" logs sftp >&2 || true
    return 1
}

print_connection() {
    cat <<EOF
SFTP test server ready.
  Host: 127.0.0.1
  Port: ${PORT}
  User: deploy
  Password: ${STYXPRESS_SFTP_PASSWORD:-correct-pass}
  Remote folder: /home/deploy/public_html
  Known hosts: ${KNOWN_HOSTS}
EOF
}

command="${1:-}"
case "$command" in
    up)
        mkdir -p "$RUNTIME_DIR/home"
        docker compose -f "$COMPOSE_FILE" up -d --build --force-recreate sftp
        wait_for_sftp
        print_connection
        ;;
    reset)
        mkdir -p "$RUNTIME_DIR/home"
        docker compose -f "$COMPOSE_FILE" up -d --build --force-recreate sftp
        wait_for_sftp
        print_connection
        ;;
    down)
        docker compose -f "$COMPOSE_FILE" down
        ;;
    known-hosts)
        wait_for_sftp
        echo "$KNOWN_HOSTS"
        ;;
    logs)
        docker compose -f "$COMPOSE_FILE" logs sftp
        ;;
    *)
        usage
        exit 2
        ;;
esac
