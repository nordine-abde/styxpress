# Styxpress V1 Implementation Plan

This plan describes the work needed to make Styxpress V1 usable as a local admin tool that renders Markdown posts and publishes static files to a self-hosted server.

Estimates are rough person-day ranges for one developer familiar with Go, Vue, and basic SSH/SFTP workflows. They do not include long product-design pauses or packaging for every operating system.

## Scope

V1 includes:

- Local Go admin server bound to `127.0.0.1`.
- Embedded Vue admin UI.
- Local `content/` source management.
- Optional server-backed `content/` upload.
- Static `public/` generation.
- Markdown rendering with escaped raw HTML by default.
- Homepage, RSS feed, and sitemap generation.
- SSH/SFTP publishing.
- Safe Caddy and Nginx examples.

V1 excludes comments, analytics, search, drafts as a managed content type, dynamic public serving, remote shell commands, rollback, and multi-user admin.

## Milestone 1: Project Foundation

Estimate: 1-2 days.

Tasks:

- Define Go package layout under `internal/` for config, content, rendering, publishing, and HTTP API.
- Add basic application config loading/saving from `~/.config/styxpress/config.toml`.
- Add restrictive config file permissions where supported.
- Add a startup session token and require it for API requests.
- Add basic structured JSON error responses.
- Add Go test scaffolding and a small set of unit tests for config and path validation.

Done when:

- `go test ./...` runs cleanly.
- The admin server exposes authenticated local API routes.

## Milestone 2: Content Model And Filesystem Operations

Estimate: 2-3 days.

Tasks:

- Implement slug validation: lowercase `a-z`, `0-9`, and `-`.
- Implement post directory read/write for `source.md`, `title.txt`, `description.txt`, `published_at.txt`, `updated_at.txt`, `cover.*`, and `assets/`.
- Implement `featured.txt` read/write.
- Support local-only and server-backed content modes in config.
- Validate asset paths and reject traversal segments.
- Detect duplicate cover files and report a clear error.
- Preserve `published_at.txt` on edits and update `updated_at.txt`.

Done when:

- Posts can be created, loaded, edited, and validated locally.
- Unit tests cover slug, metadata, and asset path edge cases.

## Milestone 3: Rendering Pipeline

Estimate: 3-5 days.

Tasks:

- Add Markdown rendering with `goldmark`.
- Escape raw HTML by default.
- Generate complete post HTML documents, not fragments.
- Escape metadata inserted into HTML attributes.
- Copy cover and managed assets into `public/posts/{slug}/`.
- Reconcile removed or replaced public assets during edits.
- Avoid overwriting previous `index.html` when rendering fails.
- Add local preview generation for the admin UI.

Done when:

- A Markdown post renders to `public/posts/{slug}/index.html`.
- Covers, assets, titles, descriptions, and Open Graph metadata render correctly.

## Milestone 4: Homepage, Feed, And Sitemap

Estimate: 2-4 days.

Tasks:

- Generate `public/index.html` from latest and featured posts.
- Sort latest posts by `published_at.txt` descending, then slug ascending.
- Generate RSS 2.0 at `public/feed.xml`.
- Generate `public/sitemap.xml`.
- Use absolute URLs from the configured site base URL.
- Exclude source files, cover images, assets, and feed URL from the sitemap.
- Add tests for ordering, featured slugs, feed URLs, and sitemap output.

Done when:

- Site-wide generated files update after publishing or editing posts.

## Milestone 5: SSH/SFTP Publishing

Estimate: 4-6 days.

Tasks:

- Add SSH key-based connection support.
- Support optional passphrase entry without storing it by default.
- Add `POST /api/test-ssh`.
- Implement remote directory creation.
- Upload rendered `public/` files to final remote paths.
- Upload `content/` files only when content storage mode is `server`.
- Treat existing remote post directories as edits.
- Preserve unmanaged remote files by default.
- Return clear cleanup paths on failed upload.

Done when:

- A post can be rendered locally and published to a test server through SFTP.

## Milestone 6: Admin API

Estimate: 2-3 days.

Tasks:

- Implement `GET /api/config` and `POST /api/config`.
- Implement post list, post detail, save post, render preview, publish post, and featured-post endpoints.
- Restrict local file access to configured paths and explicit uploads.
- Validate request payloads and return actionable errors.
- Document request and response shapes in code or docs.

Done when:

- The Vue app can perform the full create, edit, preview, and publish workflow through APIs.

## Milestone 7: Admin UI

Estimate: 5-8 days.

Tasks:

- Create Pinia stores for config, posts, publishing, preview, and UI state.
- Build reusable UI primitives for buttons, inputs, panels, badges, empty states, loading states, and confirmation prompts.
- Build configuration screens for site URL, SSH settings, remote directory, content storage mode, and local content path.
- Build post list and post editor screens.
- Add Markdown import/edit, metadata fields, cover selection, and asset selection.
- Add preview, publish, SSH test, and featured-post management flows.
- Handle loading, empty, unauthorized, validation, and error states.
- Avoid committing generated `dist` output unless explicitly requested.

Done when:

- A user can configure a site, create/edit a post, preview it, publish it, and manage featured posts from the UI.

## Milestone 8: Reverse Proxy Examples

Estimate: 1-2 days.

Tasks:

- Add a safe Caddy example that serves only `public/`.
- Add a safe Nginx example that serves only `public/`.
- Disable directory listings.
- Avoid serving hidden files.
- Keep route handling aligned with the public route allowlist.
- Document required filesystem permissions.

Done when:

- A user can copy and adapt the examples for a basic VPS deployment.

## Milestone 9: Verification And Release Readiness

Estimate: 3-5 days.

Tasks:

- Add an end-to-end local fixture site.
- Test local-only content mode.
- Test server-backed content mode.
- Test edit behavior for existing posts.
- Test failed publish behavior and cleanup messages.
- Run `go test ./...` and `npm run build`.
- Manually verify Caddy or Nginx serving against generated `public/`.
- Update `README.md` with the final V1 workflow.
- Add troubleshooting notes for SSH keys, permissions, and missing frontend builds.

Done when:

- A clean checkout can build the admin UI, build the Go binary, create a post, generate a site, and publish it to a test server.

## Estimated Total

Expected implementation effort: 23-38 person-days.

Suggested buffer: 20-30%, especially around SSH/SFTP behavior, cross-platform file permissions, and admin UI polish.

Practical V1 target: 5-8 focused weeks for one developer, or 3-5 weeks for two developers if backend/rendering and frontend UI work are split cleanly.

## Recommended Build Order

1. Foundation and config.
2. Content model.
3. Rendering pipeline.
4. Homepage, feed, and sitemap.
5. Admin API.
6. Minimal admin UI.
7. SSH/SFTP publishing.
8. Full UI workflow and polish.
9. Reverse proxy examples and release verification.

This order keeps core file generation testable before remote publishing and UI polish depend on it.
