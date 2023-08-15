FROM golang:1.20-alpine3.16 as golang-build

WORKDIR /usr/src
COPY ./go.mod /usr/src/go.mod
COPY ./go.sum /usr/src/go.sum
RUN go mod download

COPY ./*.go /usr/src/

COPY ./cmd /usr/src/cmd
COPY ./ent /usr/src/ent
COPY ./internal /usr/src/internal

WORKDIR /usr/src/cmd/shutterbase

RUN go build -o /usr/src/application

FROM alpine:3.16 as application
RUN apk add --no-cache exiftool
USER 1000
WORKDIR /usr/app
COPY --from=golang-build --chown=1000:1000 /usr/src/application /usr/app/application
COPY ./mail-templates /usr/app/mail-templates
ENTRYPOINT [ "/usr/app/application" ]