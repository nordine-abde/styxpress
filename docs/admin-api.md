# Admin API

All endpoints require the session token in either `X-Styxpress-Session` or
`Authorization: Bearer <token>`. The embedded admin UI asks for the local token
printed by `styxpress-admin`; direct API clients can use the same token.

Errors use this shape:

```json
{
  "error": {
    "code": "invalid_content",
    "message": "invalid slug"
  }
}
```

JSON request bodies reject unknown fields.

## Health

- `GET /api/health`
  Returns `{"status":"ok"}` when the token is valid.

## Config

- `GET /api/config`
  Returns the active admin config.
- `POST /api/config`
  Saves the active admin config.

Config object:

```json
{
  "name": "My Blog",
  "contentDir": "content",
  "publicDir": "public",
  "deploy": {
    "enabled": false,
    "sftp": {
      "host": "",
      "port": 22,
      "user": "",
      "remotePath": "",
      "keyPath": "",
      "knownHostsPath": ""
    }
  }
}
```

`contentDir` is always local source content. `publicDir` is always local output.
When deploy is enabled, `sftp.host`, `sftp.user`, and `sftp.remotePath` are
required. The configured remote folder is fully managed by Styxpress during
manual deploy. SFTP passwords and encrypted key passphrases are never part of
this object.

## Sites

- `GET /api/sites`
  Returns saved site configs, `activeSiteId`, and `multiSite`.
- `GET /api/sites/suggestion?name=My%20site`
  Returns the normalized unique id and default local folders for a new site.
- `POST /api/sites`
  Creates a site from `{"name":"My site","config":{...}}`, initializes its
  local folders, writes a default `site.toml`, renders default local public
  pages, and selects it.
- `POST /api/sites/{id}/select`
  Selects an existing site.
- `DELETE /api/sites/{id}`
  Deletes a site config entry. Deleting the last site leaves no active site.

When a multi-site admin session creates a site without explicit folders,
Styxpress defaults to `~/Styxpress/<site-id>/content` and
`~/Styxpress/<site-id>/public`.

When `styxpress-admin` runs with `-config`, the sites endpoint returns one
single-site entry and create/select/delete/suggestion endpoints are unavailable.

## Site Config

- `GET /api/site-config`
  Returns `site.toml` from the configured `contentDir`, or defaults when the
  file does not exist.
- `POST /api/site-config`
  Saves `site.toml` under the configured `contentDir`.
- `POST /api/site-config/preview`
  Body is a site config object. Returns `{"html":"..."}` for a draft homepage
  preview without writing public files.
- `POST /api/site-config/favicon`
  Multipart form with `file`. The file must be `.ico`; it is saved under
  `content/assets/` and selected in `site.toml`.
- `DELETE /api/site-config/favicon`
  Restores the built-in default favicon.

Site config object:

```json
{
  "title": "Styxpress",
  "description": "Latest posts",
  "favicon": "favicon.ico",
  "header": {
    "links": [
      { "label": "Home", "href": "/" },
      { "label": "RSS", "href": "/feed.xml" }
    ]
  },
  "footer": {
    "text": "Local notes.",
    "showWatermark": true,
    "links": [
      { "label": "RSS", "href": "/feed.xml" }
    ]
  }
}
```

Allowed link hrefs are root-relative paths, anchors, `http`, `https`, and
`mailto`. Favicon paths must be relative `.ico` paths.

## Posts

- `GET /api/posts`
  Returns `{"posts":[...]}` without `source` bodies.
- `GET /api/posts/{slug}`
  Returns one post including `source`.
- `POST /api/posts`
  Creates or updates a post using the slug from the body.
- `POST /api/posts/{slug}`
  Creates or updates a post using the slug from the URL. If the body also
  contains `slug`, it must match the URL slug.

Post object:

```json
{
  "slug": "hello-world",
  "title": "Hello World",
  "description": "Optional summary",
  "source": "# Hello\n",
  "cover": "cover.jpg",
  "assets": ["diagram.png"],
  "publishedAt": "2026-04-26T12:00:00Z",
  "updatedAt": "2026-04-26T12:00:00Z",
  "publishStatus": "published"
}
```

`slug`, `title`, and `source` are required for a valid saved post. Slugs contain
lowercase ASCII letters, numbers, and hyphens.

Saving a new post creates a draft. Saving an existing post preserves its
current draft/published state and updates `updatedAt`. Clients should use
`POST /api/posts/{slug}/publish` to publish; `publishedAt`, `updatedAt`, and
`publishStatus` are returned as state and are not used to force publishing in
the save endpoint.

## Uploads

- `GET /api/posts/{slug}/cover`
  Returns the current cover file for admin previews.
- `POST /api/posts/{slug}/cover`
  Multipart form with `file`. Supported extensions are `.jpg`, `.jpeg`,
  `.png`, `.webp`, and `.avif`. The uploaded file is saved as
  `cover.<extension>`.
- `DELETE /api/posts/{slug}/cover`
  Removes the current cover.
- `GET /api/posts/{slug}/assets/{assetPath...}`
  Returns one managed post asset for admin previews.
- `POST /api/posts/{slug}/assets`
  Multipart form with `file` and optional `path`. Supported uploaded asset
  extensions are `.jpg`, `.jpeg`, `.png`, `.webp`, `.avif`, and `.gif`.
- `DELETE /api/posts/{slug}/assets/{assetPath...}`
  Removes one managed asset.

Uploads are limited to 64 MiB. Asset paths are cleaned and must remain inside
`content/posts/{slug}/assets`.

## Preview And Render

- `POST /api/render-preview`
  Body is a post object. Returns `{"html":"..."}` without writing public files.
- `POST /api/posts/{slug}/render`
  Renders an already published post, homepage, feed, sitemap, stylesheet, and
  favicon locally. Draft posts return an error. Returns
  `{"post":{...},"site":{...}}`.
- `POST /api/posts/{slug}/publish`
  Marks a draft as published, then renders the post and site locally. The admin
  UI calls this as part of the single post **Save** action. Returns
  `{"post":{...},"site":{...}}`.
- `POST /api/site/render`
  Renders all public posts, homepage, feed, sitemap, stylesheet, and favicon
  locally. The admin UI calls this as part of the single site **Save** action.
  Returns `{"posts":[...],"site":{...}}`.

Render post result:

```json
{
  "slug": "hello-world",
  "publicDir": "/abs/site/public/posts/hello-world",
  "indexPath": "/abs/site/public/posts/hello-world/index.html",
  "coverPath": "/abs/site/public/posts/hello-world/cover.jpg",
  "assets": ["diagram.png"]
}
```

Render site result:

```json
{
  "indexPath": "/abs/site/public/index.html",
  "feedPath": "/abs/site/public/feed.xml",
  "sitemapPath": "/abs/site/public/sitemap.xml",
  "stylesheetPath": "/abs/site/public/assets/styxpress.css",
  "faviconPath": "/abs/site/public/favicon.ico"
}
```

## Deploy

- `GET /api/deploy/status`
  Returns whether deploy is enabled and configured, whether a session secret is
  set, and a local out-of-sync summary when possible.
- `POST /api/deploy/setup`
  Body is `{"config":{...},"secret":"","confirmRemoteOverwrite":false}`. Tests
  the proposed SFTP config before the first save. When the remote folder already
  has files, returns `{"requiresConfirmation":true,"remoteFiles":N}` without
  saving. Retrying with `confirmRemoteOverwrite:true` saves the config and
  immediately syncs local public output to the remote folder.
- `POST /api/deploy/secret`
  Body is `{"secret":"..."}`. Verifies a password or encrypted-key passphrase
  against the active SFTP config before keeping it in the running server session
  only. Returns `{"secretSet":true}`.
- `DELETE /api/deploy/secret`
  Clears the session deploy secret. Returns `{"secretSet":false}`.
- `POST /api/deploy`
  Runs a manual SFTP sync of `publicDir` to the configured remote path. Deploy
  must be enabled and configured. Matching remote files are overwritten and
  remote files missing locally are removed. Returns a deploy summary.

Deploy status response:

```json
{
  "enabled": true,
  "configured": true,
  "outOfSync": true,
  "secretSet": false,
  "summary": {
    "outOfSync": true,
    "uploaded": 1,
    "updated": 0,
    "deleted": 0,
    "unchanged": 0,
    "localFiles": 1,
    "remoteOnly": 0
  }
}
```

Deploy summaries use this shape:

```json
{
  "outOfSync": false,
  "uploaded": 1,
  "updated": 2,
  "deleted": 0,
  "unchanged": 10,
  "localFiles": 13,
  "remoteOnly": 0
}
```
