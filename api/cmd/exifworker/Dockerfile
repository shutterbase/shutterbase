FROM golang:1.22.3-alpine3.18 AS builder

WORKDIR /usr/src
COPY go.mod /usr/src/go.mod
COPY go.sum /usr/src/go.sum
RUN go mod download

COPY cmd /usr/src/cmd
COPY internal /usr/src/internal
COPY migrations /usr/src/migrations

RUN go build -o exifworker -ldflags="-s -w" cmd/exifworker/main.go 

FROM alpine:3.18
RUN apk add --no-cache exiftool
WORKDIR /usr/app
RUN chown -R 1000:1000 /usr/app
COPY --chown=1000:1000 --from=builder /usr/src/exifworker /usr/app/exifworker

USER 1000
EXPOSE 8080
ENTRYPOINT ["/usr/app/exifworker"]