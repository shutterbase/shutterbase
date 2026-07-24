# shutterbase
![shutterbase](docs/images/shutterbase.png)  
Shutterbase is a web-based application for collaborative photography teams.  
It allows to uploading, time-syncing, tagging and searching photos.

## Features
 => TODO

## Technologies
- Backend: Go ([Gin](https://gin-gonic.com/) + [ent](https://entgo.io/) ORM), single binary that embeds the built UI
- Frontend: vue.js with [Quasar.dev](https://quasar.dev/)
- Database: PostgreSQL (SQLite is used as a fallback for unit tests)
- Object storage: S3 (RustFS locally)
- Local photo processing: WASM written in Rust

## Development
### Prerequisites
To get started with development the following tools are required:
- go (1.26 or later — see `api/go.mod`)
- bun (v1.1.10 or later)
- rust (2021 edition or later) + [wasm-pack](https://rustwasm.github.io/wasm-pack/) (for the WASM module)
- docker (for running a local Postgres and S3 server)

### API Development
#### Starting local dependencies (Postgres + RustFS)
The backend stores its data in PostgreSQL and its photos in an S3-compatible bucket (RustFS locally). `docker-compose.yml` at the repo root brings both up, matching the default config exactly (`postgres`/`postgres`, database `postgres`; S3 key/secret `shutterbaseadmin`, bucket `shutterbase` auto-created on start):
```bash
just up           # starts Postgres + RustFS (docker compose up -d)
just deps-logs    # tails their logs
just down         # stops them
```
The schema is created/updated automatically on server startup (ent auto-migrate) — no manual migration step.

#### .env file
A `.env` (placed inside of the `./api` directory) or environment variables can be used to configure the api server.  
The following `.env` file is recommended for local development:
```bash
DEV=true

DOMAIN_NAME=localhost

# Required: signs the session cookies (any non-empty value in dev).
SESSION_SECRET_KEY=dev-secret-change-me

# Optional: a known dev admin. If unset, a random one-time password is generated
# and printed to the server log on first start.
DEFAULT_ADMIN_USERNAME=admin
DEFAULT_ADMIN_PASSWORD=admin

# Postgres — the defaults already match `just up`, listed here for clarity.
DATABASE_TYPE=psql
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=postgres
DATABASE_USERNAME=postgres
DATABASE_PASSWORD=postgres

# S3 (RustFS) — the defaults already match `just up`, listed here for clarity.
# Host 9000 is reserved for the Quasar UI dev server, so RustFS's S3 API is on 9010.
S3_ENDPOINT=localhost
S3_PORT=9010
S3_SSL=false
S3_ACCESS_KEY=shutterbaseadmin
S3_SECRET_KEY=shutterbaseadmin
S3_BUCKET=shutterbase
```

#### Server startup
To start the API server run:
```bash
cd api
go run cmd/server/main.go
```

This starts the server on `http://localhost:8080`. On first start it connects to Postgres, runs the schema migration, and bootstraps the admin user from `DEFAULT_ADMIN_*` (logging a generated password if none was set). In DEV mode the server proxies unknown routes to the Quasar dev server (`UI_PROXY_URL`, `:9000`); in production it serves the SPA embedded in the binary. `:8080` is the single front door in every environment — `bun run dev` opens it automatically.

#### Testing
Test recipes live in `api/justfile`:
```bash
cd api
just test-unit   # go test ./...        — no containers (ent enttest + SQLite)
just test-e2e    # go test -tags e2e    — needs Docker (testcontainers Postgres + rustfs)
just test        # unit + e2e
just seed        # seed Postgres with time-relative fixtures
```

### Frontend Development
Since `bun` is being used as package manager, starting the frontend is as simple as running:
```bash
cd ui
bun install
bun run dev
```
This starts the Quasar dev server on [http://localhost:9000](http://localhost:9000) and opens [http://localhost:8080](http://localhost:8080) in your browser — the API server, which proxies everything back to :9000 in DEV mode (see above).

### WASM Development
The Rust image-processing module is rebuilt and copied into `ui/public/` with:
```bash
./image-wasm/hack/build.sh   # or: cd image-wasm && wasm-pack build --target web
```
