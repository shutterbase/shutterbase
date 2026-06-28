# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Shutterbase is a web app for collaborative photography teams: uploading, time-syncing, tagging, and searching photos. Three codebases in one repo:

- `api/` — Go backend: **Gin** HTTP + **ent** ORM (Postgres, SQLite fallback for tests) + **go-basicauth** session/API-key auth. Builds to a single binary that embeds the built Vue SPA (`go:embed`).
- `ui/` — Vue 3 + Quasar 2 (Vite) + Tailwind SPA, package manager is **bun** (not npm)
- `image-wasm/` — Rust → WASM module the browser uses for client-side image processing (thumbnailing, QR code reading for time-sync)

## Commands

```bash
# UI (from ui/)
bun install
bun run dev          # quasar dev on :9000
bun run build        # quasar build → dist/spa
bun run test         # vitest unit tests
bun run test:e2e     # playwright e2e (needs the testserver running — see ui/tests/e2e/README.md)
bun run format       # prettier

# API (from api/ — recipes in justfile)
go run cmd/server/main.go            # server on :8080 (no subcommand; proxies the UI dev server in DEV)
just test-unit                       # go test ./...        (no containers; ent enttest + SQLite)
just test-e2e                        # go test -tags e2e ./test/e2e/...  (needs Docker: testcontainers psql + rustfs)
just test                            # unit + e2e
just seed                            # seed Postgres with time-relative fixtures

# WASM (from repo root) — rebuilds Rust and copies pkg/ into ui/public/
./image-wasm/hack/build.sh           # needs wasm-pack; or: cd image-wasm && wasm-pack build --target web
```

Local dev needs Postgres (`DATABASE_TYPE=psql`, default `:5432`), an S3 server (minio), and an `api/.env` — see README.md for the minio docker run and env vars. Target a single Go test with `go test ./internal/exif/ -run TestName`; e2e tests are gated behind the `e2e` build tag.

## Architecture

**Gin is the spine.** `server.NewServer` (`api/internal/server/server.go`) builds the Gin engine, wires middleware (auth, CORS/origin hardening, impersonation, rate limiting), registers the REST controllers under `/api`, mounts the websocket server, and serves the embedded SPA via a `NoRoute` handler with `index.html` fallback for client-side routing. The business logic is layered on top in three tiers:

1. **Controllers** (`api/internal/server/*_controller.go`) — HTTP handlers for each resource (projects, images, tags, time offsets, uploads, statistics…). They authorize, validate, and delegate.

2. **Services** (`api/internal/service/`) — cross-cutting business logic: AI detection queue, image/thumbnail processing.

3. **Repository** (`api/internal/repository/`) — data access wrapping the generated **ent** client (`api/ent/`).

**Authorization** (`api/internal/authorization/`) is a set of composable checkers — `isAdmin`, `HasRoleInProject`, `CanEditProject`, `CanManageProject`, `CanManageImageTagAssignment`, etc. (`CanManageProject` = global-admin-only create/delete; `CanEditProject` = admin or projectAdmin-of-this-project field edits). **Authentication** (`api/internal/authentication/`) uses go-basicauth cookie sessions plus a custom API-key bridge; both funnel through per-request user resolution that loads the `ent.User` into the request context — read it with `util.GetUser(ctx)`. A global middleware rejects deactivated users.

**Database / migrations:** schema is defined in `api/ent/schema/` and the ent client is generated into `api/ent/`. There are **no hand-written migration files** — `Client.Schema.Create` (ent auto-migrate) runs on startup and via `cmd/migrate` (`migrate create` is idempotent; `migrate drop` is DEV-only). Edit the ent schema, regenerate, and auto-migrate applies it.

**Image upload flow:** browser uses `image-wasm` to process the photo → requests an S3 presigned URL from the upload controller → uploads directly to S3 → record creation triggers thumbnail generation (at `THUMBNAIL_SIZES`), default tagging, and queuing for AI detection.

**AI detection** (`service/ai_service.go`) runs as a background goroutine processing a queue, calling OpenAI (needs `OPENAI_API_KEY`), with a backoff timer on rate limits. It's fire-and-forget off the hot path.

**Time-sync** is the domain's signature feature: photographers' cameras have clock offsets. Time offsets (`time_offsets_controller.go`) plus QR-code reading in WASM reconcile camera timestamps to a shared reference so photos across photographers line up chronologically.

**EXIF:** `internal/exif/inject.go` shells out to `exiftool` to inject corrected timestamps and copyright metadata on download (this replaced the old standalone exif-worker service).

### Auxiliary binaries (separate from the main server)

- `api/cmd/downloader/` — CLI to bulk-download a project's photos, filtered by an AND-list of tags (`--whitelist`). See its README for flags.
- `api/cmd/import/` — one-shot migrator: reads a legacy PocketBase SQLite DB and imports it into Postgres (drops + recreates the schema, FK-safe order; S3 untouched).
- `api/cmd/seed/` — seed fixtures. `api/cmd/testserver/` — backing server for the Playwright UI e2e suite. `api/cmd/migrate/` — thin CLI over ent auto-migrate.

### Docker

The root `Dockerfile` is a multi-stage build: wasm-builder (Rust) → ui (`bun build`) → Go builder (replaces the dev SPA placeholder with the real Quasar build, then `go build` the `server` and `import` binaries, `CGO_ENABLED=0`) → alpine runtime (adds `exiftool`, runs the static `server` binary with the SPA embedded). The server listens on `:8080` in the container.

## Conventions

- **Conventional Commits** (`feat:`, `fix:`, `docs:`, `refactor:`, `chore:`...) — release automation depends on this.
- **Branch naming**: `feature/`, `fix/`, `docs/`, `chore/`, `refactor/` prefixes.
- `main` is protected and enforces **linear history** — rebase, don't merge. PRs only.
- Config is centralized via `mxcd/go-config` in `util.InitConfig`; add new settings there, mark secrets `.Sensitive()`.
- Secrets in `credentials.secret.enc.yml` are SOPS-encrypted (`.sops.yaml`).
