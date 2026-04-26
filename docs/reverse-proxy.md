# Reverse Proxy Examples

Styxpress v1 serves public pages as pre-rendered static files. The public web server should use the generated `public/` directory as its document root and should not have read access to `content/`, admin configuration, SSH keys, or temporary upload directories.

Copy one of these examples and replace `example.com` and `/var/lib/styxpress/site/public` with your domain and generated public directory:

- [Caddyfile](examples/Caddyfile)
- [nginx.conf](examples/nginx.conf)

## Public Routes

The examples intentionally allow only these routes:

```text
GET /
GET /feed.xml
GET /sitemap.xml
GET /posts/{slug}
GET /posts/{slug}/cover.{jpg,jpeg,png,webp,avif}
GET /posts/{slug}/assets/{path}
```

Everything else returns `404`.

The examples assume slugs contain lowercase ASCII letters, numbers, and hyphens. That matches the renderer and keeps URL-to-file mapping predictable.

## Filesystem Layout

A typical VPS layout is:

```text
/var/lib/styxpress/site/
  content/       private source files
  public/        generated public files served by the reverse proxy
```

The reverse proxy root must be the `public/` directory, not `/var/lib/styxpress/site`.

## Permissions

Use separate users for serving public files and publishing new files.

Example:

```bash
sudo groupadd --system styxpress
sudo useradd --system --home /var/lib/styxpress --shell /usr/sbin/nologin styxpress-publish
sudo usermod -a -G styxpress www-data
sudo install -d -o styxpress-publish -g styxpress -m 0750 /var/lib/styxpress/site
sudo install -d -o styxpress-publish -g styxpress -m 0750 /var/lib/styxpress/site/content
sudo install -d -o styxpress-publish -g styxpress -m 0750 /var/lib/styxpress/site/public
sudo find /var/lib/styxpress/site/public -type d -exec chmod 0750 {} +
sudo find /var/lib/styxpress/site/public -type f -exec chmod 0640 {} +
```

For Nginx on Debian and Ubuntu, the web server user is usually `www-data`. For Caddy packages, it is often `caddy`; replace `www-data` with that user if needed.

The publishing SSH user needs write access to `content/` and `public/`. The reverse proxy user needs read and execute access only to `public/`.

## Hidden Files And Symlinks

The examples return `404` for hidden files and dot-directories. Do not publish secrets, local admin configuration, SSH material, or upload scratch files into `public/`.

Do not place symlinks in `public/`. Nginx can reject symlinks with `disable_symlinks`; the Caddy example relies on the document root, route allowlist, and filesystem hygiene. Before enabling the site, you can check for symlinks with:

```bash
find /var/lib/styxpress/site/public -type l -print
```

That command should print nothing.

## Manual Verification

After publishing, verify the intended routes:

```bash
curl -I https://example.com/
curl -I https://example.com/feed.xml
curl -I https://example.com/sitemap.xml
curl -I https://example.com/posts/hello-world
```

Then verify blocked paths:

```bash
curl -I https://example.com/content/
curl -I https://example.com/.env
curl -I https://example.com/posts/hello-world/source.md
curl -I https://example.com/posts/hello-world/
curl -I https://example.com/posts/../config.toml
```

The blocked paths should return `404`.
