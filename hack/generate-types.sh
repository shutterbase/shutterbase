#!/bin/bash

if ! command -v yq &> /dev/null
then
    echo "yq could not be found"
    exit 1
fi

if ! command -v sops &> /dev/null
then
    echo "sops could not be found"
    exit 1
fi

if ! command -v bunx &> /dev/null
then
    echo "bunx could not be found"
    exit 1
fi

BASE="$(git rev-parse --show-toplevel)"
[[ $? -eq 0 ]] || {
    echo 'Run this script from inside the repository (cannot determine toplevel directory)'
    exit 1
}

BASE_URL="http://127.0.0.1:8090"

USERNAME=$(sops --decrypt ${BASE}/credentials.secret.enc.yml | yq ".pocketbase.admin.username")
PASSWORD=$(sops --decrypt ${BASE}/credentials.secret.enc.yml | yq ".pocketbase.admin.password")

bunx pocketbase-typegen --url $BASE_URL --email $USERNAME --password $PASSWORD --out ${BASE}/ui/src/types/pocketbase.ts