//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/importer"
	"github.com/shutterbase/shutterbase/test/harness"
	"github.com/shutterbase/shutterbase/test/pbfixture"
)

// TestImporterE2E runs the S12 importer against the PB fixture into a DEDICATED
// testcontainers Postgres (isolated from the shared seeded stack), then asserts
// the full verification suite passes and a migrated bcrypt user can authenticate
// with its original password.
func TestImporterE2E(t *testing.T) {
	ctx := context.Background()

	// Dedicated PG + S3 so the importer's drop+create does not touch the shared
	// seeded stack the other e2e tests rely on.
	pg, err := harness.StartPostgres(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Container.Terminate(ctx) })

	s3c, err := harness.StartS3(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s3c.Container.Terminate(ctx) })

	conn, err := database.NewConnection(pg.Options) // Schema.Create + GIN guard
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	// Build the PB SQLite fixture and import it.
	pbPath := filepath.Join(t.TempDir(), "pb.db")
	bcryptHash, err := pbfixture.Build(pbPath)
	require.NoError(t, err)
	pb, err := sql.Open("sqlite", "file:"+pbPath+"?mode=ro")
	require.NoError(t, err)
	t.Cleanup(func() { pb.Close() })

	rep, err := importer.Import(ctx, pb, conn.Client)
	require.NoError(t, err)
	assert.Equal(t, 2, rep.Images)
	assert.Equal(t, 2, rep.Users)

	// bcrypt hash preserved verbatim in Postgres.
	alice := conn.Client.User.Query().AllX(ctx)
	var aliceHash string
	for _, u := range alice {
		if u.Username == pbfixture.BcryptUsername {
			aliceHash = u.PasswordHash
		}
	}
	assert.Equal(t, bcryptHash, aliceHash, "bcrypt hash must round-trip into Postgres verbatim")

	// Put the migrated storageIds in the bucket so the S3 HEAD check finds them.
	for _, key := range []string{"ab/storage_img1", "cd/storage_img2"} {
		_, err := s3c.Client.Client.PutObject(ctx, s3c.Options.Bucket, key,
			bytes.NewReader([]byte("x")), 1, minio.PutObjectOptions{})
		require.NoError(t, err)
	}

	// Full verification suite — with a real S3 client this time.
	res, err := importer.Verify(ctx, pb, conn.Client, s3c.Client)
	require.NoError(t, err)
	t.Logf("verification:\n%s", res.String())
	assert.Empty(t, res.CountMismatch, "per-table counts must match")
	assert.Empty(t, res.Orphans, "no FK orphans")
	assert.Equal(t, 0, res.DuplicateAssgn, "no duplicate (image,imageTag) assignments")
	assert.Empty(t, res.TagStatsDiff, "tag-statistics parity (LIKE vs jsonb)")
	assert.Empty(t, res.FilterDiff, "representative gallery-filter parity")
	assert.False(t, res.S3Skipped, "S3 configured -> HEADs run")
	assert.Equal(t, 2, res.S3Checked)
	assert.Empty(t, res.S3Missing, "all migrated storageIds present in S3")
	assert.True(t, res.OK(), "full verification suite must pass")

	// A migrated bcrypt user authenticates against a server on the imported DB.
	srv, err := harness.StartServer(conn, s3c.Client)
	require.NoError(t, err)
	t.Cleanup(srv.Close)

	jar := newClient(t)
	body, _ := json.Marshal(map[string]string{
		"identifier": pbfixture.BcryptUsername,
		"password":   pbfixture.BcryptPassword,
	})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/auth/login", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", srv.URL)
	resp, err := jar.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "migrated bcrypt user must log in with original password")
}
