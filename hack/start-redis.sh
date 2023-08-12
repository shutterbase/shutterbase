#!/bin/bash

BASE="$(git rev-parse --show-toplevel)"
[[ $? -eq 0 ]] || {
  echo 'Run this script from inside the repository (cannot determine toplevel directory)'
  exit 1
}

CONTAINER_NAME='shutterbase-redis'

start_redis () {
    docker run --rm -d \
        -p 6379:6379 \
        --name ${CONTAINER_NAME} \
        -v ${BASE}/hack/redis.conf:/usr/local/etc/redis/redis.conf \
        redis
}

(docker ps -a --format {{.Names}} | grep ${CONTAINER_NAME} -w) || start_redis
