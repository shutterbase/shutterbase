#!/usr/bin/env bash
# Seed the FSG car tags (hack/fsg-tags.tsv) into a running shutterbase instance
# via the REST API. TSV columns: name ⇥ displayName ⇥ description.
# Convergent: existing tags (409) get their displayName updated via PUT, so
# re-runs after editing the TSV bring everything in line.
#
#   SB_TOKEN=<keyId.secret> hack/seed-fsg-tags.sh [projectId]
#
# SB_TOKEN is an API key minted in the UI (user menu -> API keys); it must
# belong to a projectAdmin/admin of the target project. Without projectId the
# key user's activeProject is used. Override the target with SB_URL.
set -euo pipefail

BASE="${SB_URL:-http://localhost:8080/api/v1}"
ORIGIN="${BASE%/api/v1}"
AUTH="Authorization: ApiKey ${SB_TOKEN:?'Set SB_TOKEN=<keyId.secret>'}"
TSV="$(cd "$(dirname "$0")" && pwd)/fsg-tags.tsv"

if [ $# -ge 1 ]; then
  PROJECT="$1"
else
  PROJECT=$(curl -sf -H "$AUTH" "$BASE/users/me" | python3 -c 'import json,sys; print(json.load(sys.stdin)["activeProject"]["id"])')
fi
echo "seeding tags into project $PROJECT at $BASE"

# request METHOD PATH [json-body] -> sets REQUEST_BODY / REQUEST_CODE; retries
# transient failures (429 rate limiter / 5xx) with quadratic backoff
REQUEST_BODY=""
REQUEST_CODE=""
request() {
  local method="$1" path="$2" body="${3:-}" attempt=0 response code
  while :; do
    if [ -n "$body" ]; then
      response=$(curl -s -w $'\n%{http_code}' -X "$method" \
        -H "$AUTH" -H "Origin: $ORIGIN" -H 'Content-Type: application/json' \
        -d "$body" "$BASE$path")
    else
      response=$(curl -s -w $'\n%{http_code}' -X "$method" -H "$AUTH" "$BASE$path")
    fi
    code=$(printf '%s' "$response" | tail -n1)
    if { [ "$code" = 200 ] || [ "$code" = 201 ]; } || { [ "$code" != 429 ] && [ "${code:0:1}" != 5 ]; } || [ "$attempt" -ge 5 ]; then
      REQUEST_CODE="$code"
      REQUEST_BODY=$(printf '%s' "$response" | head -n -1)
      return 0
    fi
    attempt=$((attempt + 1))
    sleep $((attempt * attempt))
  done
}

created=0 duplicates=0 updated=0 failed=0
while IFS=$'\t' read -r name display desc; do
  [ -z "$name" ] && continue
  payload=$(jq -rn --arg n "$name" --arg dn "$display" --arg d "$desc" --arg p "$PROJECT" \
    '{name:$n, displayName:$dn, description:$d, isAlbum:false, type:"manual", projectId:$p}')
  code=""
  body=""
  request POST /image-tags "$payload"
  code=$REQUEST_CODE
  body=$REQUEST_BODY
  case "$code" in
    201) created=$((created + 1)) ;;
    409)
      request GET "/image-tags?projectId=$PROJECT&limit=500&search=$(jq -rn --arg n "$name" '$n | @uri')"
      tag_id=$(printf '%s' "$REQUEST_BODY" | jq -r --arg n "$name" 'first(.items[] | select(.name == $n) | .id) // empty')
      if [ -n "$tag_id" ]; then
        update_payload=$(jq -rn --arg dn "$display" '{displayName:$dn}')
        request PUT "/image-tags/$tag_id" "$update_payload"
        if [ "$REQUEST_CODE" = "200" ]; then updated=$((updated + 1)); else
          failed=$((failed + 1)); printf 'FAIL [PUT %s] %s\n' "$REQUEST_CODE" "$name" >&2
        fi
      else
        duplicates=$((duplicates + 1))
      fi
      ;;
    *)
      failed=$((failed + 1))
      printf 'FAIL [%s] %s: %s\n' "$code" "$name" "$body" >&2
      ;;
  esac
done < "$TSV"

echo "done: created=$created updated=$updated duplicates=$duplicates failed=$failed"
[ "$failed" -eq 0 ]
