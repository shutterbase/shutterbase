# Schedule API — automation guide

The scheduling module (S15) is fully drivable over the REST API, so the yearly
FSG setup — tag list, event schedule, tag suggestions — can be imported by a
script or by Claude Code from whatever source format exists (CSV, PDF schedule,
last year's export) without a dedicated import feature.

## Authentication

Mint an API key in the UI (user menu → API keys). The token has the form
`<keyId>.<secret>` and is sent as a Bearer token; requests act **as the user**,
with their permissions, and mutations are attributed to them (`createdBy` /
`updatedBy`).

```bash
export SB_URL="https://shutterbase.example.com/api/v1"
export SB_TOKEN="<keyId>.<secret>"
auth=(-H "Authorization: Bearer $SB_TOKEN" -H "Content-Type: application/json")
```

Creating schedule items requires the key's user to be **projectAdmin** of the
target project (or a global admin).

## Endpoints

| Method | Path | Notes |
|---|---|---|
| GET | `/schedule-items?projectId=…` | list; optional `from`/`to` (RFC3339, overlap filter), `mine=true`, `sort=start&order=asc`, `limit`/`offset` |
| GET | `/schedule-items/:id` | single item with `assignees` + `tags` |
| POST | `/schedule-items` | create — see payload below |
| PUT | `/schedule-items/:id` | partial update; `tagIds` replaces the whole suggestion set |
| DELETE | `/schedule-items/:id` | projectAdmin only |
| PUT | `/schedule-items/:id/assignees/:userId` | assign (self: any projectEditor+; others: projectAdmin). No cardinality cap — overbooking is allowed by design |
| DELETE | `/schedule-items/:id/assignees/:userId` | unassign (same rules) |
| PUT | `/projects/:id` | `startAt`/`endAt` frame the calendar; a zero time (`0001-01-01T00:00:00Z`) clears a bound |
| PUT | `/uploads/:id/timeline` | apply timeline tracks — used by the SPA editor, scriptable too |

### Create payload

```json
{
  "projectId": "abc123def456ghi",
  "title": "Endurance",
  "description": "Both drivers, start + finish line coverage",
  "start": "2026-08-14T09:00:00+02:00",
  "end": "2026-08-14T17:00:00+02:00",
  "cardinality": 3,
  "tagIds": ["<tag-id-endurance>", "<tag-id-track>"]
}
```

- `cardinality` is the TARGET headcount (default 1), not a cap.
- `tagIds` must belong to the same project (400 `tag_project_mismatch`
  otherwise). These tags are what the upload timeline suggests to
  photographers covering the item.

## Typical import flow (tags first, then schedule)

1. **Create the season's tag list** — `POST /image-tags` per row
   (`{"name", "description", "type": "default"|"manual", "projectId"}`).
2. **Create the schedule items** — `POST /schedule-items` per event slot,
   referencing the tag ids from step 1 as `tagIds`.
3. **Optionally pre-assign** — `PUT /schedule-items/:id/assignees/:userId`
   ("Teamfotos macht Axel"). User ids come from
   `GET /project-assignments?projectId=…`.

All list endpoints return the standard envelope
`{"limit", "offset", "total", "items"}`.

## Live updates

Every schedule mutation is broadcast to all websocket clients (all replicas,
via Postgres LISTEN/NOTIFY) as
`{"object": "scheduleItem", "action": "changed", "data": {"projectId", "itemId"}}` —
the SPA refetches on receipt; scripts can ignore it.
