// White-box tests (package service) so they can drive the unexported step() and
// inspect the DB queue state directly — fast and deterministic, no goroutine
// timing. They run on the seeded SQLite repo; presigning is offline so a dummy
// S3 client produces a URL without any network.
package service

import (
	"context"
	"errors"
	"fmt"
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

// deadlineInference times out for one image id and succeeds for the rest.
type deadlineInference struct {
	failFor string
	seen    []string
}

func (d *deadlineInference) Infer(_ context.Context, req InferenceRequest) ([]InferredTag, error) {
	d.seen = append(d.seen, req.ImageID)
	if req.ImageID == d.failFor {
		return nil, fmt.Errorf("infer: %w", context.DeadlineExceeded)
	}
	return []InferredTag{}, nil
}

// A deadline-exceeded inference retries at the BACK of the FIFO without arming
// the global backoff, and still dead-letters once attempts are exhausted.
func TestDeadlineRetryRequeuesAtBack(t *testing.T) {
	t.Setenv("SESSION_SECRET_KEY", "x")
	require.NoError(t, util.InitConfig())
	inf := &deadlineInference{}
	svc, m := newSvc(t, inf)
	ctx := context.Background()
	slow, fast := m.Images[0], m.Images[1]
	inf.failFor = slow

	base := time.Now().Add(-time.Minute)
	svc.Enqueue(slow)
	svc.repo.Client.Image.UpdateOneID(slow).SetAiQueuedAt(base).SaveX(ctx)
	svc.Enqueue(fast)
	svc.repo.Client.Image.UpdateOneID(fast).SetAiQueuedAt(base.Add(time.Second)).SaveX(ctx)

	// 1st claim: the slow image times out -> pending again, requeued behind
	// the fast one, no global backoff.
	require.True(t, svc.step(ctx))
	assert.Equal(t, []string{slow}, inf.seen)
	assert.Equal(t, "pending", aiStatus(t, svc, slow))
	assert.True(t, svc.backoffUntil.IsZero(), "deadline errors must not arm the global backoff")
	row := svc.repo.Client.Image.GetX(ctx, slow)
	assert.Equal(t, 1, row.AiAttempts)
	assert.True(t, row.AiQueuedAt.After(base.Add(time.Second)), "timed-out image must move to the back of the queue")

	// 2nd claim: the fast image goes first now.
	require.True(t, svc.step(ctx))
	assert.Equal(t, fast, inf.seen[1])
	assert.Equal(t, "done", aiStatus(t, svc, fast))

	// remaining attempts drain into the dead-letter cap.
	for svc.step(ctx) {
	}
	assert.Equal(t, "error", aiStatus(t, svc, slow), "attempts exhausted -> dead-letter")
	assert.Equal(t, maxAIAttempts, svc.repo.Client.Image.GetX(ctx, slow).AiAttempts)
}

// netTimeoutErr mimics http.Client.Timeout's error shape: a net.Error with
// Timeout()=true that does NOT wrap context.DeadlineExceeded.
type netTimeoutErr struct{}

func (netTimeoutErr) Error() string   { return "Client.Timeout exceeded while awaiting headers" }
func (netTimeoutErr) Timeout() bool   { return true }
func (netTimeoutErr) Temporary() bool { return true }

type netTimeoutInference struct{}

func (netTimeoutInference) Infer(_ context.Context, _ InferenceRequest) ([]InferredTag, error) {
	return nil, fmt.Errorf("infer: %w", netTimeoutErr{})
}

// An http-client-level timeout (net.Error, not DeadlineExceeded) classifies as
// a timeout too: back of the queue, no global backoff.
func TestClientTimeoutClassifiesAsTimeout(t *testing.T) {
	t.Setenv("SESSION_SECRET_KEY", "x")
	require.NoError(t, util.InitConfig())
	svc, m := newSvc(t, netTimeoutInference{})
	ctx := context.Background()
	img := m.Images[0]
	base := time.Now().Add(-time.Minute)
	svc.Enqueue(img)
	svc.repo.Client.Image.UpdateOneID(img).SetAiQueuedAt(base).SaveX(ctx)

	require.True(t, svc.step(ctx))
	assert.Equal(t, "pending", aiStatus(t, svc, img))
	assert.True(t, svc.backoffUntil.IsZero(), "client timeouts must not arm the global backoff")
	row := svc.repo.Client.Image.GetX(ctx, img)
	assert.True(t, row.AiQueuedAt.After(base), "timed-out image must move to the back of the queue")
}

// scopeInference records the Scope of each request it serves.
type scopeInference struct{ scopes []string }

func (s *scopeInference) Infer(_ context.Context, req InferenceRequest) ([]InferredTag, error) {
	s.scopes = append(s.scopes, req.Scope)
	return []InferredTag{}, nil
}

// A scoped project rerun persists aiScope, forwards it to the inference
// request, and clears it on done; a subsequent single-image Enqueue is a full
// run again (scope cleared).
func TestScopedRerunThreadsScope(t *testing.T) {
	inf := &scopeInference{}
	svc, m := newSvc(t, inf)
	ctx := context.Background()

	n, err := svc.EnqueueProject(m.Project, "numbers")
	require.NoError(t, err)
	require.Equal(t, len(m.Images), n)
	assert.Equal(t, "numbers", svc.repo.Client.Image.GetX(ctx, m.Images[0]).AiScope)

	for svc.step(ctx) {
	}
	require.Len(t, inf.scopes, len(m.Images))
	for _, scope := range inf.scopes {
		assert.Equal(t, "numbers", scope)
	}
	assert.Empty(t, svc.repo.Client.Image.GetX(ctx, m.Images[0]).AiScope, "done must clear aiScope")

	svc.Enqueue(m.Images[0])
	require.True(t, svc.step(ctx))
	assert.Equal(t, "", inf.scopes[len(inf.scopes)-1], "single-image rerun must be a full run")
}

// Boot recovery: STALE processing rows are re-queued by Start; fresh ones are
// left alone (they belong to another replica mid-flight during a rolling deploy).
func TestBootRecovery(t *testing.T) {
	svc, m := newSvc(t, &StubInference{Tags: []string{"none"}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stale, fresh := m.Images[0], m.Images[1]

	svc.repo.Client.Image.UpdateOneID(stale).
		SetAiStatus(entimage.AiStatusProcessing).SetAiQueuedAt(time.Now()).SaveX(ctx)
	// age the row past the orphan threshold (updatedAt is bumped by the save above)
	svc.repo.Client.Image.UpdateOneID(stale).
		SetUpdatedAt(time.Now().Add(-15 * time.Minute)).SaveX(ctx)
	svc.repo.Client.Image.UpdateOneID(fresh).
		SetAiStatus(entimage.AiStatusProcessing).SetAiQueuedAt(time.Now()).SaveX(ctx)

	svc.Start(ctx)
	cancel() // dispatcher exits; recovery already ran synchronously
	assert.Equal(t, "pending", aiStatus(t, svc, stale), "stale processing row must be re-queued")
	assert.Equal(t, "processing", aiStatus(t, svc, fresh), "fresh processing row must be left to its replica")
}
