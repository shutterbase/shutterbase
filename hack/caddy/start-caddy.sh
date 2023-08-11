#!/bin/bash

docker run --rm -it -p 8090:80 -v $PWD/Caddyfile:/etc/caddy/Caddyfile caddy