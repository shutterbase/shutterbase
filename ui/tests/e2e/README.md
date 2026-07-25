# E2E tests (Playwright)

End-to-end UI tests covering every route and the 5 seeded personas
(`admin`, `user`, `projectAdmin`, `projectEditor`, `projectViewer`).

## Prerequisite: the dev stack must be running

These tests drive the real app against a live backend in **DEV mode**:

```bash
# 1. postgres (see README.md for the docker run)            -> sb-pg
# 2. backend in DEV mode                                     -> :8080
cd api && DEV=true go run ./cmd/server serve
# 3. quasar dev server (proxies /api -> :8080)               -> :9000
cd ui && bun run dev
```

`global-setup.ts` reseeds the DB to a known fixture before the suite and fails
fast with instructions if the stack isn't reachable.

## Run

```bash
cd ui
bun run test:e2e          # headless
bun run test:e2e:ui       # Playwright UI mode
bunx playwright test personas.spec.ts   # a single spec
bunx playwright show-report              # last HTML report
```

## Layout

- `helpers.ts` — `loginAs(role)` (dev login + seed-project activation), id resolvers, JS-error collector.
- `global-setup.ts` — one-time reseed.
- `auth.spec.ts` — guard redirect, login, logout.
- `personas.spec.ts` — per-persona nav + action-gating matrix.
- `smoke.spec.ts` — every route renders without JS errors (admin).
- `gallery.spec.ts` — density / search / tag filter / sort / orientation.
- `project-tags.spec.ts` — tag create + delete through the dialog.
- `upload.spec.ts` — the browser ingest pipeline end to end: real JPEG → WASM resize/EXIF
  → presigned S3 PUT → image record. **Extra prerequisites:** the WASM module must be built
  (`./image-wasm/hack/build.sh`) and S3/RustFS must be reachable, because this spec moves
  real bytes. Fixture in `fixtures/` — its name carries a 4-digit frame number and its EXIF
  carries `DateTimeOriginal`, both of which the pipeline requires.

Note on `smoke.spec.ts`: route-render coverage is not pipeline coverage. It visits
`/uploads/:id/edit`, but the page's WASM offset mapping is a lazy `computed` with no
template consumer, so rendering never evaluates it. `upload.spec.ts` is what covers it.

The suite runs serially (`workers: 1`) because all tests share one backend DB.
