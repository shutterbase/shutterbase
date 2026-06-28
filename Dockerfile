FROM rust:1.88.0 as wasm-builder
WORKDIR /usr/src
RUN rustup target add wasm32-unknown-unknown
RUN cargo install wasm-pack
RUN cargo install wasm-opt --locked


FROM ghcr.io/shutterbase/wasm-builder:latest as image-wasm-build

WORKDIR /usr/src/image-wasm

COPY image-wasm/Cargo.toml /usr/src/image-wasm/Cargo.toml
COPY image-wasm/src /usr/src/image-wasm/src

RUN wasm-pack build --target web --release


FROM oven/bun:1.3.14 as ui

WORKDIR /usr/src

COPY ui/package.json /usr/src/package.json
COPY ui/bun.lockb /usr/src/bun.lockb
COPY ui/.npmrc /usr/src/.npmrc

COPY --from=image-wasm-build /usr/src/image-wasm/pkg /usr/image-wasm/pkg

RUN bun install --frozen-lockfile

COPY ui/public /usr/src/public
COPY ui/src /usr/src/src
COPY ui/index.html /usr/src/index.html
COPY ui/postcss.config.cjs /usr/src/postcss.config.cjs
COPY ui/quasar.config.ts /usr/src/quasar.config.ts
COPY ui/tailwind.config.js /usr/src/tailwind.config.js
COPY ui/tsconfig.json /usr/src/tsconfig.json

RUN bun run build


FROM golang:1.26.4-alpine AS builder

WORKDIR /usr/src
COPY api/go.mod /usr/src/go.mod
COPY api/go.sum /usr/src/go.sum
RUN go mod download

COPY api/cmd /usr/src/cmd
COPY api/internal /usr/src/internal
COPY api/ent /usr/src/ent

# Embed the real SPA into the server binary. internal/server/spa embeds dist/ via
# go:embed; here we replace the committed dev placeholder with the actual Quasar
# build before `go build`, so the production binary ships the app as one file.
COPY --from=ui /usr/src/dist/spa /usr/src/internal/server/spa/dist

# Pure-Go deps (pgx, modernc sqlite, stdlib image) => static build, no libc.
ENV CGO_ENABLED=0
RUN go build -o server -ldflags="-s -w" ./cmd/server
RUN go build -o import -ldflags="-s -w" ./cmd/import


FROM alpine:3.22
WORKDIR /usr/app

# exiftool: the /download route shells out to it to inject corrected EXIF
# (replaces the old standalone exif-worker service).
RUN apk add --no-cache exiftool

RUN chown -R 1000:1000 /usr/app
COPY --chown=1000:1000 --from=builder /usr/src/server /usr/app/server
COPY --chown=1000:1000 --from=builder /usr/src/import /usr/app/import

USER 1000
EXPOSE 8080
ENTRYPOINT ["/usr/app/server"]
