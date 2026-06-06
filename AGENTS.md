# Repository Guidelines

## Project Structure & Module Organization

Styxpress is a Go admin executable with a Vue/Vite admin UI. The Go entry
point is `cmd/styxpress-admin/main.go`. Active private Go packages live in
`internal/`:

- `internal/api`: authenticated local admin API.
- `internal/config`: admin config, multi-site registry, and workspace setup.
- `internal/content`: file-backed post repository and content path rules.
- `internal/deploy`: SFTP status and sync support.
- `internal/rendering`: static HTML, feed, sitemap, assets, and favicon output.
- `internal/siteconfig`: `content/site.toml` parsing and validation.

Reusable public Go packages belong in `pkg/`; it is currently only a
placeholder. The admin frontend lives in `admin/web/`, with source in
`admin/web/src/`, static assets in `admin/web/public/`, and Vite configuration
in `admin/web/vite.config.js`. Design notes, API documentation, and deployment
examples are kept in `docs/`.

The frontend production build is expected at `cmd/styxpress-admin/web/dist` so
it can be embedded by the Go binary. Do not commit generated `dist` output
unless explicitly requested.

## Build, Test, and Development Commands

- `cd admin/web && npm install`: install frontend dependencies from
  `package-lock.json`.
- `cd admin/web && npm run dev`: run the Vite development server for frontend
  work. The committed Vite config has no `/api` proxy, so the embedded Go
  server remains the normal integrated path.
- `cd admin/web && npm run build`: build the admin UI into
  `cmd/styxpress-admin/web/dist`.
- `cd admin/web && npm run preview`: preview the production frontend build with
  Vite.
- `go test ./...`: run all Go tests.
- `go build -o styxpress-admin ./cmd/styxpress-admin`: compile the local admin
  executable.
- `./styxpress-admin` or `./styxpress-admin -addr 127.0.0.1:8080`: run the
  embedded admin server with the multi-site registry.
- `./styxpress-admin -config /path/to/config.toml`: run against one explicit
  admin config file.
- `./scripts/run_admin.sh`: install frontend dependencies if needed, build the
  UI, build the Go binary, and run it.
- `./scripts/build-release.sh linux/amd64` or `./scripts/build-release.sh all`:
  build release binaries under `dist/releases/`.

## Coding Style & Naming Conventions

Use standard Go formatting with `gofmt`; keep packages small and named for
behavior, not layers. Prefer explicit error handling and conservative defaults,
especially around local files, SSH/SFTP deployment, and network binding.

Frontend code uses Vue 3 single-file components, the Composition API, ES
modules, and Pinia. Use four-space indentation in Vue templates, scripts, and
styles; use single quotes in JavaScript imports and PascalCase component names.
Keep CSS scoped when component-specific. The current admin app does not use
`vue-router`; screens are selected through `stores/ui.js`.

## Admin UI Guidelines

Use Pinia stores for data retrieval, state mapping, and cross-component state.
Current stores are flat files under `admin/web/src/stores/`; keep new state in
focused stores that match existing domains such as `config`, `deploy`, `posts`,
`siteConfig`, `siteWorkspace`, `preview`, `auth`, and `ui`.

Prefer self-contained components that interact with the relevant store directly.
Do not bubble domain actions such as site loading, navigation, deployment, post
selection, or logout to a parent only so the parent can call a store. A focused
logout button should call `authStore.logout()` directly.

Every UI element with custom behavior should generally be its own component
with its own template, logic, and scoped styles. Move buttons, profile cards,
action blocks, list items, empty states, loading states, confirmation prompts,
and panels with custom logic into focused components. Define reusable UI
primitives once and reuse them; prefer `components/ui/UiButton.vue` for command
buttons, while semantic navigation/list controls may stay local when their
behavior is specific to that component.

Use parent components primarily for layout and composition. Avoid deep prop
drilling when store state is the clearer boundary, and keep feature components
responsible only for feature-specific behavior. Keep API clients typed or
structurally documented enough that request and response contracts are obvious.
UI behavior must handle loading, empty, unauthorized, and error states.

## Testing Guidelines

Go tests are committed under `cmd/` and `internal/`. Add new Go tests as
`*_test.go` beside the package under test and run them with `go test ./...`.

For frontend behavior, add tests only with an agreed test runner; until then,
verify with `npm run build` and manual checks through the embedded admin server
or `npm run dev` where API access is not needed.

## Commit & Pull Request Guidelines

Recent commits use short, imperative or descriptive lowercase messages such as
`project structure`, `workspace cleaning`, and `simplify sftp deploy`. Keep
future commit subjects concise and focused on one change.

Pull requests should include a clear summary, note affected areas
(`cmd/styxpress-admin`, `internal`, `admin/web`, `docs`), list commands run, and
include screenshots or screen recordings for visible admin UI changes. Link
related issues or design notes when applicable.

## Security & Configuration Tips

The admin server should bind to `127.0.0.1` by default. Do not store SSH key
passphrases, SFTP passwords, or local publishing credentials in the repository
or in admin config files. SFTP passwords and encrypted-key passphrases are
session-only server memory values entered through the deploy panel.

Generated blog output belongs in the configured `publicDir`; source content is
expected under the configured `contentDir` as described in `README.md`. Public
web servers and reverse proxies should serve only generated `publicDir` files
and should not read `contentDir` or admin configuration.
