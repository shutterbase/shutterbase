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


FROM oven/bun:1.3.14 AS gallery-css
WORKDIR /usr/src
COPY api/internal/gallery/web/css/package.json api/internal/gallery/web/css/bun.lock ./
RUN bun install --frozen-lockfile
COPY api/internal/gallery/web/css/tailwind.config.js api/internal/gallery/web/css/app.src.css ./
# tailwind scans the templ sources for class names
COPY api/internal/gallery/web/*.templ /usr/src/templates/
COPY api/internal/gallery/web/static/gallery.js /usr/src/static/gallery.js
RUN sed -i 's#\.\./\*\.templ#./templates/*.templ#; s#\.\./static/gallery\.js#./static/gallery.js#' tailwind.config.js \
    && bunx tailwindcss -c tailwind.config.js -i ./app.src.css -o ./app.css --minify


FROM golang:1.26.4-alpine AS builder

WORKDIR /usr/src
COPY api/go.mod /usr/src/go.mod
COPY api/go.sum /usr/src/go.sum
# The AI-server contract is a nested module resolved via `replace ../pkg/aiserver`
# (relative to go.mod at /usr/src => /usr/pkg/aiserver).
COPY pkg /usr/pkg
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

# Public gallery (cmd/gallery): templ components are generated at build time
# and the stylesheet comes from the gallery-css stage below.
COPY --from=gallery-css /usr/src/app.css /usr/src/internal/gallery/web/static/app.css
RUN go run github.com/a-h/templ/cmd/templ@v0.3.1020 generate -path internal/gallery/web \
    && go build -o gallery -ldflags="-s -w" ./cmd/gallery


# Public gallery image: same base as the server (exiftool for EXIF-exported
# downloads, tzdata for the event wall clock), a different entrypoint.
FROM alpine:3.22 AS gallery
WORKDIR /usr/app
RUN apk add --no-cache exiftool tzdata
RUN chown -R 1000:1000 /usr/app
COPY --chown=1000:1000 --from=builder /usr/src/gallery /usr/app/gallery
USER 1000
EXPOSE 8090
ENTRYPOINT ["/usr/app/gallery"]


# Server image (default target).
FROM alpine:3.22
WORKDIR /usr/app

# exiftool: the /download route shells out to it to inject corrected EXIF
# (replaces the old standalone exif-worker service).
# tzdata: TIMEZONE (the event's wall clock, used for computedFileName and the
# $DATE/$WEEKDAY tags) is an IANA name — without the zone database alpine has no
# "Europe/Berlin" and every timestamp silently falls back to UTC.
RUN apk add --no-cache exiftool tzdata

RUN chown -R 1000:1000 /usr/app
COPY --chown=1000:1000 --from=builder /usr/src/server /usr/app/server
COPY --chown=1000:1000 --from=builder /usr/src/import /usr/app/import

USER 1000
EXPOSE 8080
ENTRYPOINT ["/usr/app/server"]

