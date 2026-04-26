# Repository Guidelines

## Project Structure & Module Organization

Styxpress is an early-stage Go admin executable with a Vue/Vite admin UI. The Go entry point is `cmd/styxpress-admin/main.go`; future private Go packages belong in `internal/`, and reusable public packages belong in `pkg/`. The admin frontend lives in `admin/web/`, with source in `admin/web/src/`, static assets in `admin/web/public/`, and Vite configuration in `admin/web/vite.config.js`. Design notes and architecture decisions are kept in `docs/`, especially `docs/initial-design.md`.

The frontend production build is expected at `cmd/styxpress-admin/web/dist` so it can be embedded by the Go binary, but do not commit generated `dist` output unless explicitly requested.

## Build, Test, and Development Commands

- `cd admin/web && npm install`: install frontend dependencies from `package-lock.json`.
- `cd admin/web && npm run dev`: run the Vite development server for the admin UI.
- `cd admin/web && npm run build`: build the admin UI for embedding.
- `go build -o styxpress-admin ./cmd/styxpress-admin`: compile the local admin executable.
- `./styxpress-admin` or `./styxpress-admin -addr 127.0.0.1:8080`: run the embedded admin server.
- `go test ./...`: run all Go tests once tests are added.

## Coding Style & Naming Conventions

Use standard Go formatting with `gofmt`; keep packages small and named for behavior, not layers. Prefer explicit error handling and conservative defaults, especially around local files and network binding.

Frontend code uses Vue 3 single-file components, the Composition API, ES modules, and Pinia. Match the existing style in `admin/web/src/App.vue`: two-space indentation in templates/styles, single quotes in JavaScript imports, and PascalCase component names. Keep CSS scoped when component-specific.

## Admin UI Guidelines

Use Pinia stores for data retrieval, state mapping, and cross-component state. Store names should follow Styxpress domains, for example future publishing configuration belongs in a publishing/config store, and content editing state belongs in a content/posts store.

Prefer self-contained components that interact with the relevant store directly. Do not bubble domain actions such as profile loading, navigation, publishing, post selection, or logout to a parent only so the parent can call a store. A focused logout button should call `authStore.logout()` directly.

Every UI element with custom behavior should generally be its own component with its own template, logic, and scoped styles. Move buttons, profile cards, action blocks, list items, empty states, loading states, confirmation prompts, and panels with custom logic into focused components. Define reusable UI primitives once and reuse them; avoid raw styled `<button>` elements inside feature components when a shared button component fits.

Use parent components primarily for layout and composition. Avoid deep prop drilling when store state is the clearer boundary, and keep feature components responsible only for feature-specific behavior. Keep API clients typed or structurally documented enough that request and response contracts are obvious. UI behavior must handle loading, empty, unauthorized, and error states.

## Testing Guidelines

There are no committed tests yet. Add Go tests as `*_test.go` beside the package under test and run them with `go test ./...`. For frontend behavior, add tests only with an agreed test runner; until then, verify with `npm run build` and manual checks through `npm run dev`.

## Commit & Pull Request Guidelines

Recent commits use short, imperative or descriptive lowercase messages such as `project structure` and `workspace cleaning`. Keep future commit subjects concise and focused on one change.

Pull requests should include a clear summary, note affected areas (`cmd/styxpress-admin`, `admin/web`, `docs`), list commands run, and include screenshots or screen recordings for visible admin UI changes. Link related issues or design notes when applicable.

## Security & Configuration Tips

The admin server should bind to `127.0.0.1` by default. Do not store SSH key passphrases or local publishing credentials in the repository. Generated blog output belongs in `public/`; source content is expected under `content/` as described in `README.md`.
