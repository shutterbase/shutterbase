# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Shutterbase is a web app for collaborative photography teams: uploading, time-syncing, tagging, and searching photos. Three codebases in one repo:

- `api/` — Go backend built on **PocketBase** (SQLite + admin UI + auth baked in)
- `ui/` — Vue 3 + Quasar 2 SPA, package manager is **bun** (not npm)
- `image-wasm/` — Rust → WASM module the browser uses for client-side image processing (thumbnailing, QR code reading for time-sync)

## Commands

```bash
# UI (from ui/)
bun install
bun run dev          # quasar dev on :9000
bun run build        # quasar build → dist/spa
bun run format       # prettier

# API (from api/)
go run cmd/server/main.go serve   # PocketBase on :8090, admin at /_/
go test ./...

# WASM (from repo root) — rebuilds Rust and copies pkg/ into ui/public/
./image-wasm/hack/build.sh        # needs wasm-pack; or: cd image-wasm && wasm-pack build --target web
```

Local dev needs an S3 server (minio) and an `api/.env` — see README.md for the exact docker run and env vars. A single `go test ./...` runs all Go tests; target one with `go test ./internal/exif/ -run TestName`.

## Architecture

**PocketBase is the spine.** The server (`api/cmd/server/main.go`) creates a PocketBase app, registers hooks, registers custom routes, then serves the built UI as static files (SPA fallback to `index.html`). Collections, auth, and the REST/realtime API come from PocketBase; the custom Go code layers business logic on top via two mechanisms:

1. **Hooks** (`api/internal/hooks/`) — record lifecycle callbacks. `RegisterHooks` wires up two groups: `authorization_*.go` files enforce per-collection access rules, and the rest implement business logic (image processing, default tags, AI detection). The `HookExecutor` holds expirable LRU caches (roles, project assignments, tags, images) — when changing authorization or tag logic, mind cache TTLs in `hooks.go`.

2. **Custom routes** (`api/internal/server/`) — endpoints PocketBase doesn't give for free: S3 presigned upload URLs, websocket server, tag sync, statistics.

**Migrations** (`api/migrations/`) are Go files, timestamp-prefixed, applied automatically. In DEV mode `Automigrate: true` is on, so editing collections in the admin UI generates new migration files. Treat migrations as append-only history.

**Image upload flow:** browser uses `image-wasm` to process the photo → requests an S3 presigned URL from the custom route → uploads directly to S3 → PocketBase record creation triggers image hooks (thumbnail generation at `THUMBNAIL_SIZES`, default tagging, queuing for AI detection).

**AI detection** (`hooks/image_ai_util.go`) runs as a background goroutine processing a queue, calling OpenAI (needs `OPENAI_API_KEY`), with a backoff timer on rate limits. It's fire-and-forget off the hot path.

**Time-sync** is the domain's signature feature: photographers' cameras have clock offsets. The `timeoffset/` package and QR-code reading in WASM reconcile camera timestamps to a shared reference so photos across photographers line up chronologically.

### Auxiliary binaries (separate from the main server)

- `api/cmd/exifworker/` — standalone **Gin** HTTP service for EXIF extraction; talks back to PocketBase via `INTERNAL_POCKETBASE_URL`. Its own config (`InitExifWorkerConfig`), its own Dockerfile.
- `api/cmd/downloader/` — CLI to bulk-download a project's photos, filtered by an AND-list of tags (`--whitelist`). See its README for flags.

### Docker

The root `Dockerfile` is a 4-stage build: wasm-builder (Rust) → ui (bun build) → Go builder (runs `go test ./...` then builds) → alpine runtime serving the server binary with the built UI in `./web`. The server listens on `:8080` in the container.

## Conventions

- **Conventional Commits** (`feat:`, `fix:`, `docs:`, `refactor:`, `chore:`...) — release automation depends on this.
- **Branch naming**: `feature/`, `fix/`, `docs/`, `chore/`, `refactor/` prefixes.
- `main` is protected and enforces **linear history** — rebase, don't merge. PRs only.
- Config is centralized via `mxcd/go-config` in `util.InitConfig` / `InitExifWorkerConfig`; add new settings there, mark secrets `.Sensitive()`.
- Secrets in `credentials.secret.enc.yml` are SOPS-encrypted (`.sops.yaml`).
