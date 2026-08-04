//go:build e2e

// S6 e2e: drive the real AIService (FIFO queue + 30s backoff goroutine) against
// the testcontainers stack — Postgres for the repo, rustfs|minio for the real
// presigned-URL path — with StubInference returning a known project tag. Assert
// the "inferred" assignment row lands in Postgres and inferredAt is stamped.
package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/internal/service"
	"github.com/shutterbase/shutterbase/internal/util"
)

func TestAIServiceInferredAssignment(t *testing.T) {
	ctx := context.Background()
	os.Setenv("SESSION_SECRET_KEY", "e2e")
	require.NoError(t, util.InitConfig())

	r := repo(t)
	m := stack.Manifest
	img := m.Images[1] // distinct image so other suites' baselines stay intact
	podium := m.Tags["Podium"]

	svc := service.NewAIService(r, stack.S3.Client, &service.StubInference{Tags: []string{"Podium"}})
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	svc.Start(runCtx)
	svc.Enqueue(img)

	require.Eventually(t, func() bool {
		return stack.DB.Client.ImageTagAssignment.Query().Where(
			imagetagassignment.ImageID(img),
			imagetagassignment.ImageTagID(podium),
			imagetagassignment.TypeEQ(imagetagassignment.TypeInferred),
		).CountX(ctx) == 1
	}, 15*time.Second, 200*time.Millisecond, "inferred assignment row must appear on Postgres")

	got, err := r.GetImage(ctx, img)
	require.NoError(t, err)
	assert.NotNil(t, got.InferredAt, "inferredAt must be stamped after inference")

	// cleanup so the seed baseline (3 assignments) is restored for sibling tests.
	a := stack.DB.Client.ImageTagAssignment.Query().
		Where(imagetagassignment.ImageID(img), imagetagassignment.ImageTagID(podium)).
		OnlyX(ctx)
	require.NoError(t, r.DeleteImageTagAssignment(ctx, a.ID))
}
