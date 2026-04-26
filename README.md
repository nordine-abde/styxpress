# Styxpress

Styxpress is a lightweight self-hosted blog engine for Markdown-based sites with very low public runtime cost.

The core idea is simple: keep source content as files, render pages ahead of time, and let a reverse proxy serve the generated `public/` directory directly. The admin and publishing workflow lives in a separate local application.

## Status

Styxpress is in early development.

The current repository contains:

- The initial design document in `docs/initial-design.md`.
- A Go local admin executable in `cmd/styxpress-admin`.
- A Vue/Vite admin frontend in `admin/web`.
- Embedded frontend serving from the Go admin executable.
- Filesystem content, rendering, admin API, admin UI, and SSH/SFTP publishing workflows.

The public blog server is intentionally not implemented for v1. Public files are expected to be served by Caddy, Nginx, or another static file server.

## Design Summary

Styxpress uses the filesystem as the source of truth.

Private source content lives under `content/`:

```text
content/
  posts/
    hello-world/
      source.md
      title.txt
      description.txt
      published_at.txt
      updated_at.txt
      cover.jpg
      assets/
```

Rendered public output lives under `public/`:

```text
public/
  index.html
  feed.xml
  sitemap.xml
  posts/
    hello-world/
      index.html
      cover.jpg
      assets/
```

The intended public route:

```text
/posts/hello-world
```

maps to:

```text
public/posts/hello-world/index.html
```

No Markdown parsing, database query, or application server should be required for normal public page views.

## V1 Direction

- Markdown source files are stored in `content/posts/{slug}/`.
- Rendered HTML and public assets are generated into `public/`.
- The admin client renders locally.
- The admin client publishes files over SSH/SFTP/SCP.
- The reverse proxy serves only `public/`.
- No database is required for the core blog workflow.
- Draft support is deferred.
- Raw HTML in Markdown is escaped by default.

For the full design, read `docs/initial-design.md`.

## Repository Layout

```text
admin/web/              Vue/Vite admin frontend
cmd/styxpress-admin/    Go local admin executable
docs/                   Project design notes
internal/               Future internal Go packages
pkg/                    Future public Go packages, if needed
```

The frontend build is written to:

```text
cmd/styxpress-admin/web/dist
```

The Go executable embeds files from:

```text
cmd/styxpress-admin/web
```

## Requirements

- Go 1.22.2 or newer.
- Node.js compatible with the version declared in `admin/web/package.json`.
- npm.

Node/Vite are build-time requirements for the admin frontend. They are not intended to be runtime requirements for the final admin binary.

## Build And Run

From the repository root:

```bash
cd admin/web
npm install
npm run build
cd ../..

go build -o styxpress-admin ./cmd/styxpress-admin
./styxpress-admin
```

The command prints a local admin URL:

```text
styxpress-admin listening on http://127.0.0.1:42317
```

Open that URL in your browser.

To use a fixed local port:

```bash
./styxpress-admin -addr 127.0.0.1:8080
```

The admin server binds to `127.0.0.1` by default. It should not bind to `0.0.0.0` by default because the admin app will eventually handle local files and SSH credentials.

## V1 Workflow

1. Build the admin frontend:

```bash
cd admin/web
npm install
npm run build
cd ../..
```

2. Build and start the local admin server:

```bash
go build -o styxpress-admin ./cmd/styxpress-admin
./styxpress-admin
```

3. Open the printed local URL and enter the printed API session token.

4. Configure the site:

- `siteBaseUrl`: the public canonical URL, for example `https://blog.example.com`.
- `contentDir`: local source content directory.
- `publicDir`: local generated output directory.
- `contentStorageMode`: `local` to publish only generated public files, or `server` to also upload source content.
- `remoteHost`, `remoteUser`, `sshKeyPath`, and `remotePublicDir` for SSH/SFTP publishing.
- `remoteContentDir` when `contentStorageMode` is `server`.

5. Create or edit posts in the admin UI. Saving writes source files under `content/posts/{slug}/`.

6. Render a post to generate `public/posts/{slug}/index.html`, the homepage, `feed.xml`, and `sitemap.xml`.

7. Publish the post. Styxpress renders locally first, then uploads generated public files over SSH/SFTP. In server-backed content mode it also uploads `content/`.

8. Serve the remote `public/` directory with Caddy, Nginx, or another static file server using the route allowlist in `docs/reverse-proxy.md`.

## Local Fixture

A small fixture site is available in `fixtures/local-site/content`. It is useful for release checks and local experiments without creating content from scratch.

Example local setup:

```bash
mkdir -p site
cp -R fixtures/local-site/content site/content
```

Then configure the admin UI with:

```text
contentDir = site/content
publicDir = site/public
siteBaseUrl = https://example.com
contentStorageMode = local
```

After rendering, inspect `site/public/index.html`, `site/public/feed.xml`, `site/public/sitemap.xml`, and `site/public/posts/hello-world/index.html`.

## Docker Test Setup

A local Docker setup is available for end-to-end testing without a real VPS. It
starts:

- an SSH/SFTP publishing target on `127.0.0.1:2222`
- an Nginx public server on `127.0.0.1:8088`

Prepare local test data and keys:

```bash
scripts/setup-docker-test.sh
```

Start the test services:

```bash
docker compose -f docker-compose.test.yml up --build
```

Then configure the admin UI with the values printed by the setup script, render
a post, publish it, and open `http://127.0.0.1:8088/`.

Full instructions are in `docs/docker-test.md`.

## Frontend Development

Run the Vite dev server from `admin/web`:

```bash
npm run dev
```

Build the frontend for embedding:

```bash
npm run build
```

The frontend stack is Vue, Vite, and Pinia.

## Public Serving Model

V1 public serving should be handled by a reverse proxy or static file server.

The public server should:

- Serve only the generated `public/` directory.
- Map `/` to `public/index.html`.
- Map `/posts/{slug}` to `public/posts/{slug}/index.html`.
- Serve `/feed.xml` and `/sitemap.xml`.
- Serve post covers and post-local assets from `public/posts/{slug}/`.
- Return 404 for everything else by default.
- Disable directory listings.
- Avoid serving hidden files.
- Avoid following symlinks that escape `public/`.
- Keep the reverse proxy user read-only against `public/`.

Copyable Caddy and Nginx examples are available in `docs/reverse-proxy.md`.

## Admin Configuration

The local admin configuration format is TOML.

Default path:

```text
~/.config/styxpress/config.toml
```

Fields include:

- `site_base_url`
- `content_dir`
- `public_dir`
- `content_storage_mode`
- `remote_host`
- `remote_user`
- `ssh_key_path`
- `remote_public_dir`
- `remote_content_dir`

SSH key passphrases should not be stored by default.

## Troubleshooting

### Missing Admin Frontend

If the admin server returns:

```text
admin frontend is not built; run npm run build in admin/web
```

build the frontend before building or running the Go binary:

```bash
cd admin/web
npm install
npm run build
cd ../..
go build -o styxpress-admin ./cmd/styxpress-admin
```

### SSH Keys

- Use an SSH key that can log in as `remote_user` on `remote_host`.
- Store the private key outside the repository and point `ssh_key_path` at it.
- If the key is encrypted, enter the passphrase only in the admin UI when testing or publishing. Styxpress does not store passphrases in config.
- Verify access outside Styxpress first with `ssh -i /path/to/key user@host`.

### Remote Permissions

- The publishing user needs write access to `remote_public_dir`.
- In server-backed content mode, the publishing user also needs write access to `remote_content_dir`.
- The reverse proxy user should have read-only access to `remote_public_dir`.
- Do not serve `content/` publicly.
- If publishing fails partway through, the API reports cleanup paths for files that were uploaded or attempted during the failed publish.

### Public Serving

- Point the public web server root at generated `public/`, not at the repository root.
- Disable directory listings.
- Block hidden files and dot-directories.
- Avoid symlinks inside `public/`.
- Use `docs/reverse-proxy.md` and the examples in `docs/examples/` as the baseline.

## License

See `LICENSE`.
