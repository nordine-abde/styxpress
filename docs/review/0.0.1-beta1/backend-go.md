# Backend Go Review

Incremental document for `cmd/styxpress-admin`, `internal`, and Go packages.

## Analysis Notes

- FND-001 fixed: config validation, API config saves, and renderer construction
  reject equal, nested, or symlink-overlapping content/public roots.
- FND-004 confirmed: feed, sitemap, canonical, and Open Graph call
  `absoluteURL`, but the function returns relative paths.
- FND-006 fixed: cover preview and media serving reject symlinked files and
  symlinked parent components under the content root.
- FND-007 fixed for content and rendering paths: repository operations and
  public output writes/deletes/copies now verify parent components before
  filesystem mutation.
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
