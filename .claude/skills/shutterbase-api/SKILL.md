---
name: shutterbase-api
description: Call the Shutterbase REST API (prod or local) to manage projects, image tags, and schedule items without reading the API source. Use when the user wants to query or mutate Shutterbase data via HTTP/curl — projects, tags, schedule items, assignments, users — or check API connectivity.
---

# Shutterbase API usage

## Base URL & auth

- **Prod:** `https://shutterbase.fsg.one/api/v1`
- **Local dev:** `http://localhost:8080/api/v1`
- **API key:** `SHUTTERBASE_PROD_API_KEY` in `api/.env` (format `keyId.secret`). Send it as:

```sh
KEY=$(grep '^SHUTTERBASE_PROD_API_KEY=' api/.env | cut -d= -f2)
curl -s -H "Authorization: ApiKey $KEY" https://shutterbase.fsg.one/api/v1/users/me
```

The key authenticates as a full user (currently the platform admin), so every endpoint below works with it.

**Gotchas**
- The base path is `/api/v1`, **not** `/api`. Any unknown path (wrong base, typo) falls through to the SPA and returns **200 with HTML** — a 200 that isn't JSON means your URL is wrong, not that it worked.
- **Mutations (POST/PUT/PATCH/DELETE) must send a same-origin `Origin` header** or the CSRF check rejects them with 403 `csrf_origin` — even with a valid API key. Always add `-H "Origin: https://shutterbase.fsg.one"` (match the host you're calling). GETs don't need it.
- Connectivity check: `GET /health` → `{"status":"ok","version":"<image tag>"}` (no auth needed). `GET /users/me` verifies the key.
- Errors come back as `{"error":"<code>","message":"<human text>"}` with 400/401/403/404.

## List envelope & pagination

Every list endpoint returns `{"limit":N,"offset":N,"total":N,"items":[...]}` and accepts:

| Query | Default | Notes |
|---|---|---|
| `limit` | 100 | max 500 |
| `offset` | 0 | |
| `order` | `desc` | `asc` or `desc` |
| `sort` | — | field name, endpoint-specific |
| `search` | — | where supported (projects, image-tags) |

## Projects

| Method & path | Auth | Notes |
|---|---|---|
| `GET /projects` | any (admin sees all, others only assigned) | `?search=` |
| `GET /projects/:id` | admin or member | |
| `POST /projects` | **global admin only** | |
| `PUT /projects/:id` | admin or projectAdmin of the project | partial update — send only changed fields |
| `DELETE /projects/:id` | **global admin only** | 204 on success |

Create payload (all strings **required** unless marked optional):

```json
{
  "name": "FSA26", "description": "...", "copyright": "...",
  "copyrightReference": "...", "locationName": "...", "locationCode": "...",
  "locationCity": "...",
  "aiSystemMessage": null, "uploadReviewEnabled": false,
  "startAt": "2026-08-01T00:00:00Z", "endAt": "2026-08-07T00:00:00Z"
}
```

`aiSystemMessage`, `uploadReviewEnabled`, `startAt`, `endAt` are optional. Update accepts the same fields, all optional. `endAt` before `startAt` → 400 `invalid_period`; a zero timestamp clears a bound. Setting `uploadReviewEnabled: true` auto-creates the reserved custom tag `error` in the project.

## Image tags

| Method & path | Auth | Notes |
|---|---|---|
| `GET /image-tags?projectId=<id>` | project member | `projectId` **required**; `?search=`, `?type=` |
| `GET /image-tags/:id` | member of the tag's project | |
| `POST /image-tags` | by type: `default`/`manual`/`template` → admin/projectAdmin; `custom` → any member | |
| `PUT /image-tags/:id` | by *resulting* type, same rule | partial update |
| `DELETE /image-tags/:id` | admin/projectAdmin | 204 |

Create payload:

```json
{ "name": "podium", "description": "…", "isAlbum": false,
  "type": "manual", "projectId": "<project id>" }
```

- `type` ∈ `template | default | manual | custom`. `default` tags are auto-applied to new uploads; `custom` tags are personal/unofficial and never exported.
- A `template` tag's name **must start with `$`** (`$PROJECT`, `$DATE`, `$WEEKDAY`, `$COPYRIGHT`) or the API rejects it with `invalid_template_name`.
- The tag named `error` is reserved for upload review — creating/renaming it needs admin/projectAdmin.

## Schedule items

| Method & path | Auth | Notes |
|---|---|---|
| `GET /schedule-items?projectId=<id>` | project member | `projectId` **required**; filters: `from`/`to` (RFC3339), `mine=true` |
| `GET /schedule-items/:id` | project member | |
| `POST /schedule-items` | admin/projectAdmin | |
| `PUT /schedule-items/:id` | admin/projectAdmin | partial update |
| `DELETE /schedule-items/:id` | admin/projectAdmin | 204 |
| `PUT /schedule-items/:id/assignees/:userId` | admin/projectAdmin | add assignee |
| `DELETE /schedule-items/:id/assignees/:userId` | admin/projectAdmin | remove assignee |

Create payload:

```json
{ "title": "Autocross heats", "description": "…",
  "start": "2026-08-05T08:00:00Z", "end": "2026-08-05T12:00:00Z",
  "cardinality": 2, "projectId": "<project id>",
  "tagIds": ["<image-tag id>", "…"] }
```

- `title`, `start`, `end`, `projectId` required; `end` must be after `start` (`invalid_window`), `cardinality` ≥ 0 (how many photographers are wanted; 0 = unbounded).
- `tagIds` must belong to the same project (`tag_project_mismatch`); on update, `tagIds` **replaces** the whole set.
- Assignees must be members of the item's project (or global admins), else 400 `not_a_member`. Responses embed `assignees` (id, name, email) and `tags` (id, name, type).

## Supporting lookups

| Method & path | Purpose |
|---|---|
| `GET /users` (admin) / `GET /users/:id` | find `userId`s for assignees |
| `GET /users/me` | who the API key acts as; includes `activeProject` |
| `GET /roles` | find `roleId`s — project roles are `projectAdmin`, `projectEditor`, `projectViewer` |
| `GET /project-assignments?projectId=<id>` | project roster (admin, self-scoped, or projectAdmin of that project) |
| `POST /project-assignments` `{projectId, userId, roleId}` | add a member (admin or projectAdmin); only the three project roles are assignable |
| `PUT /project-assignments/:id` / `DELETE …/:id` | change role / remove member |
| `GET /version` | deployed image tag + signup flag (no auth) |

## End-to-end example: schedule item with tags and an assignee

```sh
BASE=https://shutterbase.fsg.one/api/v1
AUTH="Authorization: ApiKey $KEY"

PROJECT=$(curl -s -H "$AUTH" "$BASE/projects?search=FSA26" | jq -r '.items[0].id')
TAG=$(curl -s -H "$AUTH" "$BASE/image-tags" -d '{"name":"autocross","type":"manual","projectId":"'$PROJECT'"}' -H 'Content-Type: application/json' | jq -r .id)
ITEM=$(curl -s -H "$AUTH" "$BASE/schedule-items" -H 'Content-Type: application/json' -d '{
  "title":"Autocross heats","start":"2026-08-05T08:00:00Z","end":"2026-08-05T12:00:00Z",
  "cardinality":2,"projectId":"'$PROJECT'","tagIds":["'$TAG'"]}' | jq -r .id)
USER=$(curl -s -H "$AUTH" "$BASE/project-assignments?projectId=$PROJECT" | jq -r '.items[0].user.id')
curl -s -X PUT -H "$AUTH" "$BASE/schedule-items/$ITEM/assignees/$USER" | jq '.assignees'
```
