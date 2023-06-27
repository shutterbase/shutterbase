#!/bin/bash

BASE="$(git rev-parse --show-toplevel)"
[[ $? -eq 0 ]] || {
  echo 'Run this script from inside the repository (cannot determine toplevel directory)'
  exit 1
}

DB_USER=shutterbase
DB_PASS=shutterbase
CONTAINER_NAME='shutterbase-db'

start_postgres () {
    docker run --rm -d \
        -p 5432:5432 \
        --name ${CONTAINER_NAME} \
        -e POSTGRES_USER=$DB_USER \
        -e POSTGRES_PASSWORD=$DB_PASS \
        -e POSTGRES_DB=shutterbase \
        -v ${BASE}/hack/postgres-init-script.sql:/docker-entrypoint-initdb.d/postgres-init-script.sql \
        postgres:15
}

(docker ps -a --format {{.Names}} | grep ${CONTAINER_NAME} -w) || start_postgres
