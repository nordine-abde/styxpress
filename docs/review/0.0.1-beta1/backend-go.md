# Backend Go Review

Incremental document for `cmd/styxpress-admin`, `internal`, and Go packages.

## Analysis Notes

- FND-001 confirmed: there is no barrier between the source directory and the
  public output directory. The risk is data loss during `RenderAll`, especially
  for drafts.
- FND-004 confirmed: feed, sitemap, canonical, and Open Graph call
  `absoluteURL`, but the function returns relative paths.
- FND-006 confirmed: the cover preview uses `os.ReadFile` and can follow
  symlinks, unlike `getCover` and `copyFile`.
- FND-007 confirmed: several helpers check the final file but not parent
  symlinks, so writes and deletions can escape the configured roots.
- FND-008 confirmed: `RenderAll` only cleans output for drafts that are still
  present; orphaned output for deleted/renamed posts remains public.
- FND-009 confirmed: config saves are not atomic and site id creation has no
  lock/`O_EXCL`.
- FND-022 confirmed: in `-config` mode, relative paths are resolved against the
  working directory, not the config file.

## Areas Without A Verifiable Finding

- Direct traversal through slugs and asset paths: `ValidateSlug` and
  `CleanAssetPath` reject slashes, absolute paths, `.`, and `..`.
- Raw Markdown HTML in public rendering: the custom Goldmark renderer escapes
  HTML blocks/raw HTML and tests cover raw script/div input.
