#!/bin/sh
set -eu

root_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
site_dir="$root_dir/site"
key_dir="$site_dir/docker-ssh"
remote_dir="$site_dir/docker-remote"
local_dir="$site_dir/docker-local"
config_path="$site_dir/docker-config.toml"

mkdir -p "$key_dir" "$remote_dir/public" "$remote_dir/content" "$local_dir"

if [ ! -f "$key_dir/id_ed25519" ]; then
    ssh-keygen -t ed25519 -N "" -f "$key_dir/id_ed25519" -C "styxpress-docker-test"
fi

rm -rf "$local_dir/content"
cp -R "$root_dir/fixtures/local-site/content" "$local_dir/content"
mkdir -p "$local_dir/public"

chmod 700 "$key_dir"
chmod 600 "$key_dir/id_ed25519"
chmod 644 "$key_dir/id_ed25519.pub"

cat > "$config_path" <<EOF
content_dir = "$local_dir/content"
content_storage_mode = "local"
public_dir = "$local_dir/public"
remote_content_dir = "/srv/site/content"
remote_host = "127.0.0.1:2222"
remote_public_dir = "/srv/site/public"
remote_user = "deploy"
site_base_url = "http://127.0.0.1:8088"
ssh_key_path = "$key_dir/id_ed25519"
EOF

cat <<EOF
Docker test files are ready.

Generated admin config:
  $config_path

Admin config values:
  siteBaseUrl: http://127.0.0.1:8088
  contentDir: $local_dir/content
  publicDir: $local_dir/public
  contentStorageMode: local
  remoteHost: 127.0.0.1:2222
  remoteUser: deploy
  sshKeyPath: $key_dir/id_ed25519
  remotePublicDir: /srv/site/public
  remoteContentDir: /srv/site/content

Start services:
  docker compose -f docker-compose.test.yml up --build

Run the admin app with:
  ./styxpress-admin -config $config_path

After publishing, open:
  http://127.0.0.1:8088/
  http://127.0.0.1:8088/posts/hello-world
EOF
