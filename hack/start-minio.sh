#!/bin/bash

BASE="$(git rev-parse --show-toplevel)"
[[ $? -eq 0 ]] || {
  echo 'Run this script from inside the repository (cannot determine toplevel directory)'
  exit 1
}

S3_ACCESS_KEY="my-access-key"
S3_SECRET_KEY="my-secret-key"
CONTAINER_NAME='shutterbase-s3'
DEFAULT_BUCKETS='shutterbase'

start_minio () {
  docker run --rm -d --name ${CONTAINER_NAME} \
    --publish 9000:9000 \
    --publish 9001:9001 \
    --env MINIO_DEFAULT_BUCKETS=${DEFAULT_BUCKETS} \
    --env MINIO_ROOT_USER="${S3_ACCESS_KEY}" \
    --env MINIO_ROOT_PASSWORD="${S3_SECRET_KEY}" \
    bitnami/minio:latest
}


(docker ps -a --format {{.Names}} | grep ${CONTAINER_NAME} -w) || start_minio