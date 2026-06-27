> **Orchestrator review — Wave 0 gate (verified against actual PB migrations + api/go.mod):**
> - **Module path corrected** to `github.com/shutterbase/shutterbase` (synth used `.../api`; go.mod declares no `/api` suffix). All import paths adjusted.
> - `users (firstName,lastName)` UNIQUE (`idx_EmJURt8`) and `projects.description` UNIQUE (`idx_Exllboh`) **confirmed real** in PB (not hallucinated). Spec preserves the former, deliberately drops the latter — see "Open flags".
> - Open flags for your sign-off: §0.8 assignment-type default (`manual`), projects.description UNIQUE drop, §0.4 TFA/TOTP storage fork, GIN `entsql.OpClass`.

---

# Shutterbase Rewrite — FROZEN Specification (v1)

Single anchor for all downstream implementer agents. Module path throughout: `github.com/shutterbase/shutterbase` (substitute for `<module>`). Built on the `mxcd/agentic-template` patterns (Go / Ent / gin / gorilla-sessions / go-basicauth). This document is frozen; deviations require an explicit amendment, not a local reinterpretation.

---

## 0. Conflict resolutions (read this first — the four source docs disagreed here)

These are the resolved decisions. Where a source doc said something different, the resolution below wins.

1. **User PK is `uuid.UUID`, full stop.** Doc C hedged ("may become UUID pending spike"); resolved to UUID (Doc A authority, audit-actor contract is UUID). Serialized to JSON as a string. PB id preserved in `User.legacyId`. Every other entity keeps its 15-char PB string PK.
2. **`createdBy`/`updatedBy` are `Optional().Nillable()`** (Doc A `AuditMixin`). This *resolves Doc D's flagged blocking decision* — no sentinel-actor user is needed. Migrated rows with no known author leave them null; live mutations set them from `util.GetActorID(ctx)`. Doc D's "sentinel actor" path is dropped.
3. **`User.role` is a global enum `user|admin`** (Doc A), **not** an edge to the `roles` table (Doc D's "role relation" is superseded). The `roles` table stays, but only for **project-scoped** assignments. REST `/users/me` presents `role` as `{id,key,description}` by mapping the enum to the matching seeded `roles` row (`key=admin`/`key=user`). Importer maps PB `users.role`→`roles.key`: `admin`→enum `admin`, everything else→enum `user`.
4. **User keeps `username`, `email`, `verified`, `passwordHash`, `firstName`, `lastName`, `copyrightTag`, `active`, `forcePasswordChange`, `provider`, `legacyId`.** Doc A proposed dropping `username`/`provider`/`forcePasswordChange`; that conflicts with go-basicauth (needs `username`), the importer (carries them), and the REST contract (`/users/me`, login flow). They stay. **TOTP/OIDC/settings/backupCodes are dropped** and **TFA is disabled** (`TFAEnabled:false`) — the auth-story implementer must fork the template's `repositoryStorage` adapter so it no longer references TOTP columns. `/users/me` serializes `totpEnabled:false` as a static.
5. **Project-scoped role keys are `projectAdmin`, `projectEditor`, `projectViewer`** (DB seed, Doc A+D). Doc C's `editor`/`viewer` refer to the same rows — use the canonical `projectEditor`/`projectViewer` keys everywhere.
6. **User avatar is dropped** (Doc A). REST `avatarUrl` is removed (do not serialize; do not presign). If avatars are wanted later, add `avatarStorageId string` then. Doc C's `avatarUrl` field and Doc D's avatar-migration are both dropped.
7. **`ImageTag.type` enum = `template|default|manual|custom`** (full set, Doc A+D). Doc C's `default|manual|custom` omitted `template`; `template` stays for migrated data.
8. **`ImageTagAssignment.type` is required** with values `manual|inferred|default` (Doc A elevates it). Importer **must coalesce** any PB row with an empty `type` to `manual`. ⚠️ FLAG: confirm `manual` is the right default for legacy untyped rows before running the import; it is the only blocking importer guess left.
9. **`getIdUrlParameter` is relaxed to a non-empty string** for all string-PK resources; **user routes keep `uuid.Parse`** (User PK is UUID). The template's single uuid-parsing helper must be split or parameterized.
10. **Image-create body cap raised to ≥2 MB** (PB `exifData` max was 2 MB).

---

# 1. Patterns brief — what every backend agent MUST mirror

These template files are the canonical shape. Read them before writing any entity. Do not invent new structure; copy the pattern and substitute the entity.

### Template files to mirror (in `mxcd/agentic-template`)
| File | What to copy |
|---|---|
| `ent/schema/mixin.go` | Mixin shape. **Replaced** here by our two-mixin `StringIDMixin`+`AuditMixin` split (§2). |
| `internal/repository/user.go` | The **per-entity repository pattern** — the single most important file. |
| `internal/authentication/authentication.go` | go-basicauth wiring: `repositoryStorage` adapter, `UserTransformer`, path rules, `ensureDefaultAdmin`. |
| `internal/authorization/authorization.go` | `Checker` combinators (`All`/`Any`/`Not`), `IsUser`/`IsAdmin`/`HasUserID`, `Can<Verb><Entity>(viewer, target) bool` + `<Entity>BroadcastFilter`. |
| `internal/event/websocket.go` | WS manager, `WebsocketMessage[T]{object,action,data}`, `BroadcastWebsocketMessageFiltered`. |
| `internal/server/util.go` | `ListResponse[T]`, pagination/sort/order parsing, `{message,code}` error envelope, `getIdUrlParameter`. |
| `internal/repository/repository.go` | `PaginationParameters`, `modelUpdateStatus`/`SetFieldChanged`, `isPostgres()`, `safeGo`. |
| `internal/util/context.go` | `GetUser(ctx)`, `GetActorID(ctx)`. |

### Per-entity repository pattern (mandatory, mirror `user.go`)
Every entity gets `internal/repository/<entity>.go` with:
- **Typed param structs** per operation: `Get<E>Parameters` (filters + `*PaginationParameters`), `Create<E>Parameters`, `Update<E>Parameters`. Optional fields are pointers; "field provided" = non-nil pointer (partial-update semantics).
- **`Get<E>(ctx, id)`** — single read, cache-through where a cache exists (`r.<e>Cache`), wide-event annotation (`wideevent.FromContext(ctx).Str("db.operation",...)`).
- **`Get<E>s(ctx, *params)` → `(items, total, error)`** — build `[]predicate.<E>`, `And(...)`, apply `Limit/Offset/Order(pagination.GetOrder())`, then a separate `Count`.
- **`Create<E>`** — set fields, `SetCreatedBy/SetUpdatedBy(util.GetActorID(ctx))`, `Save`, then two `safeGo` calls: WS broadcast + `CreateAuditLog`.
- **`Update<E>`** — **transactional with `.ForUpdate()` on Postgres**: open `tx`, `tx.<E>.Query().Where(id)` + `if r.isPostgres() { q = q.ForUpdate() }`, load current, build update, **track every change via `modelUpdateStatus.SetFieldChanged(field, old, new)`** (only on actual value change). **If nothing changed → rollback and return the unchanged item (no audit row).** Else `Save`, `Commit`, evict cache, re-fetch, then `safeGo` WS + audit with `GetChangedFieldData()`.
- **`Delete<E>`** — `DeleteOneID`, evict cache, `safeGo` WS (minimal `&ent.<E>{ID:id}`) + audit.
- **Sensitive values** (`passwordHash`) tracked as `"<redacted>"` in change data, never serialized.

### Audit-on-mutation
Create/Update/Delete each emit a `CreateAuditLog` via `safeGo(...)` with `context.WithoutCancel(ctx)`, `Action`, `ObjectType`, `ObjectId`, and `Data` (for updates: `{"changes": updateStatus.GetChangedFieldData()}`). Reads are never audited.

### WebSocket broadcast — **DEFERRED / YAGNI for shutterbase entities**
The template broadcasts every entity mutation filtered by `<Entity>BroadcastFilter`. **For the shutterbase rewrite, do NOT implement per-entity WS broadcasts.** The only live WS traffic is the **10-second time-sync tick** (`/ws`, S9) and the template's `ping`. Keep `event/websocket.go` and the `BroadcastWebsocketMessageFiltered` plumbing intact, but new entity repos **omit the `safeGo(event.Broadcast...)` block**. Add per-entity broadcast only when a concrete realtime UI need appears.
`// ponytail: per-entity WS broadcast is dead flexibility until a realtime gallery exists; tick is the only consumer.`

### Response & error conventions
- **List envelope:** `ListResponse[T]{ limit, offset, total, items }`. Query: `limit` (default 100, clamp 1..500), `offset` (≥0), `sort` (per-resource allowlist, default `createdAt`), `order` (`asc|desc`, default `desc`).
- **Error envelope (controllers):** `{ "message": "...", "code": "..." }`. Codes: `missing_id`, `invalid_id`, `invalid_limit`, `invalid_offset`, `invalid_sort`, `invalid_order`, plus per-resource codes (§4). Body-bind failures: gin default `{"error":"<err>"}`.
- **Error envelope (go-basicauth routes):** `{ "error":"...", "message":"..." }`. **Both shapes are intentional and stable** — the frontend handles both.

### Identity & base path
- **Base path `/api/v1`.** Single binary; gin. All `/api/v1/*` private except `/api/v1/health`; `/ws` private.
- **`util.GetUser(ctx)`** returns the **effective** user (impersonated if active, else real). All authz, queries, and `createdBy/updatedBy` use the effective user.
- **PKs:** `User.id` = `uuid.UUID`; every other entity = 15-char PB-style string (preserved verbatim → S3 keys & URLs stay valid).

---

# 2. Ent schema (FROZEN)

`go generate ./ent` → Atlas versioned migration (§3). Imports as needed: `entgo.io/ent/dialect/entsql`, `schema/index`, `schema/edge`, `schema/field`, `github.com/google/uuid`.

## 2.1 Mixins — `ent/schema/mixin.go` (replaces template `DefaultMixin`)

```go
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
	"github.com/google/uuid"

	"github.com/shutterbase/shutterbase/internal/util" // util.NewID
)

// StringIDMixin: 15-char PB-style string PK. Used by every entity EXCEPT User.
type StringIDMixin struct{ mixin.Schema }

func (StringIDMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").MaxLen(15).Immutable().
			DefaultFunc(util.NewID).StructTag(`json:"id"`),
	}
}

// AuditMixin: timestamps + actor. createdBy/updatedBy are uuid.UUID (User PK type),
// Optional+Nillable because migrated PB rows have no author. Used by EVERY entity.
type AuditMixin struct{ mixin.Schema }

func (AuditMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("createdAt").StorageKey("createdAt").
			Immutable().Default(time.Now).StructTag(`json:"createdAt"`),
		field.Time("updatedAt").StorageKey("updatedAt").
			Default(time.Now).UpdateDefault(time.Now).StructTag(`json:"updatedAt"`),
		field.UUID("createdBy", uuid.UUID{}).
			Optional().Nillable().Immutable().StructTag(`json:"createdBy,omitempty"`),
		field.UUID("updatedBy", uuid.UUID{}).
			Optional().Nillable().StructTag(`json:"updatedBy,omitempty"`),
	}
}
```

> **User** uses `AuditMixin{}` **only** (its own `field.UUID("id",...)`). Every other entity uses `StringIDMixin{}` + `AuditMixin{}`.

### `internal/util/id.go`
```go
package util

import "crypto/rand"

const idAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789" // PB id alphabet

func NewID() string {
	b := make([]byte, 15)
	rand.Read(b)
	for i := range b {
		b[i] = idAlphabet[int(b[i])%len(idAlphabet)]
	}
	return string(b)
}
// ponytail: modulo bias over 36 chars is negligible for a 36^15 keyspace; switch to
// rejection sampling only if a collision audit ever flags it.
```

## 2.2 Entities

**FK type rule:** edges targeting **User** use `field.UUID("<x>_id", uuid.UUID{})`; edges targeting any string-PK entity use `field.String("<x>_id")`. Every `edge.To(...).Field("<x>_id")` requires the matching field declared in `Fields()`. Nullable FKs get `.Optional().Nillable()`. Cascade = `Annotations(entsql.OnDelete(entsql.Cascade))` on the edge.

### 1. User — `user.go` (PK uuid.UUID; `AuditMixin` only)
| Field | Type | Constraints |
|---|---|---|
| `id` | `field.UUID("id", uuid.UUID{})` | `.Immutable().Default(uuid.New)` |
| `legacyId` | String | `.Optional().Immutable().Unique()` (PB id) |
| `username` | String | `.NotEmpty().Unique()` |
| `firstName` | String | `.NotEmpty()` |
| `lastName` | String | `.NotEmpty()` |
| `copyrightTag` | String | `.Optional()` |
| `active` | Bool | `.Default(false)` |
| `email` | String | `.Optional().Unique()` |
| `verified` | Bool | `.Default(false)` |
| `passwordHash` | String | `.Optional().Sensitive()` `json:"-"` |
| `forcePasswordChange` | Bool | `.Default(false)` |
| `provider` | `field.Enum("provider").Values("local").Default("local")` | (keep minimal; expand if OIDC returns) |
| `role` | `field.Enum("role").Values("user","admin").Default("user")` | global role |
| `active_project_id` | `field.String("active_project_id").Optional().Nillable()` | FK→Project, nullable |

**Edges:**
```go
edge.To("cameras", Camera.Type)
edge.To("uploads", Upload.Type)
edge.To("images", Image.Type)
edge.To("projectAssignments", ProjectAssignment.Type)
edge.To("activeProject", Project.Type).Field("active_project_id").Unique()
```
**Indexes:** `index.Fields("firstName","lastName").Unique()` · `index.Fields("email").Unique()` · `index.Fields("active_project_id")`.

### 2. Project — `project.go` (string PK)
| Field | Type | Constraints |
|---|---|---|
| `name` | String | `.NotEmpty().Unique()` |
| `description` | String | `.NotEmpty()` (⚠️ PB UNIQUE index on description is **dropped** — accidental constraint) |
| `copyright` | String | `.NotEmpty()` |
| `copyrightReference` | String | `.NotEmpty()` |
| `locationName` | String | `.NotEmpty()` |
| `locationCode` | String | `.NotEmpty()` |
| `locationCity` | String | `.NotEmpty()` |
| `aiSystemMessage` | String | `.Optional()` |

**Edges:** `edge.To("cameras",Camera.Type)` (optional, skip if unused) · `uploads` · `images` · `imageTags` · `projectAssignments` · `edge.From("activeForUsers",User.Type).Ref("activeProject")`.
**Indexes:** `index.Fields("name").Unique()`.

### 3. Camera — `camera.go` (string PK)
| Field | Type | Constraints |
|---|---|---|
| `name` | String | `.MinLen(3)` |
| `user_id` | `field.UUID("user_id", uuid.UUID{})` | required |

**Edges:** `edge.To("user",User.Type).Field("user_id").Unique().Required()` **+ `Annotations(entsql.OnDelete(entsql.Cascade))`** (PB cascadeDelete) · `timeOffsets` · `uploads` · `images`.
**Indexes:** `index.Fields("user_id")` · `index.Fields("name","user_id").Unique()`.

### 4. Upload — `upload.go` (string PK; ex-PB `batches`)
| Field | Type | Constraints |
|---|---|---|
| `name` | String | `.NotEmpty()` |
| `project_id` | String | required |
| `user_id` | uuid.UUID | required |
| `camera_id` | String | required |

**Edges (all `.Unique().Required()`, no cascade):** `project`(→Field `project_id`) · `user`(→`user_id`) · `camera`(→`camera_id`) · `edge.To("images",Image.Type)`.
**Indexes:** `project_id`, `user_id`, `camera_id`.

### 5. Image — `image.go` (string PK; hot table)
| Field | Type | Constraints |
|---|---|---|
| `fileName` | String | `.NotEmpty()` |
| `computedFileName` | String | `.Optional().Unique()` |
| `storageId` | String | `.NotEmpty().Unique()` |
| `exifData` | `field.JSON("exifData", map[string]any{})` | `.Optional()` → jsonb |
| `imageTags` | `field.JSON("imageTags", []string{})` | `.Optional().Default([]string{})` → jsonb |
| `capturedAt` | Time | `.Optional().Nillable()` |
| `capturedAtCorrected` | Time | `.Optional().Nillable()` |
| `inferredAt` | Time | `.Optional().Nillable()` |
| `size` | Int | `.NonNegative()` required |
| `width` | Int | `.Optional().Nillable().NonNegative()` |
| `height` | Int | `.Optional().Nillable().NonNegative()` |
| `user_id` | uuid.UUID | required |
| `upload_id` | String | required |
| `project_id` | String | required |
| `camera_id` | String | required |

**Edges (all `.Unique().Required()`):** `user`(→`user_id`, no cascade) · `upload`(→`upload_id`, **cascade**) · `project`(→`project_id`, **cascade**) · `camera`(→`camera_id`, no cascade) · `edge.To("imageTagAssignments",ImageTagAssignment.Type)`.
**Indexes:** `computedFileName`(U) · `storageId`(U) · `project_id` · `upload_id` · `user_id` · `camera_id` · `capturedAtCorrected` · `index.Fields("project_id","capturedAtCorrected")` (hot gallery) · GIN on `imageTags`:
```go
index.Fields("imageTags").Annotations(entsql.IndexType("GIN"), entsql.OpClass("jsonb_path_ops"))
```
> If the installed ent lacks `entsql.OpClass`, hand-edit the migration: `CREATE INDEX image_imagetags_gin ON images USING GIN ("imageTags" jsonb_path_ops);`

### 6. ImageTag — `image_tag.go` (string PK)
| Field | Type | Constraints |
|---|---|---|
| `name` | String | `.NotEmpty()` |
| `description` | String | `.NotEmpty()` |
| `isAlbum` | Bool | `.Default(false)` |
| `type` | `field.Enum("type").Values("template","default","manual","custom")` | required |
| `project_id` | String | required |

**Edges:** `project`(→`project_id`, **cascade**, `.Unique().Required()`) · `edge.To("tagAssignments",ImageTagAssignment.Type)`.
**Indexes:** `project_id` · `index.Fields("name","project_id").Unique()`.

### 7. ImageTagAssignment — `image_tag_assignment.go` (string PK)
| Field | Type | Constraints |
|---|---|---|
| `type` | `field.Enum("type").Values("manual","inferred","default")` | **required** |
| `image_id` | String | required |
| `image_tag_id` | String | required |

**Edges (all `.Unique().Required()`):** `image`(→`image_id`, **cascade**) · `imageTag`(→`image_tag_id`, no cascade — tag delete repaired in app).
**Indexes:** `index.Fields("image_id","image_tag_id").Unique()` (idempotency) · `image_id` · `image_tag_id`.

### 8. TimeOffset — `time_offset.go` (string PK)
| Field | Type | Constraints |
|---|---|---|
| `serverTime` | Time | required |
| `cameraTime` | Time | required |
| `timeOffset` | Int | `.Optional()` |
| `camera_id` | String | required |

**Edges:** `camera`(→`camera_id`, **cascade**, `.Unique().Required()`).
**Indexes:** `camera_id`.

### 9. Role — `role.go` (string PK)
| Field | Type | Constraints |
|---|---|---|
| `key` | String | `.MinLen(3).Unique()` |
| `description` | String | `.NotEmpty()` |

**Edges:** `edge.To("projectAssignments",ProjectAssignment.Type)`.
**Indexes:** `index.Fields("key").Unique()`.
**Seed (preserve PB ids, migration `1716221995`):** `projectAdmin`, `projectEditor`, `projectViewer`, `admin`, `user`.

### 10. ProjectAssignment — `project_assignment.go` (string PK)
| Field | Type | Constraints |
|---|---|---|
| `project_id` | String | required |
| `user_id` | uuid.UUID | required |
| `role_id` | String | required |

**Edges (all `.Unique().Required()`):** `project`(→`project_id`, **cascade**) · `user`(→`user_id`, **cascade**) · `role`(→`role_id`, no cascade).
**Indexes:** `index.Fields("project_id","user_id").Unique()` · `user_id` · `role_id`.

## 2.3 Dropped from PB
`inferences` collection · `images.downloadUrls` (computed on read) · `users.avatar` · `projects.description` UNIQUE index · PB-internal (`emailVisibility`, `tokenKey`, `lastResetSentAt`, `lastVerificationSentAt`).

---

# 3. Schema migrations — ent auto-migrate (FROZEN, amended)

**Decision (amended from Atlas):** use **ent auto-migration** (`client.Schema.Create`), NOT Atlas/golang-migrate versioned migrations. Rationale: v3 migrates into a **fresh empty Postgres** (the importer loads it) so there is no existing data to endanger at create time; the app is small and single-team; Atlas's generator + throwaway dev-DB + `golang-migrate` dep + hand-reviewed SQL are machinery this app does not need before the deadline. This is also the template's default (undivergent).

## 3.1 Boot apply
`internal/database` runs `client.Schema.Create(ctx, opts...)` at server boot for **both** Postgres and SQLite. Options pinned for safety:
- `migrate.WithDropIndex(false)`, `migrate.WithDropColumn(false)` (the defaults) — **additive-only**; auto-migrate never drops data. A destructive change (drop/rename a column) is a **deliberate manual SQL step** when it is actually needed, not automated.
- `migrate.WithForeignKeys(true)`.
- Fail-closed: a `Schema.Create` error -> `log.Panic` (do not serve a half-migrated schema).
`// ponytail: auto-migrate is additive-only via these opts; the rare destructive change is a hand-written one-off, cheaper than carrying Atlas for a single-team app.`

## 3.2 GIN jsonb index
The `images.imageTags` jsonb GIN (`jsonb_path_ops`) index is declared on the ent schema (`entsql.IndexType("GIN")`, `entsql.OpClass("jsonb_path_ops")`). ent auto-migrate applies it. **Verify** after `Schema.Create` on real Postgres that `pg_indexes` shows the GIN index; if the pinned ent version does not emit the opclass via auto-migrate, run ONE idempotent statement right after `Schema.Create`:
`CREATE INDEX IF NOT EXISTS image_image_tags ON images USING GIN ("imageTags" jsonb_path_ops);`

## 3.3 Removed
Delete the Atlas apparatus: `ent/migrate/` (generator `main.go`/`migrate.go`/`schema.go` + `migrations/` + `embed.go`), the `--feature sql/versioned-migration` flag in `ent/generate.go` (then regenerate ent), and the `github.com/golang-migrate/migrate/v4` dependency (`go mod tidy`).

## 3.4 cmd/migrate (thin)
Keep a minimal `cmd/migrate`: `create` (= `Schema.Create`, idempotent; the Dockerfile `migrate` target uses this) and a **DEV-gated** `drop` (drop all tables — for the importer fresh-start and `just migrate-reset`). No versioned up/down/force/version.

## 3.5 Harness + importer
- Test harness `Migrate` step = the **same** `Schema.Create` call as boot, against the testcontainers Postgres.
- Importer = `drop` -> `Schema.Create` -> import into the fresh DB (re-runnable). S3 untouched.

## 3.6 Backup
The deploy pipeline takes `pg_dump -Fc` before each deploy (operational, not server-side). For a single-team app this plus provider PITR is sufficient; there is no migration-version state to coordinate.

# 4. REST API contract (FROZEN)

Base `/api/v1`, gin, single binary. Cookie-session via go-basicauth (`basicauth_session`, HttpOnly, Secure off in DEV, SameSite=Lax; frontend `withCredentials:true`, WASM `credentials:include`). Programmatic clients: `Authorization: ApiKey <keyId>.<secret>` (S11) → same effective-user resolution. All `/api/v1/*` private except `/api/v1/health`.

**Global:** IDs = opaque strings (`:id` non-empty string for string-PK resources; uuid-parsed for user routes). Timestamps RFC3339 UTC. List envelope + pagination/sort/order per §1. Status codes: 200/201/204/400/401/403/404/409/413/429/500. Off-allowlist `sort` → `400 {"code":"invalid_sort"}`.

**Roles & authz:** global `admin|user`; project-scoped `projectAdmin|projectEditor|projectViewer`. Entity checkers (`CanViewImage`, `CanEditImageTag`, …) shared between HTTP and (future) WS filtering. Fixes PB's duplicate-`projectAdmin` bug.

## 4.1 Auth
- **POST `/auth/login`** `{identifier,password}` → 200 effective user (`/users/me` shape) + cookie; 401 `{"error":"invalid_credentials",...}`. Migrated users with `forcePasswordChange` carry the flag → frontend routes to change-password.
- **POST `/auth/logout`** → 200 `{"message":"Logout successful"}`, clears cookie.
- **PUT `/auth/change-password`** `{currentPassword,newPassword,newPasswordConfirm}` → 200 updated user, clears `forcePasswordChange`; 400 `passwords_do_not_match|password_requirements_not_met`; 403 `incorrect_password`.
- **POST `/auth/impersonate/:userId`** (S8) — gated on **real** user admin (`IsRealAdmin`, re-checked). Sets `impersonatedUserId`. 200 → effective user incl. `impersonating` block; 403 if real user not admin; 404 unknown. Audited.
- **DELETE `/auth/impersonate`** → 200 real user (no `impersonating`). Audited.

## 4.2 `GET /users/me`
Effective user + role (`{id,key,description}` mapped from enum) + `activeProject` (`{id,name}`|null) + `projectAssignments[]` + (only when impersonating) `impersonating:{realUserId,realUserName}` (key **omitted** when not impersonating). Includes `username,email,verified,active,firstName,lastName,copyrightTag,forcePasswordChange,totpEnabled(=false),createdAt,updatedAt`. **No `avatarUrl`.** 200 always when authed; 401 otherwise.

## 4.3 Images
**Image object:** `id, fileName, computedFileName, exifData(jsonb,may be {}), capturedAt, capturedAtCorrected, width, height, size, storageId`, embedded `user{id,firstName,lastName,copyrightTag}`, `camera{id,name}`, `project{id,name}`, `upload{id,name}`, `tags[]` (`{id,type,tag{...}}` from assignments), `imageTags[]` (denormalized id list), `downloadUrls{original,256,512,1024,2048}` (presigned, **computed on read** from `storageId` via `util.GetObjectIds`+LRU; key `XX/<storageId>[-<size>].jpg`, `XX`=first 2 chars; sizes from `THUMBNAIL_SIZES` default `256,512,1024,2048`), `createdAt, updatedAt`.

- **LIST `GET /images`** params: `projectId` (**required**, 400 `missing_project`; caller admin or assigned else 403), `uploadId?`, `cameraId?`, `userId?`, `search?` (`computedFileName ILIKE %s% OR fileName ILIKE %s%`), `tagId` (repeated, **AND** via jsonb `@>` over GIN), `orientation?` (`portrait`:w<h / `landscape`:w>h; **null w/h excluded** when set, 400 `invalid_orientation`), `limit/offset`, `sort` (allowlist: `capturedAtCorrected`(default),`capturedAt`,`createdAt`,`updatedAt`,`computedFileName`,`fileName`)/`order`. → 200 `ListResponse[Image]`; 403.
- **GET `/images/:id`** → 200 / 403 (`CanViewImage`) / 404.
- **POST `/images`** (body ≥2 MB) `{fileName,storageId,size,width,height,capturedAt,exifData,cameraId,uploadId,projectId}`. Authz: project member. Server computes `computedFileName`, `capturedAtCorrected` (closest time-offset), applies default tags, links denormalized `imageTags`, enqueues AI. → 201 Image / 400 / 403 / 409 (dup `storageId`/`computedFileName`).
- **PUT `/images/:id`** partial; editable `fileName,capturedAt,exifData,cameraId,uploadId`(re-parent admin/projectAdmin only). Recompute `computedFileName`/`capturedAtCorrected` if inputs change. No-op = rollback (no audit). → 200/403/404.
- **DELETE `/images/:id`** owner/projectAdmin/admin; deletes S3 objects by `storageId` prefix + cascades assignments. → 204/403/404.

## 4.4 Image Tags `/image-tags`
Object: `{id,name,description,isAlbum,type,project{id,name},createdAt,updatedAt}`. `type` ∈ `template|default|manual|custom`.
- **LIST** params `projectId`(required), `search`(ILIKE name), `type`, `limit/offset/sort`(`name`,`type`,`createdAt`,`updatedAt`)/`order`. Any authed. → 200.
- **GET `/:id`** → 200/404.
- **POST** `{name,description,isAlbum?,type,projectId}`. Authz: `type∈{default,manual}` → admin/projectAdmin; `type=custom` → any member. (`template` not creatable via API.) → 201/400/403/409 (dup name in project).
- **PUT `/:id`** authz by resulting `type`. → 200/403/404.
- **DELETE `/:id`** admin/projectAdmin; repairs denormalized `images.imageTags`. → 204/403/404.

## 4.5 Image Tag Assignments `/image-tag-assignments`
Object: `{id,type,image{id},tag{id,name,type,isAlbum},createdAt,updatedAt}`. `type` ∈ `manual|inferred|default`. Unique `(image,imageTag)`.
- **LIST** params `imageId?`,`tagId?`,`limit/offset`. → 200.
- **POST** `{imageId,imageTagId,type}` — **idempotent**: existing `(image,tag)` → 200 existing row (not 409). Updates denormalized `images.imageTags` + `images.updatedAt`. Authz: `projectEditor`/`projectAdmin`/`admin`; `projectViewer`→403. → 201(created)/200(existing).
- **GET `/:id`** → 200/404.
- **DELETE `/:id`** authz as POST; repairs denormalized list. → 204/403/404.

## 4.6 Projects `/projects`
Object: `{id,name,description,copyright,copyrightReference,locationName,locationCode,locationCity,aiSystemMessage,createdAt,updatedAt}`.
- **LIST** `search`,`limit/offset/sort`(`name`,`createdAt`,`updatedAt`)/`order`. Admin all; others only assigned. → 200.
- **GET `/:id`** admin/assigned else 403; 404.
- **POST** admin only; all fields required except `aiSystemMessage`. → 201/400/403/409 (dup name — description UNIQUE dropped per §2).
- **PUT `/:id`** admin only, partial. → 200/403/404.
- **DELETE `/:id`** admin only; cascades tags/images/assignments. → 204/403/404.

## 4.7 Project Assignments `/project-assignments`
Object: `{id,project{id,name},user{id,firstName,lastName,email},role{id,key,description},createdAt,updatedAt}`. Unique `(project,user)`.
- **LIST** `projectId?`,`userId?`,`limit/offset`. Any authed. → 200.
- **GET `/:id`** → 200/404.
- **POST** admin `{projectId,userId,roleId}` → 201/400/403/409 (dup pair).
- **PUT `/:id`** admin `{roleId}` → 200/403/404.
- **DELETE `/:id`** admin → 204/403/404.

## 4.8 Cameras `/cameras`
Object: `{id,name,user{id,firstName,lastName},createdAt,updatedAt}`.
- **LIST** `userId?`,`search`,`limit/offset/sort`(`name`,`createdAt`,`updatedAt`)/`order`. Admin all; others own (`user.id=me`). → 200.
- **GET `/:id`** admin/owner/403/404.
- **POST** `{name,userId?}` (`userId` defaults to effective user). Any authed. → 201/400.
- **PUT `/:id`** admin/owner, partial (`name`). → 200/403/404.
- **DELETE `/:id`** admin/owner; cascades `time_offsets`. → 204/403/404.

## 4.9 Uploads `/uploads` (PB `batches`)
Object: `{id,name,project{id,name},user{id,firstName,lastName},camera{id,name},imageCount?,createdAt,updatedAt}`.
- **LIST** `projectId?`,`userId?`,`limit/offset/sort`(`name`,`createdAt`,`updatedAt`)/`order`. Admin/projectAdmin all in project; user own. → 200.
- **GET `/:id`** → 200/403/404.
- **POST** `{name,projectId,cameraId,userId?}` (`userId` defaults effective). Project member. → 201/400/403.
- **PUT `/:id`** admin/projectAdmin/owner, partial (`name`). → 200/403/404.
- **DELETE `/:id`** admin/projectAdmin/owner; cascades images. → 204/403/404.

## 4.10 Time Offsets `/time-offsets`
Object: `{id,serverTime,cameraTime,timeOffset,camera{id,name},upToDate(=serverTime within 24h),createdAt,updatedAt}`.
- **LIST** `cameraId?`,`limit/offset/sort`(`serverTime`,`createdAt`)/`order`. Admin all; others `camera.user.id=me`. → 200.
- **GET `/:id`** → 200/403/404.
- **POST** `{cameraId,serverTime,cameraTime}`; server computes `timeOffset=serverTime−cameraTime` (seconds). Admin/camera owner. → 201/400/403.
- **PUT `/:id`** / **DELETE `/:id`** **admin only**. → 200|204/403/404.

## 4.11 Roles `/roles`
Object: `{id,key,description,createdAt,updatedAt}`.
- **LIST** `limit/offset/sort`(`key`,`createdAt`)/`order`. Any authed. → 200.
- **GET `/:id`** → 200/404.
- **POST/PUT/DELETE** admin only (seeded; create rare). DELETE 409 if in use.

## 4.12 Users `/users`
Object = `/users/me` minus `impersonating`. `passwordHash`/secrets never serialized.
- **LIST** `search`(firstName/lastName/email/username),`limit/offset/sort`(`username`,`email`,`name`,`active`,`role`,`createdAt`,`updatedAt`)/`order`. Admin (at least admin/projectAdmin for pickers). → 200.
- **GET `/:id`** admin/self/403/404.
- **POST** **admin only** (no self-signup) `{username,email,password,firstName,lastName,copyrightTag,active,roleId,forcePasswordChange}`. Password validated (min len + upper/lower/digit; 400 `password_requirements_not_met`), hashed via `basicauth.HashPassword`. `roleId` maps to the global enum (`admin`/`user` row → enum). → 201/403/409 (dup username/email/`(firstName,lastName)`).
- **PUT `/:id`** admin/self. Self: `firstName,lastName,copyrightTag,email,password`. Admin-only: `active,roleId,forcePasswordChange,activeProjectId` (403 if non-admin sends). Partial. → 200/403/404.
- **PATCH `/users/me/active-project`** `{projectId}` (must be assigned). → 200/403.
- **DELETE `/:id`** admin only → 204/403/404.

## 4.13 Custom routes
- **GET `/upload-url?name=<key>`** presigned **PUT** (4-min). `name` validated `^[0-9a-zA-Z]{2}/<storageId>(-\d+)?\.jpg$` (reject traversal). Rate-limited per user. → 200 `{url}` / 400 `missing_name`|`invalid_key` / 401 / 429.
- **GET `/download/:id/:res`** streams JPEG with EXIF/IPTC injected (folded `exiftool` shell-out, ctx timeout + temp dir + concurrency semaphore + size limit, S10). `:res ∈ original|256|512|1024|2048`. Authz `CanViewImage`. → 200 `image/jpeg` + `Content-Disposition: attachment; filename="<computedFileName>"` / 400 `invalid_resolution` / 403 / 404 / 500.
- **GET `/statistics/:projectId`** per-tag counts (join/count over denormalized `imageTags`), LRU 5-min (`TagCountCache`). Admin/assigned. → 200 `{tags:[{id,name,description,type,count}]}` / 403 / 404.
- **GET `/sync-image-tags`** rebuild all `images.imageTags` from assignments (`util.SyncImageTags`). **Admin only.** → 200 `{synced:N}` / 403.
- **GET `/ws`** (not under `/api/v1`) cookie-auth WS; same-origin upgrade check; broadcasts time-sync tick `{"object":"time","action":"tick","data":<unixSeconds>}` every 10s (S9). Per-entity broadcasts deferred (§1).

## 4.14 Sort allowlist summary
| Resource | Allowed `sort` (default first) |
|---|---|
| images | capturedAtCorrected, capturedAt, createdAt, updatedAt, computedFileName, fileName |
| image-tags | name, type, createdAt, updatedAt |
| image-tag-assignments | createdAt, updatedAt |
| projects | name, createdAt, updatedAt |
| project-assignments | createdAt, updatedAt |
| cameras | name, createdAt, updatedAt |
| uploads | name, createdAt, updatedAt |
| time-offsets | serverTime, createdAt |
| roles | key, createdAt |
| users | username, email, name, active, role, createdAt, updatedAt |

---

# 5. Importer field-map (S12, PB SQLite → Ent/Postgres)

**Conventions:** PB SQLite column = the field's `name`. Every table has `id,created,updated`; auth tables add `username,email,emailVisibility,verified,passwordHash,tokenKey,lastResetSentAt,lastVerificationSentAt`. `created→createdAt`, `updated→updatedAt` (parse `"2006-01-02 15:04:05.000Z"`). Single relation = string-id column; multi relation = JSON array of string ids.

**PK strategy:** non-user entities `.SetID(pbID)` (15-char string preserved → string FKs need no remap). **Users adopt UUID** (`uuid.New()`); PB id → `legacyId`.

**User-FK remap (build during user import, keep in memory):** `pbUserId(string) → uuid.UUID`, applied to `cameras.user`, `uploads.user`, `images.user`, `project_assignments.user` via `.SetUserID(m[pbUserId])`.

**`createdBy`/`updatedBy`:** **nullable** (§0.2 — resolves Doc D's open question). For rows with an owning user, set both to `m[ownerPbUserId]`; for owner-less rows (`roles`,`projects`,`image_tags`,`time_offsets`) **leave null**. No sentinel user.

**Import order** (FK-safe): roles → users(`activeProject=NULL`) → projects → cameras → time_offsets → uploads → image_tags → project_assignments → images → image_tag_assignments → **patch users.activeProject**.

### 1. roles (`fbvy1v7txj0ooy4`) — first
`id→.SetID`, `key`, `description`. Seed/preserve: `projectAdmin,projectEditor,projectViewer,admin,user`. `createdBy/updatedBy`=null.

### 2. users (`_pb_users_auth_`) — `activeProject=NULL`
| PB | Ent | Notes |
|---|---|---|
| `id` | `legacyId` + `id=uuid.New()` | **register `pbId→uuid` in map** |
| `username` | `username` | required, unique (PB always populates) |
| `email` | `email` | optional, unique — carry it |
| `verified` | `verified` | bool — carry it or logins regress (PB `onlyVerified:true`) |
| `passwordHash` | `passwordHash` | **BCRYPT verbatim** (`$2a$…`); go-basicauth `BcryptVerifier` rehashes to argon2id on login. **No temp pw**, `forcePasswordChange=false` |
| `firstName`/`lastName`/`copyrightTag`/`active` | same | |
| `role` (PB rel→roles.key) | enum `role` | `key=="admin"`→`admin`, else `user` (§0.3) |
| `activeProject` | **defer** → step 11 | |
| `projectAssignments` | **ignore** | derived; canonical = join table |
| `avatar` | **DROP** (§0.6) | |
| `emailVisibility,tokenKey,lastResetSentAt,lastVerificationSentAt` | **drop** | |
| — | `provider="local"`, `createdBy/updatedBy`=self `m[pbId]` | |

### 3. projects (`whgae0tyjp10p6e`)
`.SetID`, `name,description,copyright,copyrightReference,locationName,locationCode,locationCity` (all required, non-empty or insert fails), `aiSystemMessage` (optional — easy to miss). `createdBy/updatedBy`=null.

### 4. cameras (`5nhk5rl7djdx4lf`)
`.SetID`, `name`(min 3), `user`→`SetUserID(m[pbUserId])` required. `createdBy/updatedBy=m[user]`. Unique `(name,user)`.

### 5. time_offsets (`8k5kgh4acgwhuyo`)
`.SetID`, `serverTime`/`cameraTime`(date→time required), `timeOffset`(**int**, optional — easy to miss), `camera`(string FK, no remap). `createdBy/updatedBy`=null.

### 6. uploads (`55ajrfhmhgm37tz`, table `batches`→`uploads`)
`.SetID`, `name`, `project`(string FK), `user`→remap, `camera`(string FK, required, added `1716748142`). `createdBy/updatedBy=m[user]`.

### 7. image_tags (`xmc92cdxvv1ijq4`)
`.SetID`, `name`,`description`,`isAlbum`(bool), `type` enum `template|default|manual|custom` required, `project`(string FK). `createdBy/updatedBy`=null. Unique `(name,project)`.

### 8. project_assignments (`bnggeaxuv84cfwh`)
`.SetID`, `project`(string FK), `user`→remap, `role`(string FK). `createdBy/updatedBy=m[user]`. Unique `(project,user)`.

### 9. images (`5020t772ltvs9da`)
| PB | Ent | Notes |
|---|---|---|
| `id` | `.SetID` | |
| `fileName` | `fileName` | required |
| `computedFileName` | same | optional, unique |
| `exifData` | jsonb verbatim | PB max 2 MB → body cap ≥2 MB |
| `capturedAt`/`capturedAtCorrected` | date→time, optional | easy to miss |
| `size` | **int**, required | easy to miss |
| `width`/`height` | **int**, optional | easy to miss |
| `storageId` | required, unique | **verbatim** (S3 key untouched) |
| `user` | remap | |
| `upload` | string FK (was `batch`, renamed `1716762577`) | |
| `project`/`camera` | string FK | |
| `imageTags` | jsonb array of tag ids **verbatim** | |
| `downloadUrls` | **DROP** | |
| `inferredAt`/`imageTagAssignments(fwd)` | absent/derived | |
| — | `createdBy/updatedBy=m[user]` | |

### 10. image_tag_assignments (`lm56zd5xql95a0m`) — after images
`.SetID`, `type` enum `manual|inferred|default` **required** → **coalesce empty PB rows → `manual`** (§0.8 ⚠️ confirm), `imageTag`(string FK), `image`(string FK). `createdBy/updatedBy` = `m[image.user]` or null. Unique `(imageTag,image)` + indexes on `image`,`imageTag`.

### 11. PATCH users.activeProject (final)
For each PB user with non-empty `activeProject`: `client.User.UpdateOneID(m[pbUserId]).SetActiveProjectID(pbProjectId)` (project id is preserved string, no remap).

### Dropped entirely
`inferences` (`hk5sqca2ka333kn`) · `images.downloadUrls` · `users.avatar` · PB-internal user columns · `users.projectAssignments` (derived).

### Importer interaction with migrations
Runs `migrate drop` → `migrate up` → import into a **fresh empty Postgres** (re-runnable, no backup needed). S3 untouched (`storageId`/keys preserved).

---

## Open flags for implementers (the only unresolved items)
1. **§0.8** — `ImageTagAssignment.type` default for legacy untyped PB rows is set to `manual`; confirm before running the production import (only blocking importer guess).
2. **§0.4** — TFA is disabled and TOTP columns dropped; the auth-story agent must fork the template's `repositoryStorage`/`authentication.go` to stop referencing TOTP fields (template assumes they exist).
3. **GIN `entsql.OpClass`** — if the pinned ent version lacks it, hand-edit `000001_init.up.sql` to add the `jsonb_path_ops` GIN index (§2 Image).