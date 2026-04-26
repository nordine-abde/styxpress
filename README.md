# Styxpress

Styxpress is a lightweight self-hosted blog engine for Markdown-based sites with very low public runtime cost.

The core idea is simple: keep source content as files, render pages ahead of time, and let a reverse proxy serve the generated `public/` directory directly. The admin and publishing workflow lives in a separate local application.

## Status

Styxpress is in early development.

The current repository contains:

- The initial design document in `docs/initial-design.md`.
- A Go admin executable skeleton in `cmd/styxpress-admin`.
- A Vue/Vite admin frontend in `admin/web`.
- Embedded frontend serving from the Go admin executable.

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

The planned local admin configuration format is TOML.

Default path:

```text
~/.config/styxpress/config.toml
```

Expected fields include:

- Blog host/IP.
- SSH username.
- SSH key path.
- Remote site directory.
- Site base URL.

SSH key passphrases should not be stored by default.

## License

See `LICENSE`.
