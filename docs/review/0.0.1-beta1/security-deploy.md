# Security And Deploy Review

Incremental document for local authentication, admin API, file handling, SFTP,
scripts, and operational documentation.

## Analysis Notes

- FND-002 confirmed: `govulncheck` finds 9 reachable vulnerabilities in
  `golang.org/x/crypto@v0.31.0` on the SSH/SFTP surface.
- FND-005 confirmed: the admin token is injected into the SPA served without
  authentication; this model is acceptable only if the admin stays strictly
  local.
- FND-006 confirmed: the cover preview follows symlinks.
- FND-018 confirmed: the local Go 1.22.2 toolchain exposes reachable standard
  library vulnerabilities in the release binary.
- FND-019 confirmed: the SFTP secret is global to the server and is not bound to
  a site/configuration.
- FND-020 confirmed: status/sync can rely only on the local state file and not
  on the remote server.
- FND-021 confirmed: frontend/release scripts do not use a deterministic
  `npm ci` flow and do not have an audit gate.

## Automated Checks

- `npm audit --omit=dev`: 0 vulnerabilities.
- `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`: fails with
  reachable vulnerabilities in `golang.org/x/crypto`.
- `go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...` with local toolchain
  `go1.22.2`: fails with 38 reachable vulnerabilities across the standard
  library and `golang.org/x/crypto`.
