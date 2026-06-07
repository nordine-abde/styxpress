# Findings

This document is incremental. Findings are sorted by identifier, not necessarily
by severity.

| ID | Severity | Area | Status | Title |
| --- | --- | --- | --- | --- |
| FND-001 | Critical | Backend/rendering/config | Fixed | `publicDir` can equal `contentDir` and delete source content |
| FND-002 | High | Security/dependencies/SFTP | Fixed | Vulnerable `golang.org/x/crypto` with reachable SSH calls |
| FND-003 | High | Frontend/content workflow | Fixed | Image upload or deletion automatically publishes drafts |
| FND-004 | Medium | Rendering/feed/sitemap | Confirmed | Feed, sitemap, and canonical URLs use relative URLs instead of absolute URLs |
| FND-005 | High | Security/admin bootstrap | Fixed | The unauthenticated SPA receives the admin bearer token |
| FND-006 | High | Backend/preview/filesystem | Fixed | Cover preview follows symlinks and can read local files |
| FND-007 | Medium | Backend/filesystem/symlink | Fixed | Symlinks in intermediate path components can redirect writes and deletes outside roots |
| FND-008 | Medium | Rendering/cleanup | Confirmed | Output for deleted or renamed posts remains public |
| FND-009 | Low | Config/concurrency | Confirmed | Config saves are non-atomic and site creation is race-prone |
| FND-010 | Medium | Frontend/security hardening | Fixed | HTML preview iframe has no sandbox |
| FND-011 | Medium | Frontend/auth/state | Confirmed | Invalid tokens and logout leave API-backed data visible |
| FND-012 | Medium | Frontend/state loading | Confirmed | Workspace is marked loaded even if posts or site config fail |
| FND-013 | Medium | Frontend/unsaved changes | Confirmed | The Configuration screen does not participate in the unsaved changes guard |
| FND-014 | Medium | Frontend/race condition | Confirmed | Post selection is race-prone between concurrent requests |
| FND-015 | Medium | Frontend/API client | Confirmed | `apiRequest` fails with `SyntaxError` on non-JSON responses |
| FND-016 | Low | Frontend/editor robustness | Confirmed | Malformed percent-encoding in image src breaks image refresh |
| FND-017 | Low | Frontend/multi-site state | Confirmed | `Last local build` can show the previous site's result |
| FND-018 | High | Release/security/toolchain | Fixed | Release builds can use Go 1.22.2 with reachable standard library vulnerabilities |
| FND-019 | Medium | Security/deploy/secrets | Confirmed | Global SFTP secret is reusable across sites/configurations |
| FND-020 | Medium | Deploy/status | Confirmed | Local deploy state is treated as remote truth |
| FND-021 | Low | Release/frontend supply chain | Fixed | Frontend install/build are non-deterministic and have no audit gate |
| FND-022 | Medium | Config/path resolution | Confirmed | Relative paths in `-config` mode depend on the working directory |

## FND-001 - `publicDir` can equal `contentDir` and delete source content

**Severity:** Critical

**Area:** Backend/rendering/config

**Status:** Fixed. Config validation, API config saves, and renderer
construction now reject equal, nested, or symlink-overlapping content/public
roots before rendering can write or delete public output.

**References:**

- `internal/config/config.go:128`: `Config.Validate` checks NUL bytes, deploy
  mode, and SFTP fields, but does not forbid `ContentDir == PublicDir` or
  overlapping directories.
- `internal/api/server.go:1175`: `configuredPath` only applies `filepath.Abs`
  and `filepath.Clean`.
- `internal/rendering/renderer.go:247`: `RenderAll` calls
  `removeUnpublishedPostOutput`.
- `internal/rendering/renderer.go:490`: `removeUnpublishedPostOutput` runs
  `os.RemoveAll(filepath.Join(r.publicRoot, "posts", post.Slug))` for every
  unpublished post.
- `internal/rendering/renderer.go:596`: `reconcileAssets` removes the entire
  public asset directory before copying it again.

**Impact:**

If the user configures `publicDir` equal to `contentDir`, or points it at the
content root, rendering treats `content/posts/<slug>` as public output. For each
draft, `removeUnpublishedPostOutput` can delete the entire source directory for
the post, including `source.md`, metadata, and assets. Published posts also get
`index.html` and generated assets written into the same tree as the sources.

**Evidence:**

Config validation does not enforce separation between source content and public
output. The renderer directly joins `publicRoot/posts/<slug>` and removes that
directory for drafts without checking whether it is outside `contentRoot`.

**Resolution:**

Normalize `contentDir` and `publicDir` to absolute paths resolved with
`EvalSymlinks`, then reject equal or dangerously overlapping configurations.
Regression tests cover same-root and symlink-overlap cases in config, rendering,
and API config saves.

## FND-002 - Vulnerable `golang.org/x/crypto` with reachable SSH calls

**Severity:** High

**Area:** Security/dependencies/SFTP

**Status:** Fixed. `golang.org/x/crypto` is upgraded to `v0.52.0`; related
`golang.org/x/sys` checksums were updated through module resolution.

**References:**

- `go.mod:8`: the project requires `golang.org/x/crypto v0.31.0`.
- `internal/deploy/sftp.go:252`: `deploy.connect` calls `ssh.NewClientConn`.
- `internal/deploy/sftp.go:263`: `deploy.connect` calls `sftp.NewClient`, which
  uses SSH primitives.
- `internal/deploy/sftp.go:362`: `signerFromKeyFile` calls
  `ssh.ParsePrivateKey`.
- `internal/deploy/sftp.go:367`: `signerFromKeyFile` calls
  `ssh.ParsePrivateKeyWithPassphrase`.
- `internal/deploy/sftp.go:385`: `hostKeyCallback` calls `knownhosts.New`.

**Impact:**

`govulncheck` reports reachable calls to known vulnerabilities in
`golang.org/x/crypto@v0.31.0`, all on the SSH/SFTP deploy surface. For a beta
that includes SFTP deploy, this dependency must be upgraded before release.

**Evidence:**

`go run golang.org/x/vuln/cmd/govulncheck@latest ./...` reports 9 reachable
vulnerabilities in module `golang.org/x/crypto@v0.31.0`: GO-2026-5021,
GO-2026-5020, GO-2026-5019, GO-2026-5018, GO-2026-5017, GO-2026-5015,
GO-2026-5013, GO-2025-4116, GO-2025-3487. The first seven are fixed in
`golang.org/x/crypto@v0.52.0`.

**Resolution:**

`golang.org/x/crypto` was upgraded to `v0.52.0`. Verification includes Go tests
and `govulncheck` on the updated tree.

## FND-003 - Image upload or deletion automatically publishes drafts

**Severity:** High

**Area:** Frontend/content workflow

**Status:** Fixed. Media upload/delete now renders already-published posts
through the dedicated render endpoint and does not call the publish endpoint for
drafts.

**References:**

- `admin/web/src/components/PostAssetPanel.vue:56`: after asset upload, it calls
  `renderCurrentPost`.
- `admin/web/src/components/PostAssetPanel.vue:70`: after cover upload, it calls
  `renderCurrentPost`.
- `admin/web/src/components/PostAssetPanel.vue:75`: after cover deletion, it
  calls `renderCurrentPost`.
- `admin/web/src/components/PostAssetPanel.vue:80`: after asset deletion, it
  calls `renderCurrentPost`.
- `admin/web/src/components/PostAssetPanel.vue:96`: `renderCurrentPost` calls
  `buildStore.publishPost(postsStore.selectedSlug)`.
- `admin/web/src/stores/build.js:21`: `publishPost` sends
  `POST /api/posts/{slug}/publish`.
- `internal/api/server.go:766`: the publish endpoint calls
  `repo.MarkPostPublished`.
- `internal/content/repository.go:220`: `MarkPostPublished` writes
  `published_at.txt`.

**Impact:**

A user preparing a draft who uploads a cover, uploads an image, or removes media
unintentionally publishes the post. This changes the state from draft to
published, generates public output, and can trigger auto deploy if configured.

**Evidence:**

The frontend flow uses a function named `renderCurrentPost`, but the store
function it invokes is `publishPost`, not a non-mutating render operation. The
corresponding backend endpoint marks the post as published before rendering.

**Resolution:**

The media panel no longer calls `/publish` after upload/delete. It refreshes the
post state and calls the non-mutating render endpoint only when the selected
post is already published. Backend tests verify that rendering a draft returns
400 and does not create `published_at.txt` or public output.

## FND-004 - Feed, sitemap, and canonical URLs use relative URLs instead of absolute URLs

**Severity:** Medium

**Area:** Rendering/feed/sitemap

**References:**

- `internal/rendering/renderer.go:371`: `CanonicalURL` uses
  `r.absoluteURL(basePostURL)`.
- `internal/rendering/renderer.go:442`: the RSS channel link uses
  `r.absoluteURL("/")`.
- `internal/rendering/renderer.go:449`: RSS items use `r.postURL(post.Slug)`.
- `internal/rendering/renderer.go:467`: the sitemap home entry uses
  `r.absoluteURL("/")`.
- `internal/rendering/renderer.go:471`: sitemap post URLs use
  `r.postURL(post.Slug)`.
- `internal/rendering/renderer.go:622`: `absoluteURL` returns the input path
  directly.

**Impact:**

RSS, sitemap, canonical, and Open Graph contain paths such as `/` and
`/posts/slug/` instead of absolute URLs. XML sitemaps require absolute URLs, and
many RSS/SEO consumers handle relative links poorly.

**Evidence:**

There is no `baseURL`/`siteURL` field in `siteconfig.Config` or in the
documentation. The function named `absoluteURL` does not build an absolute URL.

**Suggested Fix:**

Add explicit configuration for the site's public URL, validate it as `http(s)`,
and use it for feed, sitemap, canonical, and Open Graph URLs.

## FND-005 - The unauthenticated SPA receives the admin bearer token

**Severity:** High

**Area:** Security/admin bootstrap

**Status:** Fixed. The admin SPA no longer receives the session token through
served HTML or JavaScript bootstrap.

**References:**

- `cmd/styxpress-admin/main.go:56`: `newHandler` mounts `/api/` and the SPA.
- `cmd/styxpress-admin/main.go:58`: `mux.Handle("/", embeddedSPA())` serves the
  SPA without passing the session token into the static handler.
- `cmd/styxpress-admin/main.go:70`: `embeddedSPA` reads the built `index.html`
  and serves it without token injection.
- `README.md:119`: documents `./styxpress-admin -addr 127.0.0.1:8080`, but the
  flag also allows non-loopback addresses.

**Impact:**

If the admin is started with `-addr 0.0.0.0:...`, or placed behind a reverse
proxy, any anonymous request to `/` receives the bearer token and can then call
all of `/api`. The token no longer protects the instance when the HTML that
contains it is served to untrusted clients.

**Original Evidence:**

API protection relies on `X-Styxpress-Session`/`Authorization`, but the secret is
automatically delivered by the unauthenticated SPA route.

**Resolution:**

The HTML bootstrap injection was removed. The frontend auth store now starts
without a token, and the user must paste the local token printed by the admin
process. The token is kept in frontend memory and added to API calls through
`X-Styxpress-Session`. `cmd/styxpress-admin/main_test.go` verifies that an
anonymous SPA response does not contain `window.__STYXPRESS_SESSION__`.

## FND-006 - Cover preview follows symlinks and can read local files

**Severity:** High

**Area:** Backend/preview/filesystem

**Status:** Fixed. Preview cover reads and authenticated cover/asset serving now
reject symlinked files and symlinked parent components under the content root.

**References:**

- `internal/content/repository.go:453`: `findCover` enumerates cover files by
  name without rejecting symlinks.
- `internal/rendering/renderer.go:386`: `previewCoverURL` prepares the cover for
  preview.
- `internal/rendering/renderer.go:393`: `previewCoverURL` uses direct
  `os.ReadFile`, which follows symlinks.
- `internal/api/server.go:540`: `getCover` uses `os.Lstat` and rejects
  symlinks, so preview is less restrictive than the asset route.
- `internal/rendering/renderer.go:710`: `copyFile` uses `os.Lstat` and rejects
  symlinks during public rendering.

**Impact:**

A malicious or compromised content repository can add
`content/posts/<slug>/cover.jpg` as a symlink to a local file readable by the
process. The preview renders that cover as a base64 data URL and returns it to
the admin UI, exposing the file contents.

**Evidence:**

Routes and public rendering reject symlinks, but `previewCoverURL` does not. The
difference shows that the case is treated as dangerous elsewhere but is missing
from preview.

**Resolution:**

`previewCoverURL` now reads cover files through a no-symlink path helper instead
of direct `os.ReadFile`. The API cover/asset serving path also opens media
through a helper that rejects unsafe components, symlinks, directories, and
paths that resolve outside the content root. Tests cover symlinked cover files
and symlinked post directories without leaking the target contents.

## FND-007 - Symlinks in intermediate path components can redirect writes and deletes outside roots

**Severity:** Medium

**Area:** Backend/filesystem/symlink

**Status:** Fixed for the content repository and rendering public output.
Repository write/delete paths and renderer write/delete/copy paths now verify
parent components before filesystem mutation.

**References:**

- `internal/rendering/renderer.go:299`: `renderLoadedPost` uses
  `os.MkdirAll(publicDir)`.
- `internal/rendering/renderer.go:596`: `reconcileAssets` uses
  `os.RemoveAll(publicAssetsDir)`.
- `internal/rendering/renderer.go:729`: `copyFile` creates temp files in the
  destination directory without checking parents.
- `internal/content/repository.go:582`: `writeReader` creates directories and
  opens the final path without checking parent symlinks.

**Impact:**

If an intermediate component such as `public/posts`, `public/posts/<slug>`, or
`content/posts/<slug>/assets` is a symlink to an external directory, writes, temp
files, renames, and deletions can happen outside the configured roots.

**Evidence:**

The code checks some final files with `Lstat`, but does not walk all parent
components. Filesystem calls on full paths traverse symlinks in parent
directories.

**Resolution:**

Content repository operations now walk parent components with `Lstat` before
creating, writing, reading, or deleting post, cover, and asset paths. Rendering
output operations now use root-aware helpers for public writes, atomic renames,
copy sources, asset reconciliation, and draft output cleanup. Regression tests
cover symlinked content parents, symlinked asset parents, symlinked public
parents during writes, and symlinked public parents during draft cleanup.

Residual risk: the helpers still have normal TOCTOU exposure between `Lstat` and
`OpenFile`/`Rename`/`Remove`; closing that fully would require platform-specific
`openat`/`O_NOFOLLOW` style APIs.

## FND-008 - Output for deleted or renamed posts remains public

**Severity:** Medium

**Area:** Rendering/cleanup

**References:**

- `internal/rendering/renderer.go:241`: `RenderAll` loads current posts.
- `internal/rendering/renderer.go:247`: `RenderAll` calls
  `removeUnpublishedPostOutput`.
- `internal/rendering/renderer.go:485`: cleanup only removes output for drafts
  that are still present.

**Impact:**

If a published post is deleted or renamed in the source tree, the already
generated directory under `public/posts/<old-slug>/` remains public. This can
leave content online that the user believes was removed.

**Evidence:**

Cleanup only reconciles drafts present in `ListPosts()`. It does not scan
`public/posts/*` and compare it against the current set of published slugs.

**Suggested Fix:**

During `RenderAll`, reconcile `public/posts/*` against the current published
slugs and remove orphaned output in a symlink-safe way.

## FND-009 - Config saves are non-atomic and site creation is race-prone

**Severity:** Low

**Area:** Config/concurrency

**References:**

- `internal/config/config.go:116`: `Save` opens the file with `O_TRUNC`.
- `internal/siteconfig/config.go:109`: `siteconfig.Save` opens the file with
  `O_TRUNC`.
- `internal/config/sites.go:324`: `nextSiteID` uses stat-then-save with no
  exclusion.

**Impact:**

Crashes, disk full conditions, or concurrent requests can leave config files
truncated or create collisions between sites with the same name. In a local
admin app the risk is limited, but before a beta the possibility of corrupting
primary files should be removed.

**Evidence:**

Saves write directly to the final file instead of writing to a temp file and
performing an atomic rename. Site creation computes the free id before saving,
without a mutex or `O_EXCL`.

**Suggested Fix:**

Use temp file + chmod + fsync + rename for config saves, introduce a mutex in
`SiteStore`, and create site files with atomic exclusion.

## FND-010 - HTML preview iframe has no sandbox

**Severity:** Medium

**Area:** Frontend/security hardening

**Status:** Fixed. The site preview iframe now has a restrictive empty
`sandbox` attribute.

**References:**

- `admin/web/src/stores/siteConfig.js:81`: `previewSiteConfig` receives HTML
  from the API.
- `admin/web/src/stores/siteConfig.js:152`: the HTML is converted into a
  `text/html` Blob.
- `admin/web/src/components/SiteConfigScreen.vue:156`: the iframe uses
  `:src="siteConfigStore.previewUrl"` without `sandbox`.

**Impact:**

Today the site config preview goes through `html/template` and validated links,
so no user input was found that produces script. However, a Blob HTML document
created by the app and loaded in an iframe without sandbox inherits the app's
origin context; if an XSS regression or future content preview introduced
script, that code would have a very weak barrier to the parent and local API.

**Evidence:**

The iframe sets no `sandbox` attribute. The code creates an object URL with MIME
type `text/html`.

**Resolution:**

`admin/web/src/components/SiteConfigScreen.vue` now sets `sandbox=""` on the
preview iframe, granting no script, same-origin, form, popup, or top-navigation
capabilities. A preview-specific CSP remains a possible hardening follow-up.

## FND-011 - Invalid tokens and logout leave API-backed data visible

**Severity:** Medium

**Area:** Frontend/auth/state

**References:**

- `admin/web/src/components/AuthBar.vue:23`: `applyToken` immediately sets the
  token in the store.
- `admin/web/src/stores/config.js:48`: `loadConfig` catches the error and does
  not rethrow it.
- `admin/web/src/App.vue:161`: the UI shows the session as ready based on
  `authStore.hasToken`.
- `admin/web/src/components/AuthBar.vue:35`: `clearToken` logs out but only
  resets post selection.

**Impact:**

An incorrect token can leave the UI in a "session ready" state with previous or
partial data. Logout or 401 does not clear all API-backed stores, so sites,
counts, config, deploy status, or previous results can remain visible in the
same tab.

**Evidence:**

Session state is derived from a non-empty token, not from successful validation.
`clearToken` does not reset `configStore`, `siteWorkspaceStore`,
`siteConfigStore`, `buildStore`, or `deployStore`.

**Suggested Fix:**

On logout and 401, reset all sensitive stores. Make `applyToken` fail explicitly
if validation receives 401, or introduce an `authenticated` state distinct from
`hasToken`.

## FND-012 - Workspace is marked loaded even if posts or site config fail

**Severity:** Medium

**Area:** Frontend/state loading

**References:**

- `admin/web/src/stores/siteWorkspace.js:34`: `loadCurrentSite` uses
  `Promise.all` over child loaders.
- `admin/web/src/stores/posts.js:75`: `loadPosts` catches errors without
  rethrowing them.
- `admin/web/src/stores/siteConfig.js:45`: `loadSiteConfig` catches errors
  without rethrowing them.
- `admin/web/src/stores/siteWorkspace.js:38`: `loadedSiteId` is set after
  `Promise.all`.

**Impact:**

If posts or site config fail, the workspace can still be marked loaded. Later
calls without `force` skip retry because `loadedSiteId` matches the current
site, leaving empty, default, or stale data.

**Evidence:**

Because child loaders swallow errors, `Promise.all` resolves even on application
failures.

**Suggested Fix:**

Make loaders used as dependencies rethrow errors, or return an explicit result
and set `loadedSiteId` only if all loads succeeded.

## FND-013 - The Configuration screen does not participate in the unsaved changes guard

**Severity:** Medium

**Area:** Frontend/unsaved changes

**References:**

- `admin/web/src/App.vue:47`: `hasUnsavedChanges` considers only
  `siteConfigStore.isDirty` and `postsStore.isDirty`.
- `admin/web/src/components/ConfigScreen.vue:20`: `ConfigScreen` uses a local
  form.
- `admin/web/src/App.vue:122`: `beforeunload` depends on `hasUnsavedChanges`.

**Impact:**

Changes to `contentDir`, `publicDir`, or SFTP settings can be lost by navigating
to other views or closing the page without a prompt.

**Evidence:**

There is no shared dirty tracking for `configStore`/`ConfigScreen`, and the
global guard cannot know that the local form was modified.

**Suggested Fix:**

Introduce `configStore.isDirty` or an equivalent mechanism, update it from
`ConfigScreen`, and include it in `hasUnsavedChanges` and confirmation resets.

## FND-014 - Post selection is race-prone between concurrent requests

**Severity:** Medium

**Area:** Frontend/race condition

**References:**

- `admin/web/src/components/PostList.vue:54`: `selectPost` calls
  `postsStore.selectPost(slug)` without awaiting or blocking later selections.
- `admin/web/src/stores/posts.js:83`: `selectPost` updates `draft` and
  `selectedSlug` as soon as the fetch completes.

**Impact:**

Rapid clicks on different posts can open the editor with the content from the
response that arrived last, not necessarily the last post clicked.

**Evidence:**

There are no request ids, abort controllers, or final checks on the requested
slug before applying the response.

**Suggested Fix:**

Use a sequence id or `AbortController` and apply the response only if it matches
the latest requested selection.

## FND-015 - `apiRequest` fails with `SyntaxError` on non-JSON responses

**Severity:** Medium

**Area:** Frontend/API client

**References:**

- `admin/web/src/api/client.js:29`: reads the response as text.
- `admin/web/src/api/client.js:30`: runs `JSON.parse(text)` before checking
  `response.ok`.
- `admin/web/src/api/client.js:56`: `apiBlobRequest` already has a defensive
  JSON fallback for errors.

**Impact:**

HTML/non-JSON responses, proxy errors, or a dev server without the backend
produce `SyntaxError` instead of an `ApiError` with status/code. This especially
hurts 401 handling and local troubleshooting.

**Evidence:**

`apiRequest` does not check `Content-Type` and does not catch parse errors.

**Suggested Fix:**

Parse JSON defensively, check `Content-Type`, and convert non-JSON responses
into `ApiError` with `response.status` and a readable message.

## FND-016 - Malformed percent-encoding in image src breaks image refresh

**Severity:** Low

**Area:** Frontend/editor robustness

**References:**

- `admin/web/src/components/VisualMarkdownEditor.vue:158`:
  `queueImageRefresh` calls `refreshEditorImages`.
- `admin/web/src/components/VisualMarkdownEditor.vue:330`:
  `assetPathFromMarkdownSrc` uses `decodeURI(path)` without `try/catch`.

**Impact:**

Imported Markdown with a malformed image src, for example
`![x](assets/%E0%A4%A)`, can throw an unhandled error and break image refresh in
the visual editor.

**Evidence:**

`decodeURI` throws `URIError` on invalid percent-encoding and the caller does
not catch it.

**Suggested Fix:**

Use `try/catch` around decoding, ignore invalid paths, and show a non-blocking
error if useful.

## FND-017 - `Last local build` can show the previous site's result

**Severity:** Low

**Area:** Frontend/multi-site state

**References:**

- `admin/web/src/components/SiteListScreen.vue:141`: `openActiveSite` resets
  `siteWorkspaceStore`.
- `admin/web/src/stores/build.js:59`: `buildStore.reset` exists but is not
  called in `openActiveSite`.
- `admin/web/src/components/SiteConfigScreen.vue:170`: the Site screen displays
  `buildStore.lastResult`.

**Impact:**

When opening another site, the "Last local build" section can show paths and
counts from the previous site.

**Evidence:**

`buildStore.reset()` is called in `App.leaveSite`, but not when opening or
selecting a different site from the list.

**Suggested Fix:**

Reset `buildStore` when opening/selecting a different site, together with the
workspace reset.

## FND-018 - Release builds can use Go 1.22.2 with reachable standard library vulnerabilities

**Severity:** High

**Area:** Release/security/toolchain

**References:**

- Original finding: `go.mod:3` declared `go 1.22.2`.
- Fixed state: `go.mod` declares `go 1.25.11` and prefers toolchain
  `go1.26.4`.
- Fixed state: `scripts/build-release.sh` and `scripts/run_admin.sh` call the
  shared Go toolchain gate before building.
- Review environment: `go version go1.22.2 linux/amd64`.
- Policy source: the Go project's security policy supports the latest two Go
  releases; Go 1.22 is no longer within that support window as of the review
  date. See https://github.com/golang/go/security/policy.

**Impact:**

A beta binary compiled with Go 1.22.2 inherits standard library vulnerabilities
already fixed in later releases. Relevant reachable surfaces in this project
include `html/template`, `net/http`, `net/url`, `crypto/tls`, `crypto/x509`,
`encoding/asn1`, and `encoding/pem`. This is separate from FND-002: even after
upgrading `golang.org/x/crypto`, an old toolchain can still produce vulnerable
binaries.

**Evidence:**

Command executed:

```text
go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
```

Result: 38 reachable vulnerabilities from the codebase using the local
`go1.22.2` toolchain. Among those reported:

- GO-2026-4982, GO-2026-4980, GO-2026-4865, GO-2026-4603 in `html/template`,
  with traces through `internal/rendering/renderer.go:433`.
- GO-2026-4341 and others in `net/url`/`net/http`, with traces through API and
  server code.
- GO-2026-4870, GO-2026-4340, GO-2026-4337 in `crypto/tls`.
- GO-2026-5037, GO-2026-4947, GO-2026-4946 and others in `crypto/x509`.

**Suggested Fix:**

Set a supported and updated toolchain in `go.mod`, update the release
environment, and make `scripts/build-release.sh` fail if `go version` or
`govulncheck ./...` are not acceptable. Run the gate with the same toolchain
used to build distributed binaries.

**Fix Applied:**

`go.mod` now encodes Go `1.25.11` as the minimum supported beta toolchain and
prefers Go `1.26.4`. `scripts/go-toolchain.sh` rejects unsupported Go versions
and stale patch versions before `scripts/build-release.sh` or
`scripts/run_admin.sh` builds anything. The gate accepts Go `1.25.11` or newer
on the 1.25 line, Go `1.26.4` or newer on the 1.26 line, or a newer supported
Go release.

## FND-019 - Global SFTP secret is reusable across sites/configurations

**Severity:** Medium

**Area:** Security/deploy/secrets

**References:**

- `internal/api/server.go:42`: `deploySecret` is a single server field.
- `internal/api/server.go:838`: `saveDeploySecret` saves the global secret.
- `internal/api/server.go:968`: `deployConfig` builds the active site's config.
- `internal/api/server.go:982`: every deploy config receives
  `s.deploySecretValue()`.
- `admin/web/src/stores/deploy.js:51`: `syncFromConfig` preserves `secretSet`
  even when the deploy configuration changes.

**Impact:**

A password or passphrase entered for one site can be accidentally reused when
deploying another site or after changing host/user/key. This creates operational
leakage, login attempts against the wrong host/account, and possible lockouts.

**Evidence:**

The secret is not associated with `siteID`, host, user, key path, or remote path.
The UI keeps `secretSet` across config changes when the signature is equal only
for other state fields.

**Suggested Fix:**

Bind the secret to `{siteID, host, user, keyPath}` and invalidate it
automatically on site change or deploy config change. Alternatively, always
clear it when the site changes or when the SFTP section is edited.

## FND-020 - Local deploy state is treated as remote truth

**Severity:** Medium

**Area:** Deploy/status

**References:**

- `internal/deploy/sftp.go:84`: `Status` prepares config and reads local files.
- `internal/deploy/sftp.go:93`: `Status` loads only the local state JSON.
- `internal/deploy/sftp.go:97`: `Status` returns
  `buildPlan(local, previous, ...)` without a remote connection.
- `internal/deploy/sftp.go:113`: `Sync` computes the plan from local state.
- `internal/deploy/sftp.go:114`: `Sync` returns early without connecting if the
  local state says in sync and `deleteExtra` is false.

**Impact:**

Remote files deleted or modified outside Styxpress are not detected. The panel
can show "Synced" even if the remote is different or broken, and a manual sync
can restore nothing because it returns before connecting.

**Evidence:**

`Status` does not call `connect` or `remoteFiles`. `Sync` can exit before
opening an SFTP session.

**Suggested Fix:**

Rename the status as "local-state only" or add an explicit remote verification.
Provide a "force deploy" command that uploads all files or compares the remote
before deciding to return early.

## FND-021 - Frontend install/build are non-deterministic and have no audit gate

**Severity:** Low

**Area:** Release/frontend supply chain

**Status:** Fixed. Integrated admin and release scripts now install from the
lockfile and run a production dependency audit before building the frontend.

**References:**

- `scripts/run_admin.sh:9`: installs dependencies only if `node_modules` is
  missing.
- `scripts/run_admin.sh:11`: uses `npm install` instead of `npm ci`.
- `scripts/build-release.sh:20`: assumes frontend dependencies are already
  present.
- `scripts/build-release.sh:22`: only runs `npm run build`, with no `npm ci` and
  no `npm audit`.
- `README.md:98`: the documented flow uses `npm install`.

**Impact:**

Local builds and release builds can depend on stale `node_modules` or
resolutions that do not exactly match the lockfile. `npm audit --omit=dev` is
currently clean, but there is no gate preventing releases with known
vulnerabilities in frontend dependencies.

**Evidence:**

`package-lock.json` exists, so the project already has deterministic input for
`npm ci`, but scripts do not use it.

**Resolution:**

`scripts/run_admin.sh` and `scripts/build-release.sh` now run `npm ci`, then
`npm audit --omit=dev`, then `npm run build`. This makes the script inputs
deterministic and blocks builds when production dependency advisories are
reported by npm audit.

## FND-022 - Relative paths in `-config` mode depend on the working directory

**Severity:** Medium

**Area:** Config/path resolution

**References:**

- `cmd/styxpress-admin/main.go:24`: `-config` accepts an explicit path.
- `internal/api/server.go:945`: `configuredContentDir` loads the active config.
- `internal/api/server.go:949`: `configuredContentDir` passes `cfg.ContentDir`
  to `configuredPath`.
- `internal/api/server.go:961`: `renderer` passes `cfg.PublicDir` to
  `configuredPath`.
- `internal/api/server.go:1182`: `configuredPath` uses `filepath.Abs(value)`, so
  relative paths are resolved against the current working directory.
- `README.md:169`: the sample config uses `content_dir = "content"` and
  `public_dir = "public"`.

**Impact:**

With `./styxpress-admin -config /path/to/site/config.toml`, a config with
relative paths is not resolved against `/path/to/site/`, but against the
directory from which the process was started. The admin can therefore read/write
content and output in an unexpected directory, show an empty site, or generate
public files in the wrong workspace.

**Evidence:**

The server keeps `configPath`, but `configuredPath` does not receive the config
directory and does not use `filepath.Dir(configPath)`. The documented sample
uses relative paths, so the behavior is plausible for real users.

**Suggested Fix:**

In `-config` mode, resolve `contentDir`, `publicDir`, `sftp_key_path`, and
`sftp_known_hosts_path` relative to the config file directory, or document and
enforce absolute paths. Add tests that start config from a directory different
from the working directory.
