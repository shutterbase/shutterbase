// White-box tests (package service) so they can drive the unexported step() and
// inspect the FIFO queue directly — fast and deterministic, no goroutine timing.
// They run on the seeded SQLite repo; presigning is offline so a dummy S3 client
// produces a URL without any network.
package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/seed"
	"github.com/shutterbase/shutterbase/internal/util"
)

func newSvc(t *testing.T, inference ImageInference) (*AIService, *seed.Manifest) {
	t.Helper()
	conn, err := database.NewConnection(&database.Options{DatabaseType: "sqlite", File: t.TempDir() + "/svc.db"})
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	repo, err := repository.NewRepository(&repository.Options{DatabaseConnection: conn})
	require.NoError(t, err)
	m, err := seed.Seed(context.Background(), repo.Client, time.Now())
	require.NoError(t, err)

	// offline URL builder: echoes the object name so recordInference can read the
	// storage id back out (real s3 presigning is exercised in the e2e tier).
	fakeURL := func(_ context.Context, objectName string) (string, error) {
		return "https://example.test/" + objectName, nil
	}

	svc := &AIService{repo: repo, inference: inference, timeout: 5 * time.Second, downloadURL: fakeURL}
	return svc, m
}

func inferredCount(t *testing.T, svc *AIService, imageID, tagID string) int {
	t.Helper()
	return svc.repo.Client.ImageTagAssignment.Query().
		Where(imagetagassignment.ImageID(imageID), imagetagassignment.ImageTagID(tagID),
			imagetagassignment.TypeEQ(imagetagassignment.TypeInferred)).
		CountX(context.Background())
}

// recordInference records, in order, the storage-id segment of each image URL it
// is asked to infer — so FIFO order is observable without timing.
type recordInference struct {
	tags []string
	seen []string
}

func (r *recordInference) Infer(_ context.Context, imageURL, _ string) ([]string, error) {
	// object name looks like "se/seedimg00000001-512.jpg"; capture the storage id.
	for _, part := range strings.Split(imageURL, "/") {
		if strings.HasPrefix(part, "seedimg") {
			r.seen = append(r.seen, strings.SplitN(part, "-", 2)[0])
		}
	}
	return r.tags, nil
}

type failInference struct{}

func (failInference) Infer(_ context.Context, _ string, _ string) ([]string, error) {
	return nil, errors.New("boom")
}

// Provider selection by config.
func TestNewInferenceProviderSelection(t *testing.T) {
	t.Setenv("SESSION_SECRET_KEY", "x")
	cases := map[string]any{
		"stub":       &StubInference{},
		"":           &StubInference{},
		"openai":     &openAIInference{},
		"openrouter": &openAIInference{},
		"http":       &HTTPInference{},
	}
	for provider, want := range cases {
		t.Setenv("AI_PROVIDER", provider)
		require.NoError(t, util.InitConfig())
		got, err := NewInference()
		require.NoError(t, err, provider)
		assert.IsType(t, want, got, "provider %q", provider)
	}

	t.Setenv("AI_PROVIDER", "bogus")
	require.NoError(t, util.InitConfig())
	_, err := NewInference()
	assert.Error(t, err, "unknown provider must error")
}

// FIFO: images drain front-to-back in enqueue order.
func TestFIFOOrder(t *testing.T) {
	rec := &recordInference{tags: []string{"none"}}
	svc, m := newSvc(t, rec)
	ctx := context.Background()

	for _, id := range m.Images {
		svc.Enqueue(id)
	}
	for svc.step(ctx) { // drain
	}

	require.Len(t, rec.seen, len(m.Images))
	assert.Equal(t, []string{"seedimg00000000", "seedimg00000001", "seedimg00000002"}, rec.seen)
}

// A returned tag name matching a project tag -> a single "inferred" assignment,
// and inferredAt gets stamped. Re-running is idempotent (still one row).
func TestMatchingTagInferredAndIdempotent(t *testing.T) {
	svc, m := newSvc(t, &StubInference{Tags: []string{"Podium"}})
	ctx := context.Background()
	img := m.Images[0]
	podium := m.Tags["Podium"]

	require.NoError(t, svc.process(ctx, img))
	assert.Equal(t, 1, inferredCount(t, svc, img, podium))

	got, err := svc.repo.GetImage(ctx, img)
	require.NoError(t, err)
	require.NotNil(t, got.InferredAt, "inferredAt must be stamped")

	// re-run: idempotent on (image, imageTag) -> still exactly one inferred row.
	require.NoError(t, svc.process(ctx, img))
	assert.Equal(t, 1, inferredCount(t, svc, img, podium))
}

// A "none" result (and any non-matching name) links nothing.
func TestNoneResultLinksNothing(t *testing.T) {
	svc, m := newSvc(t, &StubInference{Tags: []string{"none"}})
	ctx := context.Background()
	img := m.Images[0]

	before := svc.repo.Client.ImageTagAssignment.Query().CountX(ctx)
	require.NoError(t, svc.process(ctx, img))
	after := svc.repo.Client.ImageTagAssignment.Query().CountX(ctx)
	assert.Equal(t, before, after, "no assignment created for 'none'")
}

// Empty aiSystemMessage -> skip: no inference, no assignment, no inferredAt.
func TestEmptySystemMessageSkips(t *testing.T) {
	svc, m := newSvc(t, &StubInference{Tags: []string{"Podium"}})
	ctx := context.Background()
	img := m.Images[0]

	_, err := svc.repo.UpdateProject(ctx, m.Project, &repository.UpdateProjectParameters{
		AiSystemMessage: util.StringPointer(""),
	})
	require.NoError(t, err)

	before := svc.repo.Client.ImageTagAssignment.Query().CountX(ctx)
	require.NoError(t, svc.process(ctx, img))
	after := svc.repo.Client.ImageTagAssignment.Query().CountX(ctx)
	assert.Equal(t, before, after, "empty aiSystemMessage must skip inference")

	got, err := svc.repo.GetImage(ctx, img)
	require.NoError(t, err)
	assert.Nil(t, got.InferredAt, "skip must not stamp inferredAt")
}

// 30s backoff on error keeps the front item (no drop) and arms the backoff.
func TestBackoffKeepsItem(t *testing.T) {
	svc, m := newSvc(t, failInference{})
	ctx := context.Background()
	img := m.Images[0]
	svc.Enqueue(img)

	handled := svc.step(ctx)
	assert.True(t, handled, "an item was present")
	require.Len(t, svc.queue, 1, "errored item must NOT be dropped")
	assert.Equal(t, img, svc.queue[0], "same item stays at the front")
	assert.True(t, time.Now().Before(svc.backoffUntil), "backoff must be armed")
	assert.WithinDuration(t, time.Now().Add(aiBackoffDuration), svc.backoffUntil, time.Second)
}
