# Release Readiness

Final document for the 0.0.1-beta1 review.

## Status

Completed. The beta release should not proceed before the blocking findings
listed below are addressed.

## Checks

- `go test ./...`: passed.
- `go vet ./...`: passed.
- `go test ./internal/rendering -run 'Test|Markdown|Raw|HTML|Link|Image|Script' -v`: passed.
- `npm audit --omit=dev`: passed, 0 vulnerabilities.
- `npm run build`: passed; Vite reports a JS chunk > 500 kB.
- `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`: failed with 9
  reachable vulnerabilities in `golang.org/x/crypto@v0.31.0`.
- `go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...` with `go1.22.2`:
  failed with 38 reachable vulnerabilities across the standard library and
  `golang.org/x/crypto`.
- `go version`: `go1.22.2 linux/amd64`.

## Residual Risks

- Release blocker: FND-001 exposes a source deletion risk if `publicDir` and
  `contentDir` are equal or configured with overlap.
- Release blocker: FND-002 exposes known SSH/SFTP vulnerabilities in the
  `golang.org/x/crypto` dependency.
- Release blocker: FND-003 can publish and deploy drafts when the user uploads
  or removes images.
- Release blocker: FND-005 makes the admin token readable from the SPA if the
  admin is exposed beyond loopback.
- Release blocker: FND-006 can exfiltrate local files through cover symlinks in
  preview.
- Release blocker: FND-018 shows that binaries can be compiled with a vulnerable
  Go toolchain.

## Suggested Priority

1. Block data loss and unintended publishing: FND-001, FND-003.
2. Close immediate security risks: FND-002, FND-005, FND-006, FND-018.
3. Stabilize filesystem/deploy and public output behavior: FND-004, FND-007,
   FND-008, FND-019, FND-020.
4. Fix config, concurrency, and frontend/release hardening: FND-009, FND-010,
   FND-011, FND-012, FND-013, FND-014, FND-015, FND-016, FND-017, FND-021,
   FND-022.
