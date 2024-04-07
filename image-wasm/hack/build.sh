#!/bin/bash

BASE="$(git rev-parse --show-toplevel)"
[[ $? -eq 0 ]] || {
    echo 'Run this script from inside the repository (cannot determine toplevel directory)'
    exit 1
}

IMAGE_WASM_PATH="${BASE}/image-wasm"
UI_PATH="${BASE}/ui"


cd $IMAGE_WASM_PATH
wasm-pack build --target web
if [ $? -ne 0 ]; then
    echo "Failed to build image-wasm"
    exit 1
fi

cd $BASE

cp -r $IMAGE_WASM_PATH/pkg $UI_PATH/public/image-wasm