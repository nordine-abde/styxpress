#!/bin/sh
set -eu

mkdir -p /home/deploy/.ssh /srv/site/public /srv/site/content

if [ ! -s /keys/id_ed25519.pub ]; then
    echo "missing /keys/id_ed25519.pub; run scripts/setup-docker-test.sh first" >&2
    exit 1
fi

cp /keys/id_ed25519.pub /home/deploy/.ssh/authorized_keys
chmod 700 /home/deploy/.ssh
chmod 600 /home/deploy/.ssh/authorized_keys
chown -R deploy:deploy /home/deploy/.ssh /srv/site

exec /usr/sbin/sshd -D -e
