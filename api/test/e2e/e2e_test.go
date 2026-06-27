//go:build e2e

// Package e2e runs the S1/S2 API e2e tier against a real testcontainers stack
// (Postgres + rustfs|minio). Run with: go test -tags e2e ./test/e2e/...
package e2e

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/test/harness"
)

var (
	stack  *harness.Stack
	server *httptest.Server
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	s, err := harness.Up(ctx, time.Now())
	if err != nil {
		fmt.Fprintf(os.Stderr, "harness up failed: %v\n", err)
		os.Exit(1)
	}
	stack = s
	fmt.Printf("HARNESS_S3_IMPL=%s\n", stack.S3.Impl)

	server, err = harness.StartServer(stack.DB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start server failed: %v\n", err)
		stack.Close(ctx)
		os.Exit(1)
	}

	code := m.Run()

	server.Close()
	stack.Close(ctx)
	os.Exit(code)
}

// S1 e2e: server boots; /health and /version return 200.
func TestHealthAndVersion(t *testing.T) {
	for _, path := range []string{"/api/v1/health", "/api/v1/version"} {
		resp, err := http.Get(server.URL + path)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode, path)
		resp.Body.Close()
	}
}

// S2 e2e: Schema.Create applied on Postgres and seed loaded cleanly; counts match.
func TestSchemaAppliedAndSeedCounts(t *testing.T) {
	ctx := context.Background()
	c := stack.DB.Client

	assert.Equal(t, 5, c.User.Query().CountX(ctx))
	assert.Equal(t, 3, c.Role.Query().CountX(ctx))
	assert.Equal(t, 1, c.Project.Query().CountX(ctx))
	assert.Equal(t, 2, c.Camera.Query().CountX(ctx))
	assert.Equal(t, 3, c.ImageTag.Query().CountX(ctx))
	assert.Equal(t, 2, c.TimeOffset.Query().CountX(ctx))
	assert.Equal(t, 3, c.Image.Query().CountX(ctx))
	assert.Equal(t, 3, c.ProjectAssignment.Query().CountX(ctx))
	assert.Equal(t, 3, c.ImageTagAssignment.Query().CountX(ctx))
}

// S2 e2e: unique constraints reject dups (must run on Postgres).
func TestUniqueConstraints(t *testing.T) {
	ctx := context.Background()
	c := stack.DB.Client
	m := stack.Manifest

	editor := m.Users["projectEditor"]

	// computed_file_name unique: reuse a seeded computed name.
	_, err := c.Image.Create().
		SetFileName("dup.jpg").SetComputedFileName("FSG_0000.jpg").
		SetStorageId("uniquestorage01").SetSize(1).
		SetUserID(editor).SetUploadID(m.Upload).SetProjectID(m.Project).SetCameraID(m.Cameras["fresh"]).
		Save(ctx)
	assert.Error(t, err, "duplicate computedFileName must be rejected")

	// storage_id unique: reuse a seeded storageId.
	_, err = c.Image.Create().
		SetFileName("dup2.jpg").SetComputedFileName("UNIQUE_0001.jpg").
		SetStorageId("seedimg00000000").SetSize(1).
		SetUserID(editor).SetUploadID(m.Upload).SetProjectID(m.Project).SetCameraID(m.Cameras["fresh"]).
		Save(ctx)
	assert.Error(t, err, "duplicate storageId must be rejected")

	// (image_id, image_tag_id) unique: re-link an already-linked default tag.
	firstImage := m.Images[0]
	defaultTag := m.Tags["Default"]
	_, err = c.ImageTagAssignment.Create().
		SetType(imagetagassignment.TypeManual).
		SetImageID(firstImage).SetImageTagID(defaultTag).
		Save(ctx)
	assert.Error(t, err, "duplicate (image,imageTag) must be rejected")

	// (project_id, user_id) unique: re-assign the editor to the project.
	_, err = c.ProjectAssignment.Create().
		SetProjectID(m.Project).SetUserID(editor).SetRoleID(m.Roles["projectViewer"]).
		Save(ctx)
	assert.Error(t, err, "duplicate (project,user) must be rejected")
}

// S2 e2e: jsonb @> containment uses the GIN(jsonb_path_ops) index.
func TestJSONBContainmentUsesGIN(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("pgx", rawDSN(t))
	require.NoError(t, err)
	defer db.Close()

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	// Force the planner to prefer the index for a deterministic plan on small data.
	_, err = conn.ExecContext(ctx, "SET enable_seqscan = off")
	require.NoError(t, err)

	tag := stack.Manifest.Tags["Default"]
	q := fmt.Sprintf(`EXPLAIN SELECT id FROM images WHERE image_tags @> '["%s"]'`, tag)
	rows, err := conn.QueryContext(ctx, q)
	require.NoError(t, err)
	defer rows.Close()

	var plan strings.Builder
	for rows.Next() {
		var line string
		require.NoError(t, rows.Scan(&line))
		plan.WriteString(line + "\n")
	}
	require.NoError(t, rows.Err())

	out := plan.String()
	t.Logf("EXPLAIN:\n%s", out)
	assert.True(t,
		strings.Contains(out, "image_image_tags") || strings.Contains(out, "Bitmap Index Scan"),
		"jsonb @> should use the GIN index, got:\n%s", out)
}

// S2/harness: presigned URLs are signed against the externally-mapped host:port.
func TestPresignUsesMappedEndpoint(t *testing.T) {
	url, err := stack.S3.Client.GetSignedUploadUrl(context.Background(), "ab/test.jpg")
	require.NoError(t, err)
	want := fmt.Sprintf("%s:%d", stack.S3.Options.Endpoint, stack.S3.Options.Port)
	assert.Contains(t, url, want, "presign must target the mapped host:port, not the container hostname")
}

func rawDSN(t *testing.T) string {
	t.Helper()
	o := stack.PG.Options
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		o.Host, o.Port, o.Username, o.Password, o.Database)
}
