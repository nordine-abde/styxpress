#!/bin/sh
set -eu

user="deploy"
home="/home/${user}"
password="${STYXPRESS_SFTP_PASSWORD:-correct-pass}"
remote_path="${STYXPRESS_SFTP_REMOTE_PATH:-/home/deploy/public_html}"
reset_remote="${STYXPRESS_SFTP_RESET_REMOTE:-1}"
seed_remote="${STYXPRESS_SFTP_SEED_REMOTE:-1}"

mkdir -p "$home" /run/sshd

if ! id "$user" >/dev/null 2>&1; then
    addgroup -g 1000 "$user" >/dev/null 2>&1 || addgroup "$user" >/dev/null
    adduser -D -H -u 1000 -G "$user" -h "$home" -s /bin/sh "$user" >/dev/null
fi

printf '%s:%s\n' "$user" "$password" | chpasswd

if [ "$reset_remote" = "1" ]; then
    case "$remote_path" in
        "$home"/*)
            rm -rf "$remote_path"
            ;;
        *)
            echo "Refusing to reset remote path outside ${home}: ${remote_path}" >&2
            exit 1
            ;;
    esac
fi

mkdir -p "$remote_path"
if [ "$seed_remote" = "1" ]; then
    mkdir -p "$remote_path/stale"
    printf 'remote only\n' >"$remote_path/remote-only.txt"
    printf 'nested remote only\n' >"$remote_path/stale/nested.txt"
fi

chown -R "$user:$user" "$home"
ssh-keygen -A >/dev/null

cat >/etc/ssh/sshd_config <<'EOF'
Port 22
ListenAddress 0.0.0.0
PasswordAuthentication yes
KbdInteractiveAuthentication yes
PubkeyAuthentication no
PermitRootLogin no
X11Forwarding no
AllowTcpForwarding no
Subsystem sftp internal-sftp
EOF

exec /usr/sbin/sshd -D -e
