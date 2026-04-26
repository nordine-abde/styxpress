# Admin API

All endpoints require the session token printed by `styxpress-admin` in either
`X-Styxpress-Session` or `Authorization: Bearer <token>`.

Errors use this shape:

```json
{
  "error": {
    "code": "invalid_content",
    "message": "invalid slug"
  }
}
```

## Config

- `GET /api/config`
  Returns the saved config or defaults.
- `POST /api/config`
  Saves a config object. SSH passphrases are not part of the config and are not
  stored.
- `POST /api/test-ssh`
  Body: `{"passphrase":"optional"}`. Tests the configured SSH key and host.

Config object:

```json
{
  "siteBaseUrl": "https://blog.example.com",
  "contentDir": "content",
  "publicDir": "public",
  "contentStorageMode": "local",
  "remoteHost": "example.com:22",
  "remoteUser": "deploy",
  "sshKeyPath": "/home/user/.ssh/id_ed25519",
  "remotePublicDir": "/srv/site/public",
  "remoteContentDir": "/srv/site/content"
}
```

## Posts

- `GET /api/posts`
  Returns `{"posts":[...]}` without `source` bodies.
- `GET /api/posts/{slug}`
  Returns one post including `source`.
- `POST /api/posts`
  Creates or updates a post using the slug from the body.
- `POST /api/posts/{slug}`
  Creates or updates a post using the slug from the URL.

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
  "updatedAt": "2026-04-26T12:00:00Z"
}
```

`publishedAt` and `updatedAt` are optional on save. Existing posts preserve
`publishedAt` and update `updatedAt`.

## Uploads

- `POST /api/posts/{slug}/cover`
  Multipart form with `file`. Filename must be one of `cover.jpg`,
  `cover.jpeg`, `cover.png`, `cover.webp`, or `cover.avif`.
- `DELETE /api/posts/{slug}/cover`
  Removes the current cover.
- `POST /api/posts/{slug}/assets`
  Multipart form with `file` and optional `path`. Asset paths are always cleaned
  and must remain inside `content/posts/{slug}/assets`.
- `DELETE /api/posts/{slug}/assets/{assetPath...}`
  Removes one managed asset.

The API does not accept arbitrary read paths. Local file access is limited to
configured `contentDir` and `publicDir`, plus files explicitly uploaded through
multipart requests.

## Preview, Render, Publish, Featured

- `POST /api/render-preview`
  Body is a post object. Returns `{"html":"..."}` without writing public files.
- `POST /api/posts/{slug}/render`
  Renders the post, homepage, feed, and sitemap locally.
- `POST /api/posts/{slug}/publish`
  Body: `{"passphrase":"optional"}`. Renders locally, then publishes configured
  `publicDir`, and `contentDir` when `contentStorageMode` is `server`.
- `POST /api/publish`
  Body: `{"slug":"hello-world","passphrase":"optional"}`.
- `GET /api/featured`
  Returns `{"slugs":["hello-world"]}`.
- `POST /api/featured`
  Saves `{"slugs":["hello-world"]}` to `content/featured.txt`.
