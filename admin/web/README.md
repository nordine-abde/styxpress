# Styxpress Admin Web

This is the Vue admin UI embedded by `cmd/styxpress-admin`.

## Stack

- Vue 3 single-file components with the Composition API.
- Pinia stores for API-backed state and cross-component state.
- Vite for development and production builds.
- Toast UI Editor for the visual Markdown editor.
- No committed `vue-router` dependency; screen selection is controlled through
  `src/stores/ui.js`.

Use `package.json` as the source of truth for exact dependency and Node engine
versions. The current Node requirement is `^20.19.0 || >=22.12.0`.

## Project Layout

```text
admin/web/
  src/
    api/              fetch helpers and API error wrapper
    assets/           admin UI assets
    components/       feature components and UI primitives
    stores/           Pinia stores
    App.vue
    main.js
  public/             static files copied by Vite
  vite.config.js
```

The production build output is configured in `vite.config.js`:

```text
cmd/styxpress-admin/web/dist
```

That folder is embedded by the Go binary and should not be committed unless
explicitly requested.

## Commands

```sh
npm install
npm run dev
npm run build
npm run preview
```

`npm run dev` starts Vite for frontend work. The committed Vite config does not
proxy `/api`, and API requests are relative paths, so full API-backed testing is
normally done through the embedded Go server after `npm run build`.

From the repository root, this integrated path builds and runs the admin:

```sh
./scripts/run_admin.sh
```

## Session Token

The Go server injects `window.__STYXPRESS_SESSION__` into the embedded
`index.html`. The auth store uses that token automatically. If the UI is served
without the Go server, the app can still accept a pasted token, but API calls
must reach the same origin because the fetch client calls relative `/api/*`
paths.
