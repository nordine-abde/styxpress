# Security And Deploy Review

Incremental document for local authentication, admin API, file handling, SFTP,
scripts, and operational documentation.

## Analysis Notes

- FND-002 fixed: `golang.org/x/crypto` is upgraded to `v0.52.0`, closing the
  reachable SSH/SFTP dependency vulnerabilities found in the initial review.
- FND-005 fixed: the admin token is no longer injected into the SPA. The admin
  UI now requires pasting the local token printed by the server.
- FND-006 fixed: cover preview and authenticated cover/asset serving reject
  symlinked files and symlinked content parents.
- FND-018 fixed: release and local admin build scripts now reject unsupported or
  stale Go toolchains before compiling binaries.
- FND-019 confirmed: the SFTP secret is global to the server and is not bound to
  a site/configuration.
- FND-020 confirmed: status/sync can rely only on the local state file and not
  on the remote server.
- FND-021 fixed: frontend/admin release scripts now use `npm ci` and
  `npm audit --omit=dev` before frontend builds.

## Automated Checks

- `npm audit --omit=dev`: 0 vulnerabilities.
- `GOTOOLCHAIN=go1.26.4 go run golang.org/x/vuln/cmd/govulncheck@latest ./...`:
  passed, no vulnerabilities found.
- `go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...` with local toolchain
  `go1.22.2`: fails with 38 reachable vulnerabilities across the standard
  library and `golang.org/x/crypto`.
- FND-018 patch verification: shell syntax checks pass for the release/admin
  scripts and their shared Go toolchain gate; `GOTOOLCHAIN=auto` selects Go
  1.26.4 and passes the gate, while `GOTOOLCHAIN=local` rejects the local stale
  toolchain.
- FND-021 patch verification: `npm ci`, `npm audit --omit=dev`, and
  `npm run build` pass, with the existing Vite chunk-size warning.
