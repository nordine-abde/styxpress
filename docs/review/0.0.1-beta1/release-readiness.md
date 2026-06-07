# Release Readiness

Final document for the 0.0.1-beta1 review.

## Status

Completed and updated after the first security-fix pass. The original release
blockers identified in this review have been addressed; the remaining findings
should be triaged before beta scope is frozen.

## Checks

- Initial review: `go test ./...` passed.
- Initial review: `go vet ./...` passed.
- Initial review: `npm audit --omit=dev` passed, 0 vulnerabilities.
- Initial review: `npm run build` passed; Vite reported a JS chunk > 500 kB.
- Initial review: `govulncheck` failed on `golang.org/x/crypto@v0.31.0` and an
  unsupported Go 1.22.2 toolchain.
- Fix verification: `GOTOOLCHAIN=auto go test ./...` passed.
- Fix verification: `GOTOOLCHAIN=auto go vet ./...` passed.
- Fix verification:
  `GOTOOLCHAIN=go1.26.4 go run golang.org/x/vuln/cmd/govulncheck@latest ./...`
  passed with no vulnerabilities found.
- Fix verification: `npm ci` passed with peer dependency warnings.
- Fix verification: `npm audit --omit=dev` passed, 0 vulnerabilities.
- Fix verification: `npm run build` passed; Vite still reports a JS chunk >
  500 kB.
- Fix verification: shell syntax checks passed for `scripts/run_admin.sh`,
  `scripts/build-release.sh`, and `scripts/go-toolchain.sh`.
- Fix verification: the Go toolchain gate passes with `GOTOOLCHAIN=auto`
  selecting Go 1.26.4 and rejects the stale local toolchain with
  `GOTOOLCHAIN=local`.

## Residual Risks

- Addressed: FND-001 now rejects equal, nested, or symlink-overlapping
  content/public roots.
- Addressed: FND-002 upgrades `golang.org/x/crypto` to `v0.52.0`.
- Addressed: FND-003 no longer publishes drafts from media upload/delete flows.
- Addressed: FND-005 no longer makes the admin token readable from the
  unauthenticated SPA.
- Addressed: FND-006 blocks cover/media symlink reads in preview and API
  serving paths.
- Addressed: FND-007 blocks content and rendering writes/deletes through
  symlinked parent paths.
- Addressed: FND-018 now has a release/admin build gate that rejects unsupported
  or stale Go toolchains before binaries are compiled.
- Addressed: FND-021 adds deterministic frontend install and npm audit gates to
  admin/release scripts.

## Suggested Priority

1. Stabilize public output behavior: FND-004, FND-008, FND-020.
2. Decide deploy secret/session boundaries for beta: FND-019.
3. Fix config, concurrency, auth-state, and frontend robustness findings:
   FND-009, FND-011, FND-012, FND-013, FND-014, FND-015, FND-016, FND-017,
   FND-022.
