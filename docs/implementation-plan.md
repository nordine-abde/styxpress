# Implementation Plan

This plan tracks the simplified first release.

## Release Goal

Ship a local static blog generator:

1. Read Markdown content from a local `contentDir`.
2. Store site settings in `content/site.toml`.
3. Render public output into a local `publicDir`.
4. Let the user create, edit, preview, publish, and render posts from the admin UI.

## MVP Milestones

### 1. Local Config

- Keep app config to `name`, `contentDir`, and `publicDir`.
- Keep content local only.
- Remove remote config, SSH config, and content storage modes.

### 2. Site Config

- Keep `title`, `description`, header links, footer text, footer links, and
  `showWatermark`.
- Remove theme presets, custom CSS, saved themes, header variants, and footer
  variants.

### 3. Content

- Keep post CRUD.
- Keep covers and post assets.
- Keep draft/published status.
- Remove featured posts and remote sync timestamps.

### 4. Rendering

- Render homepage from latest published posts only.
- Render post pages, feed, sitemap, and one fixed stylesheet.
- Remove stale draft output during full renders.

### 5. Admin UI

- Keep My sites, Configuration, Site, and Posts.
- Replace remote Publish controls with local Build controls.
- Remove SSH gate, remote verification, featured manager, and CSS editor.

### 6. Verification

- Run `go test ./...`.
- Run `npm run build` from `admin/web`.
- Build the admin binary with `go build -o styxpress-admin ./cmd/styxpress-admin`.

## Deferred Work

- Remote publishing.
- Deployment adapters.
- Multiple public themes.
- Custom CSS.
- Featured posts.
- Search, comments, analytics, and multi-user admin.
