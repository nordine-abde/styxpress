# Docker Test Setup

This setup creates a local SSH/SFTP publishing target and a local Nginx public
server. It is intended for end-to-end testing before using a real VPS.

The setup writes runtime data under `site/`, which is ignored by Git:

```text
site/
  docker-local/     local admin content and public output
  docker-remote/    simulated remote /srv/site
  docker-ssh/       generated test SSH key
  docker-config.toml generated admin config
```

## Start

From the repository root:

```bash
scripts/setup-docker-test.sh
docker compose -f docker-compose.test.yml up --build
```

The first run may pull Docker images and build the SSH test image.

## Admin Configuration

Use the values printed by `scripts/setup-docker-test.sh`.

The script also writes a ready-to-use config file:

```text
site/docker-config.toml
```

After building the admin binary, run it with:

```bash
./styxpress-admin -config site/docker-config.toml
```

Typical values are:

```text
siteBaseUrl = http://127.0.0.1:8088
contentDir = /absolute/path/to/site/docker-local/content
publicDir = /absolute/path/to/site/docker-local/public
contentStorageMode = local
remoteHost = 127.0.0.1:2222
remoteUser = deploy
sshKeyPath = /absolute/path/to/site/docker-ssh/id_ed25519
remotePublicDir = /srv/site/public
remoteContentDir = /srv/site/content
```

For server-backed content mode, change `contentStorageMode` to `server`. The
same container also exposes `/srv/site/content`.

## Test Flow

1. Build and run the admin app.
2. Save the Docker test config in the admin UI.
3. Use the SSH test action; it should succeed without a passphrase.
4. Open the fixture post or create a new post.
5. Render the post.
6. Publish the post.
7. Open `http://127.0.0.1:8088/` and
   `http://127.0.0.1:8088/posts/hello-world`.

The Nginx container uses the same route allowlist as the production example:
homepage, feed, sitemap, post pages, covers, and post-local assets only.

## Reset

Stop containers:

```bash
docker compose -f docker-compose.test.yml down
```

Remove all generated Docker test data:

```bash
rm -rf site/docker-local site/docker-remote site/docker-ssh
```
