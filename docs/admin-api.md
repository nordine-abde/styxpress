# Admin API

All endpoints require the session token in either `X-Styxpress-Session` or
`Authorization: Bearer <token>`. The embedded admin UI receives this token
automatically; direct API clients can use the token printed by `styxpress-admin`.

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

## Site Config

- `GET /api/site-config`
  Returns `site.toml` from the configured `contentDir`, or defaults when the
  file does not exist.
- `POST /api/site-config`
  Saves the site presentation config as `site.toml` under the configured
  `contentDir` and returns the normalized config.
- `POST /api/site-config/preview`
  Body is a site config object. Returns `{"html":"..."}` for a draft homepage
  preview without writing public files.
- `POST /api/site-config/style-guide`
  Body is a site config object. Returns a renderer-derived custom CSS starter
  and editable selectors for the draft theme without writing public files. The
  `starterCss` value is generated from the backend renderer stylesheet and
  intentionally excludes `theme.customCss` and `savedThemes[].customCss`.

Style guide response:

```json
{
  "starterCss": "/* Styxpress custom CSS starter. */\n:root { ... }\n",
  "customCssIncluded": false,
  "bodyClasses": ["theme-midnight", "font-mono", "layout-wide", "radius-none"],
  "selectors": [
    {
      "selector": "body.theme-midnight.font-mono.layout-wide.radius-none",
      "kind": "body",
      "current": true,
      "description": "Current body class combination rendered on every page."
    },
    {
      "selector": ".site-header-minimal",
      "className": "site-header-minimal",
      "kind": "headerVariant",
      "current": false,
      "description": "Header variant class."
    }
  ],
  "headerVariants": [
    {
      "variant": "nav",
      "selector": ".site-header-nav",
      "className": "site-header-nav",
      "current": true,
      "rendered": true
    }
  ],
  "footerVariants": [
    {
      "variant": "simple",
      "selector": ".site-footer-simple",
      "className": "site-footer-simple",
      "current": true,
      "rendered": true
    }
  ],
  "notes": [
    "starterCss is generated from the renderer base stylesheet and intentionally excludes theme.customCss and savedThemes[].customCss."
  ]
}
```

Site config object:

```json
{
  "title": "Styxpress",
  "description": "Latest posts",
  "theme": {
    "palette": "ink",
    "font": "system",
    "layout": "classic",
    "radius": "soft",
    "customCss": ".site-main { max-width: 68rem; }"
  },
  "savedThemes": [
    {
      "id": "quiet-serif",
      "name": "Quiet Serif",
      "palette": "sage",
      "font": "serif",
      "layout": "classic",
      "radius": "soft",
      "customCss": ""
    }
  ],
  "header": {
    "variant": "nav",
    "title": "",
    "tagline": "",
    "links": [
      { "label": "Home", "href": "/" },
      { "label": "RSS", "href": "/feed.xml" }
    ]
  },
  "footer": {
    "variant": "simple",
    "text": "Published with Styxpress",
    "links": [
      { "label": "RSS", "href": "/feed.xml" }
    ]
  }
}
```

Allowed theme values are `palette` `ink`, `sage`, `clay`, or `midnight`; `font`
`system`, `serif`, or `mono`; `layout` `classic` or `wide`; and `radius` `none`
or `soft`. Saved themes are root-level entries and include their own
`customCss`.

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
  Renders the post, homepage, feed, sitemap, and stylesheet locally. Returns
  `{"post":{...},"site":{...}}`.
- `POST /api/posts/{slug}/publish`
  Body: `{"passphrase":"optional"}`. Renders locally, then publishes configured
  `publicDir`, and `contentDir` when `contentStorageMode` is `server`. Returns
  `{"post":{...},"site":{...},"publish":{...}}`.
- `POST /api/publish`
  Body: `{"slug":"hello-world","passphrase":"optional"}`. Equivalent to
  `POST /api/posts/{slug}/publish` with the slug in the body.
- `POST /api/site/render`
  Renders all public posts, the homepage, feed, sitemap, and stylesheet locally.
  Returns `{"posts":[...],"site":{...}}`.
- `POST /api/site/publish`
  Body: `{"passphrase":"optional"}`. Renders all public pages locally, then
  publishes configured `publicDir`, and `contentDir` when `contentStorageMode`
  is `server`. Returns `{"posts":[...],"site":{...},"publish":{...}}`.
- `GET /api/featured`
  Returns `{"slugs":["hello-world"]}`.
- `POST /api/featured`
  Saves `{"slugs":["hello-world"]}` to `content/featured.txt`.
