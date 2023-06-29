FROM golang:1.20-alpine3.16 as golang-build

WORKDIR /usr/src
COPY ./api/go.mod /usr/src/go.mod
COPY ./api/go.sum /usr/src/go.sum
RUN go mod download

COPY ./api/*.go /usr/src/

COPY ./api/cmd /usr/src/cmd
COPY ./api/ent /usr/src/ent
COPY ./api/internal /usr/src/internal

WORKDIR /usr/src/cmd/shutterbase

RUN go build -o /usr/src/application

# FROM node:18-alpine3.16 as web-build
# WORKDIR /usr/src
# RUN npm i -g @quasar/cli pnpm
# RUN pnpm install quasar
# COPY ./ui/pnpm-lock.yaml /usr/src/pnpm-lock.yaml
# RUN pnpm fetch
# COPY ./ui /usr/src
# RUN pnpm install --force -r --offline
# RUN quasar build -m spa


FROM alpine:3.16 as application
USER 1000
WORKDIR /usr/app
COPY --from=golang-build --chown=1000:1000 /usr/src/application /usr/app/application
# COPY --from=web-build --chown=1000:1000 /usr/src/dist/spa /usr/app/web
COPY ./api/mail-templates /usr/app/mail-templates
ENTRYPOINT [ "/usr/app/application" ]