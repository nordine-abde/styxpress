#!/usr/bin/env bash

STYXPRESS_MIN_GO_1_25_VERSION="${STYXPRESS_MIN_GO_1_25_VERSION:-1.25.11}"
STYXPRESS_MIN_GO_1_26_VERSION="${STYXPRESS_MIN_GO_1_26_VERSION:-1.26.4}"

styxpress_parse_go_version() {
    local raw="${1#go}"

    if [[ "$raw" =~ ^([0-9]+)\.([0-9]+)(\.([0-9]+))?([[:space:]-]|$) ]]; then
        printf '%s.%s.%s\n' "${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}" "${BASH_REMATCH[4]:-0}"
        return 0
    fi

    echo "Unsupported Go version format: ${1:-unknown}" >&2
    return 1
}

styxpress_version_ge() {
    local current_major current_minor current_patch
    local minimum_major minimum_minor minimum_patch

    IFS=. read -r current_major current_minor current_patch <<< "$1"
    IFS=. read -r minimum_major minimum_minor minimum_patch <<< "$2"

    if (( current_major != minimum_major )); then
        (( current_major > minimum_major ))
        return
    fi
    if (( current_minor != minimum_minor )); then
        (( current_minor > minimum_minor ))
        return
    fi
    (( current_patch >= minimum_patch ))
}

styxpress_current_go_version() {
    local raw_version

    if ! command -v go >/dev/null 2>&1; then
        echo "Go is required but was not found on PATH." >&2
        return 1
    fi

    raw_version="$(go env GOVERSION 2>/dev/null || true)"
    if [ -z "$raw_version" ]; then
        raw_version="$(go version 2>/dev/null | awk '{print $3}' || true)"
    fi

    styxpress_parse_go_version "$raw_version"
}

styxpress_require_supported_go_toolchain() {
    local current_version current_major current_minor current_patch required_version

    current_version="$(styxpress_current_go_version)" || return 1
    IFS=. read -r current_major current_minor current_patch <<< "$current_version"

    if (( current_major > 1 )); then
        echo "Using Go $current_version."
        return 0
    fi

    if (( current_major < 1 || current_minor < 25 )); then
        echo "Go $current_version is unsupported for Styxpress release builds." >&2
        echo "Use Go ${STYXPRESS_MIN_GO_1_25_VERSION}+ on the 1.25 line, Go ${STYXPRESS_MIN_GO_1_26_VERSION}+ on the 1.26 line, or a newer supported Go release." >&2
        return 1
    fi

    if (( current_minor == 25 )); then
        required_version="$STYXPRESS_MIN_GO_1_25_VERSION"
    elif (( current_minor == 26 )); then
        required_version="$STYXPRESS_MIN_GO_1_26_VERSION"
    else
        echo "Using Go $current_version."
        return 0
    fi

    if ! styxpress_version_ge "$current_version" "$required_version"; then
        echo "Go $current_version is below the required patched Go $required_version toolchain for this release line." >&2
        echo "Upgrade Go before building Styxpress release binaries." >&2
        return 1
    fi

    echo "Using Go $current_version."
}
