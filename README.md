# Styxpress

<p align="center">
  <img src="admin/web/src/assets/styxpress-mark.png" alt="Styxpress mark" width="160" height="160">
</p>

Styxpress is a local static blog generator with a small Go admin server and a
Vue admin UI. Source content stays on the user's computer. Styxpress renders
HTML, feeds, sitemap, media, favicon files, and a single built-in stylesheet
into a local `publicDir`.

The public blog is static. Serve the generated `publicDir` with Caddy, Nginx,
or another static file server. The admin can optionally sync the generated
public folder over SFTP, but the public site never needs a dynamic application
server.

## Current Scope

Included:

- Multi-site local registry when running without `-config`.
- Local content folders.
- Local public output folders.
- Markdown posts stored as files.
- Draft and published post states.
- Visual and raw Markdown post editing.
- Post covers and post-local image assets.
- Local preview and render.
- One clean public stylesheet.
- Site title and description.
- Generated favicon with optional `.ico` replacement.
- Header links.
- Footer text and footer links.
- Footer watermark toggle for `Published with Styxpress`.
- RSS feed and sitemap output.
- Optional manual SFTP deploy for generated public output.

Not included in the first release:

- Server-backed content storage.
- Featured posts.
- Theme presets.
- Custom CSS editing.
- Saved themes.
- Remote verification.
- Multi-user admin, comments, search, or analytics.

## Content Layout

Source content lives under the configured `contentDir`:

```text
content/
  site.toml
  assets/
    brand.ico
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

Generated output lives under the configured `publicDir`:

```text
public/
  favicon.ico
  index.html
  feed.xml
  sitemap.xml
  assets/
    styxpress.css
    brand.ico
  posts/
    hello-world/
      index.html
      cover.jpg
      assets/
```

No Markdown parsing, database query, or application server is required for
normal public page views.

## Build And Run

The frontend requires Node `^20.19.0 || >=22.12.0`, as declared in
`admin/web/package.json`.

Go builds require a supported patched Go toolchain. For the beta release gate,
use Go `1.25.11` or newer on the Go 1.25 line, Go `1.26.4` or newer on the Go
1.26 line, or a newer supported Go release. The module encodes Go `1.25.11` as
the minimum and prefers the Go `1.26.4` toolchain.

From the repository root:

```bash
cd admin/web
npm install
npm run build
cd ../..

go build -o styxpress-admin ./cmd/styxpress-admin
./styxpress-admin
```

The command prints a local admin URL and the API session token:

```text
styxpress-admin listening on http://127.0.0.1:42317
styxpress-admin API session token: <token>
```

Open the printed URL in your browser and paste the session token into the admin
UI when prompted.

To use a fixed local port:

```bash
./styxpress-admin -addr 127.0.0.1:8080
```

The admin server binds to `127.0.0.1` by default. Running without `-config`
uses the multi-site registry under the user's config directory and suggests
new site workspaces under `~/Styxpress/<site-id>/`.

To run one explicit admin config file instead of the multi-site registry:

```bash
./styxpress-admin -config /path/to/config.toml
```

In `-config` mode, the admin UI exposes a single configured site and disables
creating or deleting sites.

## Workflow

1. Create or open a site from **My sites**. New sites get a normalized unique
   site id, default folders under `~/Styxpress/<site-id>/`, a default
   `site.toml`, and rendered local public pages.
2. In **Configuration**, set:
   - `contentDir`: local source content directory.
   - `publicDir`: local generated output directory.
   - optional SFTP deployment settings.
3. If SFTP deploy is enabled for the first time, Styxpress tests the SSH/SFTP
   connection before saving. When the remote folder already contains files, the
   UI asks for confirmation because the configured remote folder becomes fully
   managed by Styxpress. After confirmation, Styxpress immediately syncs the
   current local public output there.
4. If SFTP deploy needs a password or encrypted-key passphrase, enter it during
   setup or in the deploy panel. The secret is verified before use and kept only
   in the running admin server session.
5. In **Site**, edit:
   - title
   - description
   - favicon
   - header links
   - footer text
   - footer links
   - watermark visibility
   The preview updates while editing. **Save** writes `site.toml` and renders
   the public site.
6. In **Posts**, choose an existing post from the list or create a new one.
7. Edit content in visual compose mode or raw Markdown mode.
8. Use **Save** to write the post, mark it as published, and render the public
   output locally. Use **Deploy** from the deploy panel when you want to sync
   the generated output.

## Admin Config

The admin config is stored in the selected site registry entry, or in the file
passed with `-config`.

```toml
name = "My Blog"
content_dir = "content"
public_dir = "public"
deploy_enabled = false
sftp_host = ""
sftp_known_hosts_path = ""
sftp_key_path = ""
sftp_port = 22
sftp_remote_path = ""
sftp_user = ""
```

When SFTP deploy is enabled, `sftp_host`, `sftp_user`, and `sftp_remote_path`
are required. The configured remote folder is managed by Styxpress: manual
deploy overwrites matching remote files and removes files that are not present
in the local public folder. Secrets are not written to this config.

## Site Config

`site.toml` is stored in `contentDir`:

```toml
title = "My Blog"
description = "Latest posts"
favicon = "favicon.ico"

[header]

[[header.links]]
label = "Home"
href = "/"

[[header.links]]
label = "RSS"
href = "/feed.xml"

[footer]
text = "Local notes."
showWatermark = true

[[footer.links]]
label = "RSS"
href = "/feed.xml"
```

Allowed link hrefs are root-relative paths, anchors, `http`, `https`, and
`mailto`.

## SFTP Deploy

SFTP deploy syncs the generated `publicDir` to a configured remote folder.
Authentication can use `ssh-agent`, an SSH private key, a session password, or
a session passphrase for encrypted keys. Host keys are checked through the
configured `known_hosts` path or the user's default SSH known hosts files.
The first SFTP setup is tested before saving and warns when the remote folder is
not empty.

Local renders mark output as changed and let the user press **Deploy**. Deploy
is always an explicit manual action in the beta. The configured remote folder is
treated as owned by Styxpress during deploy.

## Local Output

After rendering, inspect:

```text
site/public/index.html
site/public/feed.xml
site/public/sitemap.xml
site/public/assets/styxpress.css
site/public/posts/hello-world/index.html
```

## Development Commands

```bash
go test ./...
go build -o styxpress-admin ./cmd/styxpress-admin

cd admin/web
npm install
npm run dev
npm run build
npm run preview
```

Useful scripts:

```bash
./scripts/run_admin.sh
./scripts/build-release.sh linux/amd64
./scripts/build-release.sh all
```

The frontend production build is expected at
`cmd/styxpress-admin/web/dist` for embedding by the Go binary. Do not commit
generated `dist` output unless explicitly requested.
