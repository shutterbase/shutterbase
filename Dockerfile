FROM rust:1.77.1 as wasm-builder
WORKDIR /usr/src
RUN rustup target add wasm32-unknown-unknown
RUN cargo install wasm-pack
RUN cargo install wasm-opt --locked


FROM ghcr.io/shutterbase/wasm-builder:latest as image-wasm-build

WORKDIR /usr/src/image-wasm

COPY image-wasm/Cargo.toml /usr/src/image-wasm/Cargo.toml 
COPY image-wasm/src /usr/src/image-wasm/src

RUN wasm-pack build --target web --release

FROM oven/bun:1.0.29 as ui 

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


FROM golang:1.22-alpine3.18 AS builder

WORKDIR /usr/src
COPY api/go.mod /usr/src/go.mod
COPY api/go.sum /usr/src/go.sum
RUN go mod download

COPY api/cmd /usr/src/cmd
COPY api/internal /usr/src/internal

RUN go test ./...
RUN go build -o server -ldflags="-s -w" cmd/server/main.go 

FROM alpine:3.18
WORKDIR /usr/app
RUN chown -R 1000:1000 /usr/app
COPY --chown=1000:1000 --from=builder /usr/src/server /usr/app/server
COPY --chown=1000:1000 --from=ui /usr/src/dist/spa /usr/app/web

USER 1000
EXPOSE 8080
ENTRYPOINT ["/usr/app/server", "serve", "--http=0.0.0.0:8080"]