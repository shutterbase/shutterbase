// Package importer migrates a legacy PocketBase SQLite database into the new
// Ent/Postgres schema (REWRITE-SPEC §5). It reads PB tables by collection name
// (PB v0.22 names the physical SQLite table after the collection's name), maps
// every column to its Ent field, remaps user PKs to UUIDs, and writes via the
// raw Ent client in FK-safe order. S3 is untouched — storageId/keys are carried
// verbatim.
package importer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/user"
)

// pbTimeLayouts are the timestamp formats PB writes into SQLite (date/datetime
// fields). Empty string means "no value" → leave the Ent field at its default.
var pbTimeLayouts = []string{"2006-01-02 15:04:05.000Z", "2006-01-02 15:04:05Z", time.RFC3339}

// Report counts the rows imported per entity (sanity for the CLI / tests).
type Report struct {
	Roles, Users, Projects, Cameras, TimeOffsets, Uploads int
	ImageTags, ProjectAssignments, Images, Assignments    int
}

// Import runs the full PB→Ent migration against an already-open PB SQLite DB and
// an already-open (freshly migrated, empty) Ent client. Re-runnable: the caller
// is expected to have dropped+recreated the schema first.
func Import(ctx context.Context, pb *sql.DB, client *ent.Client) (*Report, error) {
	rep := &Report{}

	// 1. roles ---------------------------------------------------------------
	roleKeyByID := map[string]string{} // pb role id -> key (for users.role)
	roles, err := scanRows(pb, `SELECT id, created, updated, key, description FROM roles`)
	if err != nil {
		return nil, fmt.Errorf("read roles: %w", err)
	}
	for _, r := range roles {
		id := str(r, "id")
		key := str(r, "key")
		roleKeyByID[id] = key
		b := client.Role.Create().SetID(id).SetKey(key).SetDescription(str(r, "description"))
		applyTimes(b.SetCreatedAt, b.SetUpdatedAt, r)
		if _, err := b.Save(ctx); err != nil {
			return nil, fmt.Errorf("create role %s: %w", id, err)
		}
		rep.Roles++
	}

	// 2. users (activeProject deferred) --------------------------------------
	userMap := map[string]uuid.UUID{}        // pb user id -> new uuid
	userActiveProject := map[string]string{} // pb user id -> pb project id
	users, err := scanRows(pb, `SELECT id, created, updated, username, email, verified, passwordHash, firstName, lastName, copyrightTag, active, role, activeProject FROM users`)
	if err != nil {
		return nil, fmt.Errorf("read users: %w", err)
	}
	for _, u := range users {
		pbID := str(u, "id")
		newID := uuid.New()
		userMap[pbID] = newID
		if ap := str(u, "activeProject"); ap != "" {
			userActiveProject[pbID] = ap
		}

		role := user.RoleUser
		if roleKeyByID[str(u, "role")] == "admin" {
			role = user.RoleAdmin
		}

		b := client.User.Create().
			SetID(newID).
			SetLegacyId(pbID).
			SetUsername(str(u, "username")).
			SetFirstName(str(u, "firstName")).
			SetLastName(str(u, "lastName")).
			SetActive(toBool(u["active"])).
			SetVerified(toBool(u["verified"])).
			SetForcePasswordChange(false).
			SetProvider(user.ProviderLocal).
			SetRole(role).
			SetCreatedBy(newID).SetUpdatedBy(newID) // self
		if v := str(u, "email"); v != "" {
			b.SetEmail(v)
		}
		if v := str(u, "copyrightTag"); v != "" {
			b.SetCopyrightTag(v)
		}
		if v := str(u, "passwordHash"); v != "" {
			b.SetPasswordHash(v) // BCRYPT verbatim — BcryptVerifier rehashes on login
		}
		applyTimes(b.SetCreatedAt, b.SetUpdatedAt, u)
		if _, err := b.Save(ctx); err != nil {
			return nil, fmt.Errorf("create user %s: %w", pbID, err)
		}
		rep.Users++
	}

	// 3. projects ------------------------------------------------------------
	projects, err := scanRows(pb, `SELECT id, created, updated, name, description, copyright, copyrightReference, locationName, locationCode, locationCity, aiSystemMessage FROM projects`)
	if err != nil {
		return nil, fmt.Errorf("read projects: %w", err)
	}
	for _, p := range projects {
		b := client.Project.Create().
			SetID(str(p, "id")).
			SetName(str(p, "name")).
			SetDescription(str(p, "description")).
			SetCopyright(str(p, "copyright")).
			SetCopyrightReference(str(p, "copyrightReference")).
			SetLocationName(str(p, "locationName")).
			SetLocationCode(str(p, "locationCode")).
			SetLocationCity(str(p, "locationCity"))
		if v := str(p, "aiSystemMessage"); v != "" {
			b.SetAiSystemMessage(v)
		}
		applyTimes(b.SetCreatedAt, b.SetUpdatedAt, p)
		if _, err := b.Save(ctx); err != nil {
			return nil, fmt.Errorf("create project %s: %w", str(p, "id"), err)
		}
		rep.Projects++
	}

	// 4. cameras -------------------------------------------------------------
	cameras, err := scanRows(pb, `SELECT id, created, updated, name, user FROM cameras`)
	if err != nil {
		return nil, fmt.Errorf("read cameras: %w", err)
	}
	for _, c := range cameras {
		owner, ok := userMap[str(c, "user")]
		if !ok {
			return nil, fmt.Errorf("camera %s references unknown user %s", str(c, "id"), str(c, "user"))
		}
		b := client.Camera.Create().
			SetID(str(c, "id")).
			SetName(str(c, "name")).
			SetUserID(owner).
			SetCreatedBy(owner).SetUpdatedBy(owner)
		applyTimes(b.SetCreatedAt, b.SetUpdatedAt, c)
		if _, err := b.Save(ctx); err != nil {
			return nil, fmt.Errorf("create camera %s: %w", str(c, "id"), err)
		}
		rep.Cameras++
	}

	// 5. time_offsets --------------------------------------------------------
	offsets, err := scanRows(pb, `SELECT id, created, updated, serverTime, cameraTime, timeOffset, camera FROM time_offsets`)
	if err != nil {
		return nil, fmt.Errorf("read time_offsets: %w", err)
	}
	for _, o := range offsets {
		st, _ := parseTime(str(o, "serverTime"))
		ct, _ := parseTime(str(o, "cameraTime"))
		b := client.TimeOffset.Create().
			SetID(str(o, "id")).
			SetServerTime(st).
			SetCameraTime(ct).
			SetCameraID(str(o, "camera"))
		if v, ok := optInt(o["timeOffset"]); ok {
			b.SetTimeOffset(v)
		}
		applyTimes(b.SetCreatedAt, b.SetUpdatedAt, o)
		if _, err := b.Save(ctx); err != nil {
			return nil, fmt.Errorf("create time_offset %s: %w", str(o, "id"), err)
		}
		rep.TimeOffsets++
	}

	// 6. uploads (table renamed batches->uploads) ----------------------------
	uploads, err := scanRows(pb, `SELECT id, created, updated, name, project, user, camera FROM uploads`)
	if err != nil {
		return nil, fmt.Errorf("read uploads: %w", err)
	}
	for _, up := range uploads {
		owner, ok := userMap[str(up, "user")]
		if !ok {
			return nil, fmt.Errorf("upload %s references unknown user %s", str(up, "id"), str(up, "user"))
		}
		b := client.Upload.Create().
			SetID(str(up, "id")).
			SetName(str(up, "name")).
			SetProjectID(str(up, "project")).
			SetUserID(owner).
			SetCameraID(str(up, "camera")).
			SetCreatedBy(owner).SetUpdatedBy(owner)
		applyTimes(b.SetCreatedAt, b.SetUpdatedAt, up)
		if _, err := b.Save(ctx); err != nil {
			return nil, fmt.Errorf("create upload %s: %w", str(up, "id"), err)
		}
		rep.Uploads++
	}

	// 7. image_tags ----------------------------------------------------------
	tags, err := scanRows(pb, `SELECT id, created, updated, name, description, isAlbum, type, project FROM image_tags`)
	if err != nil {
		return nil, fmt.Errorf("read image_tags: %w", err)
	}
	for _, t := range tags {
		b := client.ImageTag.Create().
			SetID(str(t, "id")).
			SetName(str(t, "name")).
			SetDescription(str(t, "description")).
			SetIsAlbum(toBool(t["isAlbum"])).
			SetType(imagetag.Type(str(t, "type"))).
			SetProjectID(str(t, "project"))
		applyTimes(b.SetCreatedAt, b.SetUpdatedAt, t)
		if _, err := b.Save(ctx); err != nil {
			return nil, fmt.Errorf("create image_tag %s: %w", str(t, "id"), err)
		}
		rep.ImageTags++
	}

	// 8. project_assignments -------------------------------------------------
	pas, err := scanRows(pb, `SELECT id, created, updated, project, user, role FROM project_assignments`)
	if err != nil {
		return nil, fmt.Errorf("read project_assignments: %w", err)
	}
	for _, pa := range pas {
		owner, ok := userMap[str(pa, "user")]
		if !ok {
			return nil, fmt.Errorf("project_assignment %s references unknown user %s", str(pa, "id"), str(pa, "user"))
		}
		b := client.ProjectAssignment.Create().
			SetID(str(pa, "id")).
			SetProjectID(str(pa, "project")).
			SetUserID(owner).
			SetRoleID(str(pa, "role")).
			SetCreatedBy(owner).SetUpdatedBy(owner)
		applyTimes(b.SetCreatedAt, b.SetUpdatedAt, pa)
		if _, err := b.Save(ctx); err != nil {
			return nil, fmt.Errorf("create project_assignment %s: %w", str(pa, "id"), err)
		}
		rep.ProjectAssignments++
	}

	// 9. images --------------------------------------------------------------
	imageUser := map[string]string{} // pb image id -> pb user id (for assignment authorship)
	images, err := scanRows(pb, `SELECT id, created, updated, fileName, computedFileName, exifData, capturedAt, capturedAtCorrected, size, width, height, storageId, user, upload, project, camera, imageTags FROM images`)
	if err != nil {
		return nil, fmt.Errorf("read images: %w", err)
	}
	for _, im := range images {
		pbUser := str(im, "user")
		imageUser[str(im, "id")] = pbUser
		owner, ok := userMap[pbUser]
		if !ok {
			return nil, fmt.Errorf("image %s references unknown user %s", str(im, "id"), pbUser)
		}
		b := client.Image.Create().
			SetID(str(im, "id")).
			SetFileName(str(im, "fileName")).
			SetStorageId(str(im, "storageId")).
			SetSize(mustInt(im["size"])).
			SetUserID(owner).
			SetUploadID(str(im, "upload")).
			SetProjectID(str(im, "project")).
			SetCameraID(str(im, "camera")).
			SetImageTags(parseStringArray(str(im, "imageTags"))).
			SetCreatedBy(owner).SetUpdatedBy(owner)
		if v := str(im, "computedFileName"); v != "" {
			b.SetComputedFileName(v)
		}
		if exif := parseJSONObject(str(im, "exifData")); exif != nil {
			b.SetExifData(exif)
		}
		if t, ok := parseTime(str(im, "capturedAt")); ok {
			b.SetCapturedAt(t)
		}
		if t, ok := parseTime(str(im, "capturedAtCorrected")); ok {
			b.SetCapturedAtCorrected(t)
		}
		if v, ok := optInt(im["width"]); ok {
			b.SetWidth(v)
		}
		if v, ok := optInt(im["height"]); ok {
			b.SetHeight(v)
		}
		applyTimes(b.SetCreatedAt, b.SetUpdatedAt, im)
		if _, err := b.Save(ctx); err != nil {
			return nil, fmt.Errorf("create image %s: %w", str(im, "id"), err)
		}
		rep.Images++
	}

	// 10. image_tag_assignments (coalesce empty type -> manual) ---------------
	assigns, err := scanRows(pb, `SELECT id, created, updated, type, imageTag, image FROM image_tag_assignments`)
	if err != nil {
		return nil, fmt.Errorf("read image_tag_assignments: %w", err)
	}
	for _, a := range assigns {
		typ := str(a, "type")
		if typ == "" {
			typ = "manual" // §0.8: legacy untyped rows default to manual
		}
		b := client.ImageTagAssignment.Create().
			SetID(str(a, "id")).
			SetType(imagetagassignment.Type(typ)).
			SetImageID(str(a, "image")).
			SetImageTagID(str(a, "imageTag"))
		if owner, ok := userMap[imageUser[str(a, "image")]]; ok {
			b.SetCreatedBy(owner).SetUpdatedBy(owner)
		}
		applyTimes(b.SetCreatedAt, b.SetUpdatedAt, a)
		if _, err := b.Save(ctx); err != nil {
			return nil, fmt.Errorf("create image_tag_assignment %s: %w", str(a, "id"), err)
		}
		rep.Assignments++
	}

	// 11. PATCH users.activeProject -----------------------------------------
	for pbUserID, pbProjectID := range userActiveProject {
		if _, err := client.User.UpdateOneID(userMap[pbUserID]).SetActiveProjectID(pbProjectID).Save(ctx); err != nil {
			return nil, fmt.Errorf("patch activeProject for user %s: %w", pbUserID, err)
		}
	}

	return rep, nil
}

// --- PB row reading helpers -------------------------------------------------

// scanRows runs a query and returns each row as a column->value map. []byte
// values (text/blob) are normalized to string; INTEGER/REAL stay int64/float64.
func scanRows(db *sql.DB, query string) ([]map[string]any, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		m := make(map[string]any, len(cols))
		for i, c := range cols {
			v := vals[i]
			if b, ok := v.([]byte); ok {
				v = string(b)
			}
			m[c] = v
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func str(m map[string]any, k string) string {
	switch v := m[k].(type) {
	case nil:
		return ""
	case string:
		return v
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return fmt.Sprint(v)
	}
}

func toBool(v any) bool {
	switch t := v.(type) {
	case int64:
		return t != 0
	case bool:
		return t
	case string:
		return t == "1" || t == "true"
	default:
		return false
	}
}

// optInt returns (value, true) when v carries a real number; (0,false) for
// nil/empty (PB optional numeric not set).
func optInt(v any) (int, bool) {
	switch t := v.(type) {
	case nil:
		return 0, false
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	case string:
		if t == "" {
			return 0, false
		}
		n, err := strconv.Atoi(t)
		return n, err == nil
	default:
		return 0, false
	}
}

func mustInt(v any) int {
	n, _ := optInt(v)
	return n
}

func parseTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range pbTimeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// parseStringArray decodes a PB multi-relation JSON array ("[\"a\",\"b\"]").
// Empty/invalid → empty slice (never nil, so the jsonb column is [] not null).
func parseStringArray(s string) []string {
	if s == "" {
		return []string{}
	}
	var arr []string
	if err := json.Unmarshal([]byte(s), &arr); err != nil {
		return []string{}
	}
	return arr
}

func parseJSONObject(s string) map[string]any {
	if s == "" || s == "null" {
		return nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		return nil
	}
	return obj
}

// applyTimes sets createdAt/updatedAt from PB created/updated when present. R is
// the entity's create-builder type, inferred from the passed method values.
func applyTimes[R any](setCreated, setUpdated func(time.Time) R, m map[string]any) {
	if t, ok := parseTime(str(m, "created")); ok {
		setCreated(t)
	}
	if t, ok := parseTime(str(m, "updated")); ok {
		setUpdated(t)
	}
}
