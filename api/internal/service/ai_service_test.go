// White-box tests (package service) so they can drive the unexported step() and
// inspect the DB queue state directly — fast and deterministic, no goroutine
// timing. They run on the seeded SQLite repo; presigning is offline so a dummy
// S3 client produces a URL without any network.
package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	entimage "github.com/shutterbase/shutterbase/ent/image"
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

	svc := &AIService{
		repo: repo, inference: inference, timeout: 5 * time.Second,
		downloadURL: fakeURL, wake: make(chan struct{}, 1),
	}
	return svc, m
}

func inferredCount(t *testing.T, svc *AIService, imageID, tagID string) int {
	t.Helper()
	return svc.repo.Client.ImageTagAssignment.Query().
		Where(imagetagassignment.ImageID(imageID), imagetagassignment.ImageTagID(tagID),
			imagetagassignment.TypeEQ(imagetagassignment.TypeInferred)).
		CountX(context.Background())
}

func aiStatus(t *testing.T, svc *AIService, imageID string) string {
	t.Helper()
	img := svc.repo.Client.Image.GetX(context.Background(), imageID)
	if img.AiStatus == nil {
		return ""
	}
	return string(*img.AiStatus)
}

// recordInference records, in order, the storage-id segment of each image URL it
// is asked to infer — so FIFO order is observable without timing.
type recordInference struct {
	tags []string
	seen []string
}

func (r *recordInference) Infer(_ context.Context, req InferenceRequest) ([]InferredTag, error) {
	// object name looks like "se/seedimg00000001-512.jpg"; capture the storage id.
	for _, part := range strings.Split(req.ImageURL, "/") {
		if strings.HasPrefix(part, "seedimg") {
			r.seen = append(r.seen, strings.SplitN(part, "-", 2)[0])
		}
	}
	tags := make([]InferredTag, 0, len(r.tags))
	for _, name := range r.tags {
		tags = append(tags, InferredTag{Name: name, Confidence: 1})
	}
	return tags, nil
}

type failInference struct{}

func (failInference) Infer(_ context.Context, _ InferenceRequest) ([]InferredTag, error) {
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

// FIFO: images drain oldest-aiQueuedAt-first in enqueue order, and land done.
func TestFIFOOrder(t *testing.T) {
	rec := &recordInference{tags: []string{"none"}}
	svc, m := newSvc(t, rec)
	ctx := context.Background()

	// distinct aiQueuedAt values: SQLite timestamps are coarse, so set them
	// explicitly instead of relying on Enqueue's time.Now() spacing.
	base := time.Now().Add(-time.Minute)
	for i, id := range m.Images {
		svc.Enqueue(id)
		svc.repo.Client.Image.UpdateOneID(id).SetAiQueuedAt(base.Add(time.Duration(i) * time.Second)).SaveX(ctx)
	}
	for svc.step(ctx) { // drain
	}

	require.Len(t, rec.seen, len(m.Images))
	assert.Equal(t, []string{"seedimg00000000", "seedimg00000001", "seedimg00000002"}, rec.seen)
	for _, id := range m.Images {
		assert.Equal(t, "done", aiStatus(t, svc, id))
	}
}

// A returned tag name matching a project tag -> a single "inferred" assignment,
// inferredAt stamped, aiStatus done. Re-running replaces (still one row).
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
	assert.Equal(t, "done", aiStatus(t, svc, img))

	// re-run: replace semantics -> still exactly one inferred row.
	require.NoError(t, svc.process(ctx, img))
	assert.Equal(t, 1, inferredCount(t, svc, img, podium))
}

// A rerun whose fresh result no longer contains a tag drops the stale
// assignment (replace, not accumulate).
func TestRerunReplacesStaleInferred(t *testing.T) {
	svc, m := newSvc(t, &StubInference{Tags: []string{"Podium"}})
	ctx := context.Background()
	img := m.Images[0]
	podium := m.Tags["Podium"]

	require.NoError(t, svc.process(ctx, img))
	require.Equal(t, 1, inferredCount(t, svc, img, podium))

	svc.inference = &StubInference{Tags: []string{"none"}}
	require.NoError(t, svc.process(ctx, img))
	assert.Equal(t, 0, inferredCount(t, svc, img, podium), "stale inferred assignment must be replaced")
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

// Empty aiSystemMessage -> skip: no inference, no assignment, queue state
// cleared (null, not eternally pending).
func TestEmptySystemMessageSkips(t *testing.T) {
	svc, m := newSvc(t, &StubInference{Tags: []string{"Podium"}})
	ctx := context.Background()
	img := m.Images[0]

	_, err := svc.repo.UpdateProject(ctx, m.Project, &repository.UpdateProjectParameters{
		AiSystemMessage: util.StringPointer(""),
	})
	require.NoError(t, err)
	svc.Enqueue(img)

	before := svc.repo.Client.ImageTagAssignment.Query().CountX(ctx)
	assert.True(t, svc.step(ctx), "pending image must be claimed")
	after := svc.repo.Client.ImageTagAssignment.Query().CountX(ctx)
	assert.Equal(t, before, after, "empty aiSystemMessage must skip inference")

	got, err := svc.repo.GetImage(ctx, img)
	require.NoError(t, err)
	assert.Nil(t, got.InferredAt, "skip must not stamp inferredAt")
	assert.Empty(t, aiStatus(t, svc, img), "queue state must be cleared")
}

// Transient error: image goes back to pending (not lost), attempts increment,
// global backoff armed so claimNext returns nil. After maxAIAttempts it is
// parked as error with the message recorded.
func TestBackoffAndDeadLetter(t *testing.T) {
	svc, m := newSvc(t, failInference{})
	ctx := context.Background()
	img := m.Images[0]
	svc.Enqueue(img)

	for attempt := 1; attempt <= maxAIAttempts; attempt++ {
		svc.lock.Lock()
		svc.backoffUntil = time.Time{} // disarm for the next attempt
		svc.lock.Unlock()
		require.True(t, svc.step(ctx), "attempt %d must claim the image", attempt)

		row := svc.repo.Client.Image.GetX(ctx, img)
		assert.Equal(t, attempt, row.AiAttempts)
		if attempt < maxAIAttempts {
			assert.Equal(t, "pending", aiStatus(t, svc, img), "transient failure keeps the image queued")
			assert.True(t, time.Now().Before(svc.backoffUntil), "backoff must be armed")
			assert.False(t, svc.step(ctx), "backoff must block the next claim")
		} else {
			assert.Equal(t, "error", aiStatus(t, svc, img), "attempts exhausted -> dead-letter")
			assert.Contains(t, row.AiError, "boom")
		}
	}

	// dead-lettered images are not claimed again.
	svc.lock.Lock()
	svc.backoffUntil = time.Time{}
	svc.lock.Unlock()
	assert.False(t, svc.step(ctx))

	// manual rerun resets the row and it processes again.
	svc.Enqueue(img)
	assert.Equal(t, "pending", aiStatus(t, svc, img))
	assert.Equal(t, 0, svc.repo.Client.Image.GetX(ctx, img).AiAttempts)
}

// Boot recovery: rows orphaned in processing are re-queued by Start.
func TestBootRecovery(t *testing.T) {
	svc, m := newSvc(t, &StubInference{Tags: []string{"none"}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	img := m.Images[0]

	svc.repo.Client.Image.UpdateOneID(img).
		SetAiStatus(entimage.AiStatusProcessing).SetAiQueuedAt(time.Now()).SaveX(ctx)

	svc.Start(ctx)
	cancel() // dispatcher exits; recovery already ran synchronously
	assert.Equal(t, "pending", aiStatus(t, svc, img))
}
