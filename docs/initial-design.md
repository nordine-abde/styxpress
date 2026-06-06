# Initial Design

Styxpress is a local admin application for generating a static Markdown blog.

The first release keeps the product intentionally small:

- Source content is local.
- Generated output is local.
- Rendering happens ahead of time.
- The public site is served as static files.
- Styxpress does not upload to a server.
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

## Site Config

`content/site.toml` controls only basic presentation:

- site title
- site description
- favicon
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
- `cover.*` is optional.
- `assets/` contains post-local files.

Post status is either `draft` or `published`.

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

## Admin Scope

The admin server:

- binds locally by default
- manages saved site configs
- reads and writes configured local content folders
- writes generated files to configured local public folders
- serves the embedded Vue admin UI

The admin server should not expose broad filesystem access to the browser.
Uploads are accepted only through explicit cover and asset endpoints.

## Deferred

These are not part of the first release:

- remote publishing
- server-backed content
- SSH/SFTP
- remote verification
- featured posts
- custom CSS
- theme presets
- saved themes
- dynamic public serving
- comments
- search
- analytics
