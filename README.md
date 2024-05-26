# shutterbase
![shutterbase](docs/images/shutterbase.png)  
Shutterbase is a web-based application for collaborative photography teams.  
It allows to uploading, time-syncing, tagging and searching photos.

## Features
 => TODO

## Technologies
- Backend: [Pocketbase](https://pocketbase.io/)
- Frontend: vue.js with [Quasar.dev](https://quasar.dev/)
- Database: SQLite
- Local photo processing: WASM written in Rust

## Development
### Prerequisites
To get started with development the following tools are required:
- go (1.21.6 or later)
- bun (v1.1.10 or later)
- rust (2021 edition or later)
- docker (for running a local s3 server)

### API Development
#### Starting minio
A S3 bucket is required for storing photos. To start a local minio server run:
```bash
docker run --rm -p 8091:9000 -p 8092:9001 \
  -e MINIO_ROOT_USER="minio-root-user" \
  -e MINIO_ROOT_PASSWORD="minio-root-password" \
  -e MINIO_DEFAULT_BUCKETS="shutterbase" \
  bitnami/minio:latest
```

#### .env file
A `.env` (placed inside of the `./api` directory) or environment variables can be used to configure the api server.  
The following `.env` file is recommended for local development:
```bash
DEV=true

DOMAIN_NAME=localhost

S3_ENDPOINT=localhost
S3_PORT="8091"
S3_SSL=false
S3_ACCESS_KEY=minio-root-user
S3_SECRET_KEY=minio-root-password
S3_BUCKET=shutterbase
```

#### Server startup
To start the API server run:
```bash
cd api
go run cmd/server/main.go serve
```

This will start the Pocketbase server on `http://localhost:8090` and initialize an empty SQLite database in the `pb_data` directory.  
The admin backend can be accessed at [http://localhost:8090/_/](http://localhost:8090/_/)

### Frontend Development
Since `bun` is being used as package manager, starting the frontend is as simple as running:
```bash
cd ui
bun install
bun run dev
```
This will start the UI server on [http://localhost:9000](http://localhost:9000)