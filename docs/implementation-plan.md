# Implementation Status

This document tracks the current simplified first release.

## Release Goal

Ship a local static blog generator:

1. Read Markdown content from a local `contentDir`.
2. Store site presentation settings in `content/site.toml`.
3. Render public output into a local `publicDir`.
4. Let the user create, edit, preview, publish, render, and optionally deploy
   posts from the admin UI.

## Implemented MVP Areas

### 1. Local And Multi-Site Config

- App config includes `name`, `contentDir`, `publicDir`, and optional deploy
  settings.
- Running without `-config` uses the multi-site registry and stores site config
  files under the user's config directory.
- New sites get normalized unique ids and default local folders under
  `~/Styxpress/<site-id>/`, plus a default `site.toml` and rendered local public
  pages.
- Running with `-config /path/to/config.toml` uses one explicit config file and
  disables create/delete site actions in the UI.

### 2. Site Config

- Site config includes `title`, `description`, `favicon`, header links, footer
  text, footer links, and `showWatermark`.
- The admin supports `.ico` favicon upload and reset to the built-in default.
- Theme presets, custom CSS, saved themes, header variants, and footer variants
  remain out of scope.

### 3. Content

- Post CRUD is file-backed under `content/posts/{slug}`.
- New post saves create drafts; publishing writes `published_at.txt`.
- Existing published posts preserve published state when saved.
- Covers and post-local image assets are supported.
- Featured posts and remote sync timestamps remain out of scope.

### 4. Rendering

- The renderer writes homepage, post pages, feed, sitemap, favicon, post media,
  and one fixed stylesheet.
- Homepage, feed, and sitemap use published posts only.
- Full renders remove stale public output for drafts.
- Raw HTML in Markdown is escaped.

### 5. SFTP Deploy

- Deploy settings live in admin config and support explicit manual sync.
- SFTP sync uploads generated public files and can optionally delete remote
  files that are no longer present locally.
- Deploy state is tracked locally under the admin config area.
- Passwords and encrypted-key passphrases are session-only and are not written
  to config.
- Remote verification beyond SSH host-key checking remains out of scope.

### 6. Admin UI

- The UI has **My sites**, **Configuration**, **Site**, and **Posts** areas.
- Navigation is store-driven through `stores/ui.js`; there is no committed
  `vue-router` dependency.
- Configuration includes local paths and SFTP settings.
- Site editing previews the homepage and saves `site.toml`.
- Post editing supports a visual editor, raw Markdown mode, cover upload, image
  assets, publish, render, and deploy status.

## Verification Commands

Use these commands before release-oriented changes:

```bash
go test ./...

cd admin/web
npm install
npm run build

cd ../..
go build -o styxpress-admin ./cmd/styxpress-admin
```

For release builds:

```bash
./scripts/build-release.sh linux/amd64
./scripts/build-release.sh all
```

## Deferred Work

- Server-backed content storage.
- Multiple public themes.
- Custom CSS.
- Featured posts.
- Remote verification.
- Search, comments, analytics, and multi-user admin.
