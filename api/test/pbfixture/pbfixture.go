// Package pbfixture hand-crafts a small but representative PocketBase v0.22
// SQLite database matching the real PB physical schema (tables named after the
// collection name; columns named after each field's `name`). It is shared by the
// importer unit tests and the API e2e tier so both exercise the same data.
//
// Coverage: 2 users (one with a bcrypt passwordHash + an activeProject), a
// project, 2 cameras, time_offsets, uploads, image_tags (incl. a 'template'),
// project_assignments, images (exifData jsonb, imageTags array, the easily-missed
// numeric fields size/width/height), image_tag_assignments (incl. one empty type),
// and an 'inferences' row that must be dropped.
package pbfixture

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// Known credential for the migrated bcrypt user — the e2e auth check logs in
// with exactly this username/password.
const (
	BcryptUsername = "alice"
	BcryptPassword = "AlicePass123"

	AdminRoleID   = "role_admin00000"
	UserRoleID    = "role_user000000"
	AliceUserID   = "user_alice00000"
	BobUserID     = "user_bob0000000"
	ProjectID     = "proj00000000001"
	CameraAID     = "cam0000000000a1"
	CameraBID     = "cam0000000000b2"
	TagDefaultID  = "tag_default0001"
	TagManualID   = "tag_manual00002"
	TagTemplateID = "tag_template003"
	Image1ID      = "img00000000001"
	Image2ID      = "img00000000002"
)

// Build creates the PB SQLite fixture at path and returns the bcrypt hash that
// was stored for the alice user (so tests can assert verbatim preservation).
func Build(path string) (bcryptHash string, err error) {
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		return "", err
	}
	defer db.Close()

	schema := []string{
		`CREATE TABLE roles (id TEXT PRIMARY KEY, created TEXT, updated TEXT, key TEXT, description TEXT)`,
		`CREATE TABLE users (id TEXT PRIMARY KEY, created TEXT, updated TEXT, username TEXT, email TEXT,
			emailVisibility INTEGER, verified INTEGER, passwordHash TEXT, tokenKey TEXT, lastResetSentAt TEXT,
			firstName TEXT, lastName TEXT, copyrightTag TEXT, active INTEGER, role TEXT, activeProject TEXT, avatar TEXT)`,
		`CREATE TABLE projects (id TEXT PRIMARY KEY, created TEXT, updated TEXT, name TEXT, description TEXT,
			copyright TEXT, copyrightReference TEXT, locationName TEXT, locationCode TEXT, locationCity TEXT, aiSystemMessage TEXT)`,
		`CREATE TABLE cameras (id TEXT PRIMARY KEY, created TEXT, updated TEXT, name TEXT, user TEXT)`,
		`CREATE TABLE time_offsets (id TEXT PRIMARY KEY, created TEXT, updated TEXT, serverTime TEXT, cameraTime TEXT, timeOffset INTEGER, camera TEXT)`,
		`CREATE TABLE uploads (id TEXT PRIMARY KEY, created TEXT, updated TEXT, name TEXT, project TEXT, user TEXT, camera TEXT)`,
		`CREATE TABLE image_tags (id TEXT PRIMARY KEY, created TEXT, updated TEXT, name TEXT, description TEXT, isAlbum INTEGER, type TEXT, project TEXT)`,
		`CREATE TABLE project_assignments (id TEXT PRIMARY KEY, created TEXT, updated TEXT, project TEXT, user TEXT, role TEXT)`,
		`CREATE TABLE images (id TEXT PRIMARY KEY, created TEXT, updated TEXT, fileName TEXT, computedFileName TEXT,
			exifData TEXT, capturedAt TEXT, capturedAtCorrected TEXT, size INTEGER, width INTEGER, height INTEGER,
			storageId TEXT, user TEXT, upload TEXT, project TEXT, camera TEXT, imageTags TEXT, downloadUrls TEXT)`,
		`CREATE TABLE image_tag_assignments (id TEXT PRIMARY KEY, created TEXT, updated TEXT, type TEXT, imageTag TEXT, image TEXT)`,
		`CREATE TABLE inferences (id TEXT PRIMARY KEY, created TEXT, updated TEXT, image TEXT, result TEXT)`,
	}
	for _, s := range schema {
		if _, err := db.Exec(s); err != nil {
			return "", fmt.Errorf("create table: %w", err)
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(BcryptPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	bcryptHash = string(hash)

	const t0 = "2024-05-20 10:00:00.000Z"
	const t1 = "2024-05-21 12:30:00.123Z"

	exec := func(q string, args ...any) error {
		_, e := db.Exec(q, args...)
		return e
	}

	// roles
	if err := exec(`INSERT INTO roles (id,created,updated,key,description) VALUES (?,?,?,?,?),(?,?,?,?,?)`,
		AdminRoleID, t0, t0, "admin", "Global admin",
		UserRoleID, t0, t0, "user", "Regular user"); err != nil {
		return "", err
	}

	// users: alice (bcrypt, admin, activeProject set), bob (no password, user role)
	if err := exec(`INSERT INTO users
		(id,created,updated,username,email,emailVisibility,verified,passwordHash,tokenKey,firstName,lastName,copyrightTag,active,role,activeProject,avatar)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		AliceUserID, t0, t1, BcryptUsername, "alice@example.com", 0, 1, bcryptHash, "tok_alice", "Alice", "Anderson", "AA", 1, AdminRoleID, ProjectID, "avatar_a.png"); err != nil {
		return "", err
	}
	if err := exec(`INSERT INTO users
		(id,created,updated,username,email,emailVisibility,verified,passwordHash,tokenKey,firstName,lastName,copyrightTag,active,role,activeProject,avatar)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		BobUserID, t0, t0, "bob", "bob@example.com", 0, 0, "", "tok_bob", "Bob", "Brown", "", 1, UserRoleID, "", ""); err != nil {
		return "", err
	}

	// project
	if err := exec(`INSERT INTO projects
		(id,created,updated,name,description,copyright,copyrightReference,locationName,locationCode,locationCity,aiSystemMessage)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		ProjectID, t0, t0, "Demo Project", "A demo", "ACME", "REF-1", "Stadium", "STD", "Berlin", "describe the photo"); err != nil {
		return "", err
	}

	// cameras (alice + bob)
	if err := exec(`INSERT INTO cameras (id,created,updated,name,user) VALUES (?,?,?,?,?),(?,?,?,?,?)`,
		CameraAID, t0, t0, "Alice Cam", AliceUserID,
		CameraBID, t0, t0, "Bob Cam", BobUserID); err != nil {
		return "", err
	}

	// time_offsets (one with timeOffset set, one without)
	if err := exec(`INSERT INTO time_offsets (id,created,updated,serverTime,cameraTime,timeOffset,camera) VALUES (?,?,?,?,?,?,?),(?,?,?,?,?,?,?)`,
		"toff0000000001", t0, t0, t0, "2024-05-20 09:59:23.000Z", 37, CameraAID,
		"toff0000000002", t0, t0, t0, t0, nil, CameraBID); err != nil {
		return "", err
	}

	// uploads
	if err := exec(`INSERT INTO uploads (id,created,updated,name,project,user,camera) VALUES (?,?,?,?,?,?,?)`,
		"upload00000001", t0, t0, "Batch 1", ProjectID, AliceUserID, CameraAID); err != nil {
		return "", err
	}

	// image_tags (default, manual, template)
	if err := exec(`INSERT INTO image_tags (id,created,updated,name,description,isAlbum,type,project) VALUES (?,?,?,?,?,?,?,?),(?,?,?,?,?,?,?,?),(?,?,?,?,?,?,?,?)`,
		TagDefaultID, t0, t0, "Default", "default tag", 0, "default", ProjectID,
		TagManualID, t0, t0, "Sunset", "manual tag", 0, "manual", ProjectID,
		TagTemplateID, t0, t0, "Album-A", "template tag", 1, "template", ProjectID); err != nil {
		return "", err
	}

	// project_assignments (alice admin, bob user)
	if err := exec(`INSERT INTO project_assignments (id,created,updated,project,user,role) VALUES (?,?,?,?,?,?),(?,?,?,?,?,?)`,
		"pa000000000001", t0, t0, ProjectID, AliceUserID, AdminRoleID,
		"pa000000000002", t0, t0, ProjectID, BobUserID, UserRoleID); err != nil {
		return "", err
	}

	// images: img1 has 2 tags + width/height + exif + capturedAt; img2 has 1 tag, no width/height
	exif := `{"Make":"Canon","Model":"EOS","ISO":400}`
	if err := exec(`INSERT INTO images
		(id,created,updated,fileName,computedFileName,exifData,capturedAt,capturedAtCorrected,size,width,height,storageId,user,upload,project,camera,imageTags,downloadUrls)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		Image1ID, t0, t1, "IMG_0001.JPG", "DEMO_0001.jpg", exif, t0, "2024-05-20 10:00:37.000Z", 1048576, 6000, 4000,
		"ab/storage_img1", AliceUserID, "upload00000001", ProjectID, CameraAID,
		fmt.Sprintf(`["%s","%s"]`, TagDefaultID, TagManualID), `["http://old/download/1"]`); err != nil {
		return "", err
	}
	if err := exec(`INSERT INTO images
		(id,created,updated,fileName,computedFileName,exifData,capturedAt,capturedAtCorrected,size,width,height,storageId,user,upload,project,camera,imageTags,downloadUrls)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		Image2ID, t0, t0, "IMG_0002.JPG", "DEMO_0002.jpg", "", "", "", 2097152, nil, nil,
		"cd/storage_img2", BobUserID, "upload00000001", ProjectID, CameraBID,
		fmt.Sprintf(`["%s"]`, TagDefaultID), ""); err != nil {
		return "", err
	}

	// image_tag_assignments: typed + one EMPTY type (must coalesce to manual)
	if err := exec(`INSERT INTO image_tag_assignments (id,created,updated,type,imageTag,image) VALUES (?,?,?,?,?,?),(?,?,?,?,?,?),(?,?,?,?,?,?)`,
		"ita00000000001", t0, t0, "default", TagDefaultID, Image1ID,
		"ita00000000002", t0, t0, "", TagManualID, Image1ID, // empty type -> manual
		"ita00000000003", t0, t0, "default", TagDefaultID, Image2ID); err != nil {
		return "", err
	}

	// inferences: must be dropped by the importer
	if err := exec(`INSERT INTO inferences (id,created,updated,image,result) VALUES (?,?,?,?,?)`,
		"inf00000000001", t0, t0, Image1ID, `{"label":"cat"}`); err != nil {
		return "", err
	}

	return bcryptHash, nil
}
