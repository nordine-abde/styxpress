# Admin API

All endpoints require the session token in either `X-Styxpress-Session` or
`Authorization: Bearer <token>`. The embedded admin UI receives this token
automatically; direct API clients can use the token printed by
`styxpress-admin`.

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
  Returns the active local site config.
- `POST /api/config`
  Saves the active local site config.

Config object:

```json
{
  "name": "My Blog",
  "contentDir": "content",
  "publicDir": "public"
}
```

`contentDir` is always local source content. `publicDir` is always local output.

## Sites

- `GET /api/sites`
  Returns saved site configs, `activeSiteId`, and `multiSite`.
- `POST /api/sites`
  Creates a site from `{"name":"My site","config":{...}}` and selects it.
- `POST /api/sites/{id}/select`
  Selects an existing site.
- `DELETE /api/sites/{id}`
  Deletes a site. Deleting the last site leaves no active site.

## Site Config

- `GET /api/site-config`
  Returns `site.toml` from the configured `contentDir`, or defaults when the
  file does not exist.
- `POST /api/site-config`
  Saves `site.toml` under the configured `contentDir`.
- `POST /api/site-config/preview`
  Body is a site config object. Returns `{"html":"..."}` for a draft homepage
  preview without writing public files.

Site config object:

```json
{
  "title": "Styxpress",
  "description": "Latest posts",
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
`mailto`.

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
  "updatedAt": "2026-04-26T12:00:00Z",
  "publishStatus": "published"
}
```

`publishedAt` and `updatedAt` are optional on save. Existing posts preserve
`publishedAt` and update `updatedAt`. `publishStatus` is returned as `draft` or
`published`.

## Uploads

- `POST /api/posts/{slug}/cover`
  Multipart form with `file`. Filename must be one of `cover.jpg`,
  `cover.jpeg`, `cover.png`, `cover.webp`, or `cover.avif`.
- `DELETE /api/posts/{slug}/cover`
  Removes the current cover.
- `POST /api/posts/{slug}/assets`
  Multipart form with `file` and optional `path`.
- `DELETE /api/posts/{slug}/assets/{assetPath...}`
  Removes one managed asset.

Asset paths are cleaned and must remain inside
`content/posts/{slug}/assets`.

## Preview And Render

- `POST /api/render-preview`
  Body is a post object. Returns `{"html":"..."}` without writing public files.
- `POST /api/posts/{slug}/render`
  Renders an already published post, homepage, feed, sitemap, and stylesheet
  locally. Returns `{"post":{...},"site":{...}}`.
- `POST /api/posts/{slug}/publish`
  Marks a draft as published, then renders the post and site locally. Returns
  `{"post":{...},"site":{...}}`.
- `POST /api/site/render`
  Renders all public posts, homepage, feed, sitemap, and stylesheet locally.
  Returns `{"posts":[...],"site":{...}}`.
