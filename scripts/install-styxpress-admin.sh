#!/usr/bin/env bash
set -euo pipefail

REPO="${STYXPRESS_REPO:-nordine-abde/styxpress}"
VERSION="${STYXPRESS_VERSION:-}"
INSTALL_DIR="${STYXPRESS_INSTALL_DIR:-$HOME/.local/bin}"
PROFILE_FILE="${STYXPRESS_PROFILE:-}"

usage() {
    cat <<'USAGE'
Usage:
  install-styxpress-admin.sh [options]

Examples:
  curl -fsSL https://raw.githubusercontent.com/nordine-abde/styxpress/main/scripts/install-styxpress-admin.sh | bash
  curl -fsSL https://raw.githubusercontent.com/nordine-abde/styxpress/main/scripts/install-styxpress-admin.sh | bash -s -- --version v0.0.1-beta.1

Options:
  --version <tag>       Install a specific GitHub release tag.
                        Defaults to the newest published release, including prereleases.
  --install-dir <dir>   Install directory. Defaults to ~/.local/bin.
  --repo <owner/repo>   GitHub repository. Defaults to nordine-abde/styxpress.
  -h, --help            Show this help.

Environment:
  STYXPRESS_VERSION      Same as --version.
  STYXPRESS_INSTALL_DIR  Same as --install-dir.
  STYXPRESS_REPO         Same as --repo.
  STYXPRESS_PROFILE      Shell profile file to update with PATH.
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
        --version)
            [ "$#" -ge 2 ] || die "--version requires a value."
            VERSION="$2"
            shift 2
            ;;
        --version=*)
            VERSION="${1#--version=}"
            shift
            ;;
        --install-dir)
            [ "$#" -ge 2 ] || die "--install-dir requires a value."
            INSTALL_DIR="$2"
            shift 2
            ;;
        --install-dir=*)
            INSTALL_DIR="${1#--install-dir=}"
            shift
            ;;
        --repo)
            [ "$#" -ge 2 ] || die "--repo requires a value."
            REPO="$2"
            shift 2
            ;;
        --repo=*)
            REPO="${1#--repo=}"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            die "Unknown argument: $1"
            ;;
    esac
done

require_command curl
require_command tar

if [ -z "$VERSION" ]; then
    echo "Resolving latest Styxpress release from $REPO..."
    VERSION="$(curl -fsSL "https://api.github.com/repos/$REPO/releases" \
        | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' \
        | head -n 1)"
fi

[ -n "$VERSION" ] || die "Could not resolve a release tag for $REPO."

case "$VERSION" in
    */*|*\\*|*[[:space:]]*)
        die "Version must not contain whitespace, slash, or backslash characters."
        ;;
esac

case "$(uname -s)" in
    Linux)
        goos="linux"
        ;;
    Darwin)
        goos="darwin"
        ;;
    *)
        die "Automatic install is supported on Linux and macOS only."
        ;;
esac

case "$(uname -m)" in
    x86_64|amd64)
        goarch="amd64"
        ;;
    arm64|aarch64)
        goarch="arm64"
        ;;
    *)
        die "Unsupported architecture: $(uname -m)"
        ;;
esac

asset_base="styxpress-admin_${VERSION}_${goos}_${goarch}"
asset="$asset_base.tar.gz"
release_url="https://github.com/$REPO/releases/download/$VERSION"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

echo "Downloading $asset..."
curl -fsSL "$release_url/$asset" -o "$tmp_dir/$asset"
curl -fsSL "$release_url/SHA256SUMS" -o "$tmp_dir/SHA256SUMS"

cd "$tmp_dir"

checksum_line="$(grep " $asset$" SHA256SUMS || true)"
[ -n "$checksum_line" ] || die "SHA256SUMS does not contain $asset."

if command -v sha256sum >/dev/null 2>&1; then
    printf '%s\n' "$checksum_line" | sha256sum -c -
elif command -v shasum >/dev/null 2>&1; then
    printf '%s\n' "$checksum_line" | shasum -a 256 -c -
else
    die "sha256sum or shasum is required to verify the downloaded archive."
fi

tar -xzf "$asset"

binary="$tmp_dir/$asset_base/styxpress-admin_$VERSION"
[ -x "$binary" ] || die "Downloaded archive did not contain $asset_base/styxpress-admin_$VERSION."

mkdir -p "$INSTALL_DIR"
versioned_command="$INSTALL_DIR/styxpress-admin_$VERSION"
stable_command="$INSTALL_DIR/styxpress-admin"

cp "$binary" "$versioned_command"
chmod +x "$versioned_command"
ln -sfn "$(basename "$versioned_command")" "$stable_command"

if [ -z "$PROFILE_FILE" ]; then
    case "$(basename "${SHELL:-}")" in
        zsh)
            PROFILE_FILE="$HOME/.zshrc"
            ;;
        bash)
            PROFILE_FILE="$HOME/.bashrc"
            ;;
        *)
            PROFILE_FILE="$HOME/.profile"
            ;;
    esac
fi

path_line="export PATH=\"$INSTALL_DIR:\$PATH\""
if [ "$INSTALL_DIR" = "$HOME/.local/bin" ]; then
    path_line='export PATH="$HOME/.local/bin:$PATH"'
fi

case ":$PATH:" in
    *":$INSTALL_DIR:"*)
        path_ready=1
        ;;
    *)
        path_ready=0
        ;;
esac

if [ "$path_ready" -eq 0 ]; then
    touch "$PROFILE_FILE"
    if ! grep -Fq "$path_line" "$PROFILE_FILE"; then
        printf '\n%s\n' "$path_line" >> "$PROFILE_FILE"
        echo "Added $INSTALL_DIR to PATH in $PROFILE_FILE."
    fi
fi

echo "Installed:"
echo "  $stable_command"
echo "  $versioned_command"

if [ "$path_ready" -eq 0 ]; then
    echo "Open a new terminal, or run:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
fi

echo "Run:"
echo "  styxpress-admin"
