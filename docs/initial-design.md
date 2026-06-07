# Current Design

Styxpress is a local admin application for generating a static Markdown blog.
The admin server runs on the user's machine, edits local source files, renders
static output ahead of time, and can optionally sync that generated output over
SFTP.

The current release keeps the product intentionally small:

- Source content is local.
- Generated output is local.
- Rendering happens ahead of time.
- The public site is served as static files.
- Optional SFTP deploy syncs only the generated public output.
- Styxpress does not manage themes or custom CSS.

## File Model

Source content:

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

Generated output:

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

`content/` is the source of truth. `public/` is disposable generated output.

## Admin Config

The admin config tracks:

- site name
- local `contentDir`
- local `publicDir`
- optional SFTP deploy settings

When `styxpress-admin` runs without `-config`, site configs are stored in the
multi-site registry under the user's config directory and new site workspaces
default to `~/Styxpress/<site-id>/`. When it runs with `-config`, that explicit
file is the only admin config for the session.

SFTP deploy config contains connection details only. Passwords and encrypted
key passphrases are session-only values kept in the running admin server
process.

## Site Config

`content/site.toml` controls only basic presentation:

- site title
- site description
- favicon path
- header links
- footer text
- footer links
- footer watermark toggle

The public stylesheet is built into the renderer and is not editable from the
admin UI.

## Post Model

Each post is a folder under `content/posts/{slug}`.

- `source.md` is required.
- `title.txt` is required.
- `description.txt` is optional.
- `published_at.txt` exists only for published posts.
- `updated_at.txt` is maintained by the admin.
- `cover.*` is optional and supports `.jpg`, `.jpeg`, `.png`, `.webp`, and
  `.avif`.
- `assets/` contains post-local image files.

Post status is either `draft` or `published`. Saving a new post creates a
draft. Publishing writes `published_at.txt` and renders the post into public
output.

## Rendering

Rendering writes:

- post pages
- homepage
- feed
- sitemap
- stylesheet
- favicon
- copied covers
- copied post assets

Draft posts are omitted from public output. When rendering the whole site,
stale output for drafts is removed.

Markdown is rendered with Goldmark. Raw HTML in Markdown is escaped by the
renderer.

## Deploy

SFTP deploy is optional and syncs the configured `publicDir` to a remote
absolute path. Deploy status is based on local public files and the previous
deploy state file stored under the admin config area. Local renders mark output
as changed; the beta only supports explicit manual deploy.

Authentication can use `ssh-agent`, a configured SSH key, default SSH key
paths, a session password, or a session passphrase for encrypted keys. Host
keys are verified through a configured `known_hosts` path or the user's default
SSH known hosts files.

## Admin Scope

The admin server:

- binds locally by default
- manages saved site configs
- reads and writes configured local content folders
- writes generated files to configured local public folders
- serves the embedded Vue admin UI
- exposes authenticated local API endpoints
- optionally syncs generated public output over SFTP

The admin server should not expose broad filesystem access to the browser.
Uploads are accepted only through explicit favicon, cover, and asset endpoints.

## Deferred

These are not part of the current release:

- server-backed content
- remote verification
- featured posts
- custom CSS
- theme presets
- saved themes
- dynamic public serving
- multi-user admin
- comments
- search
- analytics
