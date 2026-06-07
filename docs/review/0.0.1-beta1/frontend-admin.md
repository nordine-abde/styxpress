# Frontend Admin Review

Incremental document for `admin/web`.

## Analysis Notes

- FND-003 fixed: media upload/delete no longer calls the publish endpoint and
  only renders already-published posts through the dedicated render endpoint.
- FND-010 fixed: the HTML preview iframe is now sandboxed with no granted
  capabilities.
- FND-011 confirmed: invalid token/logout do not reliably clear API-backed
  stores.
- FND-012 confirmed: `loadCurrentSite` can mark a site as loaded even when
  posts/site config loading fails.
- FND-013 confirmed: `ConfigScreen` has a local form with no dirty tracking in
  the global guard.
- FND-014 confirmed: `selectPost` does not protect against races between
  concurrent fetches.
- FND-015 confirmed: `apiRequest` performs non-defensive JSON parsing.
- FND-016 confirmed: unguarded `decodeURI` can break image refresh.
- FND-017 confirmed: `buildStore.lastResult` can leak across site changes.
