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

The suite runs serially (`workers: 1`) because all tests share one backend DB.
