# Styxpress Initial Design

Styxpress is a lightweight blog engine for people who want to self-host a Markdown-based site with very low runtime cost.

This document records the initial direction and decisions made during brainstorming.

## Intended Use

Styxpress is designed first for self-hosted publishing.

The first version assumes the site owner is comfortable with:

- Running a service on their own machine or VPS.
- Managing files on disk.
- Configuring a reverse proxy.
- Restricting access to private routes through infrastructure.
- Editing Markdown directly when needed.

The initial scope is intentionally small: Styxpress focuses on a simple self-hosted workflow rather than a fully managed hosted CMS.

## Core Goals

- Very low memory usage.
- Very low CPU usage during public page serving.
- Markdown files as the source of truth.
- Self-hosted deployment.
- Minimal moving parts.
- No external database for the core blog engine.
- No generated metadata database or manifest for the first design.
- Public files should be static and read-only at serving time.
- The reverse proxy can serve public files directly in the first version.
- Admin/publishing operations should live in a separate client application.

## Storage Model

The database for the first version is the filesystem.

Private source content and public rendered output should be stored separately.

Private post content lives under `content/`. Public files served by Caddy, Nginx, or another reverse proxy live under `public/`.

The `content/` directory may exist on the server or only on the administrator's local machine, depending on the publishing profile. In server-backed content mode, the admin client uploads source Markdown, metadata, covers, and source assets to remote `content/` as well as rendered output to remote `public/`. In local-only content mode, the admin client keeps `content/` on the administrator's machine and uploads only generated `public/` files to the server.

In both modes, `content/` remains the source of truth for rendering, and `public/` remains the only directory that should be served to readers.

Posts are directories. The post slug is the directory name.

Example:

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

public/
  posts/
    hello-world/
      index.html
      cover.jpg
      assets/
```

The public URL maps directly to the post directory:

```text
/posts/hello-world
```

returns:

```text
public/posts/hello-world/index.html
```

The homepage is generated at:

```text
public/index.html
```

The engine should not require Postgres, MySQL, Redis, SQLite, or any external service for the core blog use case.

## Post Directory Contract

Each post is represented by a private source directory and a public output directory.

```text
content/posts/
  {slug}/
    source.md
    title.txt
    description.txt
    published_at.txt
    updated_at.txt
    cover.*
    assets/

public/posts/
  {slug}/
    index.html
    cover.*
    assets/
```

Private files:

- `source.md`: Markdown source edited or uploaded by the admin.
- `title.txt`: plain text post title.
- `description.txt`: plain text post description.
- `published_at.txt`: publication timestamp used for stable listing order.
- `updated_at.txt`: latest content or metadata update timestamp.
- `cover.*`: optional source cover image.
- `assets/`: optional source post-local assets.

Public files:

- `index.html`: final pre-rendered HTML returned to public readers.
- `cover.*`: copied public cover image for homepage cards, featured posts, and Open Graph image metadata.
- `assets/`: copied public post-local assets.

Rules:

- The slug is the post directory name.
- Slugs should be lowercase URL-safe names.
- Initial slug format: `a-z`, `0-9`, and `-`.
- Nested slugs are not supported in the first design.
- `title.txt` is required.
- `description.txt` is optional but recommended.
- `published_at.txt` is required for published posts.
- `updated_at.txt` is required for published posts.
- `cover.*` is optional.
- Other images and files should go under `assets/`.
- Text metadata files should be plain UTF-8.
- `title.txt` should be a single logical line after trimming whitespace.
- `description.txt` should be short plain text. It is not Markdown.
- `published_at.txt` should contain one RFC3339 timestamp, preferably UTC.
- `published_at.txt` should be set once on first publish and should not change during normal edits.
- `updated_at.txt` should contain one RFC3339 timestamp, preferably UTC.
- `updated_at.txt` should be written on first publish with the same value as `published_at.txt`.
- `updated_at.txt` should change when the admin updates the post source, metadata, cover image, or assets.
- If multiple `cover.*` files exist for one post, the admin/render step should reject or warn instead of guessing.
- Public cover and asset files should be copied from `content/posts/{slug}/` into `public/posts/{slug}/`.
- `content/posts/{slug}/assets/` is the canonical source for post assets.
- `public/posts/{slug}/assets/` contains only the public copy generated by the admin client.
- When editing a post, the admin client should overwrite replaced public assets and delete public assets that are no longer referenced or managed by the post.
- Symlinks from `public/` into `content/` should not be used in the first version.
- Uploaded assets should not be symlinks.
- Asset paths under `assets/` must stay inside the post asset directory after cleaning and validation.
- Asset paths containing traversal segments such as `..` should be rejected.
- The reverse proxy should serve only `public/`.
- Private `content/` files should not be reachable through public HTTP routes.

Valid slug examples:

```text
hello-world
go-http-server
post-2026
```

Invalid slug examples:

```text
../secret
Hello World
hello/world
hello_world
```

Allowed cover examples:

```text
cover.jpg
cover.jpeg
cover.png
cover.webp
cover.avif
```

## Drafts

Draft support is deferred.

The first version does not manage drafts as a separate server-side content type. The admin client publishes directly into `content/posts/{slug}/` and `public/posts/{slug}/`, and any unpublished writing can stay outside Styxpress as local Markdown files.

Future draft support may use a separate `drafts/` directory tree, but it is not part of the v1 storage contract.

## Rendering Model

Rendering is eager, not request-time.

Rendering is performed locally by the admin client, not by the public serving layer.

Remote render commands are not supported in the first version.

When a Markdown file is uploaded, created, or changed through the admin client:

1. The Markdown file is stored as `content/posts/{slug}/source.md`.
2. The admin-provided title is stored as `content/posts/{slug}/title.txt`.
3. The admin-provided description is stored as `content/posts/{slug}/description.txt` when present.
4. The publication timestamp is stored as `content/posts/{slug}/published_at.txt` when the post is first published.
5. The latest update timestamp is stored as `content/posts/{slug}/updated_at.txt`. On first publish, it has the same value as `published_at.txt`.
6. The optional cover image and post assets are stored under `content/posts/{slug}/`.
7. The engine parses and renders the Markdown once.
8. The engine writes the complete final page to `public/posts/{slug}/index.html`.
9. The engine copies the public cover image and assets into `public/posts/{slug}/`.

Public requests should serve already-rendered HTML from disk. This keeps CPU usage low and avoids parsing Markdown during normal page views.

The public serving layer should not need Markdown rendering dependencies at runtime.

If rendering fails, the previous `index.html` should not be overwritten with broken output.

Raw HTML policy:

- Raw HTML in Markdown should be escaped by default in the first version.
- The renderer should not pass arbitrary raw HTML from `source.md` into public `index.html`.
- HTML sanitization is deferred; v1 should prefer escaping over trying to sanitize.
- A future explicit configuration option may allow raw HTML for site owners who intentionally want that behavior.

## HTML Metadata

`index.html` should be a complete HTML document, not only an article fragment.

The render step should write page metadata into the HTML head.

`title.txt` is used for:

```html
<title>Hello World</title>
<meta property="og:title" content="Hello World">
```

`description.txt` is used for:

```html
<meta name="description" content="A practical setup using systemd, Caddy, and plain files.">
<meta property="og:description" content="A practical setup using systemd, Caddy, and plain files.">
```

`cover.*` is used for:

```html
<meta property="og:image" content="/posts/hello-world/cover.jpg">
```

Open Graph image URLs should preferably be absolute URLs generated from the configured site base URL:

```html
<meta property="og:image" content="https://blog.example.com/posts/hello-world/cover.jpg">
```

Metadata values must be escaped before being inserted into HTML attributes.

## Public Request Path

The public serving path should be cheap:

1. Receive request.
2. Map the URL to a file path.
3. Stream or send the file from disk.

Examples:

```text
GET /
-> public/index.html
```

```text
GET /posts/hello-world
-> public/posts/hello-world/index.html
```

```text
GET /posts/hello-world/assets/diagram.png
-> public/posts/hello-world/assets/diagram.png
```

```text
GET /posts/hello-world/cover.jpg
-> public/posts/hello-world/cover.jpg
```

No Markdown parsing should happen on public page requests.

No database query should be required for serving an individual post page.

For v1, the reverse proxy may serve the generated `public/` directory directly. In that setup, the reverse proxy configuration is the public routing layer.

The public routing layer should use a strict route allowlist.

Allowed public routes:

```text
GET /
GET /feed.xml
GET /sitemap.xml
GET /posts/{slug}
GET /posts/{slug}/cover.{jpg,jpeg,png,webp,avif}
GET /posts/{slug}/assets/{path}
```

Everything else should return 404 by default.

Reverse proxy security rules:

- The document root should be `public/`, not the whole site directory.
- The reverse proxy should not have access to `content/`, `config/`, `.tmp/`, local admin config, SSH keys, or upload scratch directories.
- The project should ship official safe Caddy and Nginx examples.
- The reverse proxy operating system user should have read-only access to `public/`.
- The admin SSH user should have write access to `content/` and `public/`.
- Directory listings should be disabled.
- Symlink following should be disabled or restricted so public routes cannot escape `public/`.
- Hidden files and dot-directories should not be served.
- Generic static file fallback should not expose arbitrary files under `public/`; routes should stay allowlisted.
- Request paths should be normalized before matching when the reverse proxy supports explicit normalization controls.
- Encoded traversal attempts such as `..`, `%2e%2e`, and mixed slash encodings should not resolve to files.
- Unknown extensions at the post root should not be served by default.

If a future Styxpress server binary is introduced, it should validate slugs before mapping URLs to filesystem paths and should follow the same route allowlist.

## Homepage

The homepage should also be pre-rendered by default.

Initial homepage content:

- Latest posts.
- Featured posts.

Latest posts:

- Derived from directories under `content/posts/`.
- Ordered by `published_at.txt` descending.
- If multiple posts have the same publication timestamp, sorted by slug ascending for stable output.
- Shows title and cover image.
- Generated by scanning post directories and reading metadata files when the homepage is regenerated.
- Filesystem timestamps should not define the public "latest posts" order.
- The homepage output is a static file generated by the admin client.
- The default latest-post count and any future customization options can be decided later.

Featured posts:

- Chosen by the admin.
- Stored in `content/featured.txt`.
- Contains one post slug per line.
- Rendered into the homepage when the homepage is regenerated.
- External URLs are not supported in the first version.

Example `content/featured.txt`:

```text
hello-world
another-post
```

No persistent generated metadata manifest is required for the homepage. The admin client can scan the private post directories during homepage generation, and the generated homepage HTML becomes the cached public output.

## Feed And Sitemap

Feed and sitemap support should be static in the first version.

The admin client should generate and upload:

```text
public/feed.xml
public/sitemap.xml
```

These files should be regenerated from `content/posts/` when the homepage is regenerated.

The public serving layer should serve them as normal static files. No server-side feed or sitemap generation should happen during public requests.

Feed decision:

- Generate RSS 2.0 at `public/feed.xml`.
- Use absolute URLs generated from the configured site base URL.
- Include the homepage/site title, post title, post URL, publication timestamp, update timestamp when available, and description when available.
- Include summaries/descriptions, not full rendered article HTML, in the first version.
- The default feed item count can be decided later.

Sitemap decision:

- Generate XML sitemap at `public/sitemap.xml`.
- Include the homepage URL.
- Include one URL for each published post.
- Use `updated_at.txt` as the sitemap `lastmod` value for posts.
- Use the latest post update time as the homepage `lastmod` value when available.
- Do not include source files, assets, cover images, or the feed URL in the sitemap for v1.

## Project Split

The first version can be shipped as one main executable plus static hosting configuration:

1. Styxpress Admin Client.
2. Official reverse proxy examples for serving `public/`.

A dedicated Styxpress Server binary is deferred. It can be added later if the project needs turnkey static serving, built-in health checks, custom routing, or dynamic features.

## Stack Decisions

Decision: the first Styxpress executable should be written in Go:

1. `styxpress-admin`

The admin GUI can use a frontend framework, built to static files and embedded into the Go admin binary.

The preferred frontend direction for the admin client is Vue with Vite.

This keeps final distribution simple:

```text
styxpress-admin
site/public/
reverse proxy config
```

No Python, Node, Bun, or Java runtime should be required on the user's machine after release builds are produced.

Node/Vite is only a build-time dependency for the admin frontend.

### Public Serving

Decision: v1 public serving can be handled by a reverse proxy or static file server, without a Styxpress backend process.

Reasoning:

- Public output is already static HTML and assets.
- Caddy, Nginx, and similar tools are mature static file servers.
- Serving only `public/` creates a clear security boundary.
- The site owner is already expected to configure a reverse proxy.
- Skipping a public backend reduces moving parts for the first version.

The reverse proxy should only serve files from `site/public/`. It should not be configured with access to `site/content/`, local admin config, temporary upload directories, or SSH material.

Deferred option:

- A future `styxpress-server` binary could provide the same static serving behavior in Go for users who prefer a single project-provided server process.

### Admin Stack

Decision: Styxpress Admin will be a Go local web app with an embedded static frontend.

Expected shape:

```text
styxpress-admin
-> starts a local server on 127.0.0.1
-> opens http://127.0.0.1:{port}
-> serves embedded Vue/Vite build assets
-> exposes local API endpoints for render, preview, and publish
```

The Go admin backend is trusted to perform privileged operations:

- Read explicitly selected local files.
- Render Markdown.
- Generate post directories.
- Open SSH/SFTP connections.
- Upload files to the remote server.
- Upload directly into the remote site directory for the first version.

The browser UI is only the control surface.

Expected admin frontend structure:

```text
admin/
  web/
    package.json
    vite.config.*
    src/
    dist/
```

Expected Go embedding model:

```go
//go:embed web/dist/*
var webFiles embed.FS
```

Likely Go dependencies for `styxpress-admin`:

- Markdown rendering: `github.com/yuin/goldmark`
- SSH: `golang.org/x/crypto/ssh`
- SFTP: `github.com/pkg/sftp`

The admin backend should still prefer Go standard library packages where possible:

- `net/http`
- `embed`
- `html/template`
- `os`
- `path/filepath`
- `mime/multipart`
- `encoding/json`
- `crypto/rand`

The admin frontend can call local API endpoints such as:

```text
GET  /api/config
POST /api/config
POST /api/test-ssh
POST /api/render-preview
POST /api/publish
POST /api/featured
```

Reasoning:

- Single-binary distribution for the admin client.
- No Python packaging problems.
- No runtime installation for end users.
- Go can embed the built frontend assets directly.
- Vue/Vite gives a comfortable modern GUI without making public serving depend on Node.
- Go can still be used later for a dedicated public server if needed.
- This avoids exposing any admin routes through the public site.

Rejected or deferred admin options:

- Python local web app: easy to develop, but packaging is less clean than a Go binary.
- Tkinter/PySide desktop GUI: workable, but less natural for HTML preview and less aligned with the browser-based admin direction.
- Electron: easy GUI development, but too heavy for this project.
- Tauri: attractive, but adds Rust/tooling complexity.

### Styxpress Admin Client

The admin client runs on the administrator's local computer.

The current preferred direction is a local web application:

- A Go backend runs on `127.0.0.1`.
- The browser provides the GUI.
- The Go backend handles local file access, Markdown rendering, SSH/SFTP, and uploads.

The browser should not open SSH connections directly and should not receive broad access to the local filesystem.

Responsibilities:

- Configure the blog machine host/IP.
- Configure SSH username.
- Configure SSH key path.
- Optionally accept an SSH key passphrase.
- Configure the remote site directory.
- Create post directory structures.
- Edit existing post directories.
- Render Markdown into complete `index.html` files.
- Write private source files under `content/` and public rendered files under `public/`.
- Upload files to the server over SSH/SFTP/SCP.
- Regenerate and upload the homepage.
- Regenerate and upload `feed.xml` and `sitemap.xml`.
- Manage featured posts.
- Optionally provide local preview before upload.

This avoids exposing admin routes on the internet.

SSH becomes the admin authorization layer.

The admin client must handle SSH credentials carefully. It should avoid storing key passphrases unless the user explicitly chooses that behavior.

Security rules for the local admin app:

- Bind only to `127.0.0.1`.
- Never bind to `0.0.0.0` by default.
- Generate a random session token on startup.
- Require the token for API calls.
- Disable or strictly limit CORS.
- Do not expose a generic "read any local path" endpoint.
- Do not store SSH key passphrases by default.
- Prefer explicit user-selected files and configured paths.
- Store local config with restrictive file permissions where possible.

File selection options:

- Use browser file inputs for Markdown files and cover images.
- Use configured paths or a small native file picker for SSH key paths if browser file handling is too limited.

Reasoning:

- The admin client runs on the administrator's machine, so its memory footprint is less critical than the public serving path.
- The local web app model keeps HTML preview natural and avoids native desktop GUI complexity.

## Publishing Over SSH

The admin client publishes files to the server using SSH-based file operations.

V1 publish flow for a new post:

1. Render the post locally.
2. Write or update the local source post directory under `content/posts/{slug}/`.
3. If the publishing profile uses server-backed content, create the final private remote post directory under `content/posts/{slug}/`.
4. Create the final public remote post directory under `public/posts/{slug}/`.
5. If server-backed content is enabled, upload source, metadata, cover image, and source assets directly into remote `content/posts/{slug}/`.
6. Upload rendered HTML, copied cover image, and copied assets directly into remote `public/posts/{slug}/`.
7. Regenerate the homepage, feed, and sitemap locally.
8. Upload the regenerated homepage, `feed.xml`, and `sitemap.xml` directly into their final remote paths under `public/`.

Example final paths:

```text
content/posts/hello-world/  optional remote source copy
public/posts/hello-world/   required remote public output
```

For the first version, the admin client should not require server-side shell operations, remote temporary directories, remote atomic moves, or backup rotation.

Overwrite behavior:

- If `content/posts/{slug}/` or `public/posts/{slug}/` already exists, the admin client should treat the operation as an edit of an existing post.
- Editing an existing post overwrites the files managed by Styxpress for that slug.
- The admin client should make overwrite behavior explicit in the UI before publishing changes to an existing slug.
- `published_at.txt` should be preserved when editing an existing post.
- `updated_at.txt` should be changed when editing an existing post.
- Public assets managed by Styxpress should be reconciled during edit: replaced files are overwritten and removed files are deleted.
- Unknown unmanaged files should not be deleted by default.
- A future explicit "clean publish" option may delete unmanaged files after warning the administrator.

This intentionally accepts a simpler failure model:

- Public readers may briefly see a partially uploaded post.
- Public readers may briefly see a homepage, feed, or sitemap that references a post whose upload has not completed.
- A failed upload may leave incomplete files in `content/posts/{slug}/` or `public/posts/{slug}/`.
- The administrator may need to manually delete or re-upload a broken post directory.
- When an upload fails, the admin UI should show the remote paths that may need cleanup.

This is acceptable for v1 because the first release targets small self-hosted sites and the priority is keeping implementation small.

Deferred safer publish flow:

- Upload to a remote temporary path.
- Verify required files remotely.
- Move the completed private and public directories into place atomically when possible.
- Publish the homepage, feed, and sitemap through temporary files and rename.
- Keep a backup or rollback path for replacing existing posts.

## Content Lifecycle

Initial lifecycle for a post:

1. User creates or imports a Markdown file in the admin client.
2. User provides a title.
3. User optionally provides a description.
4. User optionally provides a cover image.
5. The admin client sets `published_at.txt` on first publish.
6. The admin client sets `updated_at.txt` to the same value as `published_at.txt` on first publish.
7. The admin client writes or updates the post directory locally.
8. The admin client renders `source.md` into `index.html`.
9. The admin client regenerates the homepage, feed, and sitemap if needed.
10. The admin client uploads the result to the server over SSH/SFTP/SCP.
11. The reverse proxy serves pre-rendered HTML and static assets from `public/`.

Editing existing posts is supported in the admin GUI. The admin client is responsible for updating the relevant `content/posts/{slug}/` files, regenerating the corresponding `public/posts/{slug}/` files, and updating `updated_at.txt`.

## Database Decision

Decision: no database for the initial version.

Reasoning:

- Markdown files already provide durable storage.
- The site is expected to run on a single machine.
- The core workload is read-heavy.
- Rendering is done by the admin client when content changes, not by the server per request.
- External databases add deployment, backup, migration, and resource overhead.
- A generated metadata manifest is not needed for the first design.

Potential future option:

- SQLite could be introduced later as an optional cache or index if search, analytics, comments, or complex admin workflows need it.

SQLite should not replace Markdown files and post directories as the source of truth unless the product direction changes.

## Configuration

Configuration is split between public serving configuration on the blog machine and admin publishing profiles on the administrator's machine.

### Public Serving Configuration

The public serving layer is expected to be a reverse proxy or static file server for v1.

Expected configuration:

- Serve only `/var/lib/styxpress/site/public`.
- Map `/` to `public/index.html`.
- Map `/posts/{slug}` to `public/posts/{slug}/index.html`.
- Map post covers and assets to files under `public/posts/{slug}/`.
- Serve `/feed.xml` from `public/feed.xml`.
- Serve `/sitemap.xml` from `public/sitemap.xml`.
- Return 404 for everything else by default.
- Use the official Styxpress Caddy or Nginx example as the starting point.

### Admin Configuration

The admin client needs local publishing profiles on the administrator's machine.

The initial admin configuration format should be TOML.

Default local path:

```text
~/.config/styxpress/config.toml
```

Expected fields:

- Blog host/IP.
- SSH username.
- SSH key path.
- Remote site directory.
- Site base URL.
- Content storage mode: `server` or `local`.
- Local content directory path when using local-only content mode.

The admin config should be stored locally on the admin machine. It should not be uploaded into the public site directory by default.

SSH key passphrases should not be stored by default.

## Initial Folder Structure

Possible site data structure:

```text
site/
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
    featured.txt
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

This is only a starting point. The exact names can change.
