package repository

import (
	"context"
	"slices"
	"time"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/ent/upload"
	"github.com/shutterbase/shutterbase/internal/util"
)

var uploadSortFields = map[string]string{
	"name":      upload.FieldName,
	"createdAt": upload.FieldCreatedAt,
	"updatedAt": upload.FieldUpdatedAt,
}

func (r *Repository) GetUpload(ctx context.Context, id string) (*ent.Upload, error) {
	item, err := r.Client.Upload.Query().Where(upload.IDEQ(id)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Msg("error getting upload")
	}
	return item, err
}

type GetUploadParameters struct {
	ProjectID            *string
	UserID               *uuid.UUID
	State                *upload.State
	PaginationParameters *PaginationParameters
}

func (r *Repository) GetUploads(ctx context.Context, parameters *GetUploadParameters) ([]*ent.Upload, int, error) {
	predicates := []predicate.Upload{}
	if parameters.ProjectID != nil {
		predicates = append(predicates, upload.ProjectID(*parameters.ProjectID))
	}
	if parameters.UserID != nil {
		predicates = append(predicates, upload.UserID(*parameters.UserID))
	}
	if parameters.State != nil {
		predicates = append(predicates, upload.StateEQ(*parameters.State))
	}
	where := upload.And(predicates...)

	limit, offset, order, err := parameters.PaginationParameters.build(uploadSortFields, "createdAt")
	if err != nil {
		return nil, 0, err
	}
	items, err := r.Client.Upload.Query().Where(where).Limit(limit).Offset(offset).Order(order).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting uploads")
		return nil, 0, err
	}
	total, err := r.Client.Upload.Query().Where(where).Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type CreateUploadParameters struct {
	Name      string
	ProjectID string
	UserID    uuid.UUID
	CameraID  string
}

func (r *Repository) CreateUpload(ctx context.Context, parameters *CreateUploadParameters) (*ent.Upload, error) {
	item, err := r.Client.Upload.Create().
		SetName(parameters.Name).
		SetProjectID(parameters.ProjectID).
		SetUserID(parameters.UserID).
		SetCameraID(parameters.CameraID).
		SetCycleStartedAt(util.Now()). // first open -> ready cycle starts now
		SetCreatedBy(util.GetActorID(ctx)).
		SetUpdatedBy(util.GetActorID(ctx)).
		Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error creating upload")
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "create", ObjectType: util.StringPointer("upload"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"name": item.Name},
		})
	})
	return item, nil
}

type UpdateUploadParameters struct {
	Name  *string
	State *upload.State
}

// applyCycle folds a state change into the review-cycle metrics and returns the
// new (timeToReadySeconds, reviewCycles, cycleStartedAt). Leaving "open" closes
// a cycle — the wall clock since the cycle started is banked and a submission is
// counted; returning to "open" (sent back / reopened) starts the next one. Pure,
// so the accounting is unit-testable without a database.
func applyCycle(up *ent.Upload, next upload.State, now time.Time) (int, int, *time.Time) {
	seconds, cycles, start := up.TimeToReadySeconds, up.ReviewCycles, up.CycleStartedAt
	switch {
	case up.State == upload.StateOpen && next != upload.StateOpen:
		from := up.CreatedAt
		if start != nil {
			from = *start
		}
		if d := int(now.Sub(from).Seconds()); d > 0 {
			seconds += d
		}
		cycles++
		start = nil
	case up.State != upload.StateOpen && next == upload.StateOpen:
		at := now
		start = &at
	}
	return seconds, cycles, start
}

func (r *Repository) UpdateUpload(ctx context.Context, id string, parameters *UpdateUploadParameters) (*ent.Upload, error) {
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	q := tx.Upload.Query().Where(upload.IDEQ(id))
	if r.isPostgres() {
		q = q.ForUpdate()
	}
	item, err := q.Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	update := tx.Upload.UpdateOneID(id).SetUpdatedBy(util.GetActorID(ctx))
	st := modelUpdateStatus{}
	if parameters.Name != nil && item.Name != *parameters.Name {
		update.SetName(*parameters.Name)
		st.SetFieldChanged(upload.FieldName, item.Name, *parameters.Name)
	}
	if parameters.State != nil && item.State != *parameters.State {
		seconds, cycles, start := applyCycle(item, *parameters.State, util.Now())
		update.SetState(*parameters.State).SetTimeToReadySeconds(seconds).SetReviewCycles(cycles)
		if start != nil {
			update.SetCycleStartedAt(*start)
		} else {
			update.ClearCycleStartedAt()
		}
		st.SetFieldChanged(upload.FieldState, item.State.String(), parameters.State.String())
	}
	if !st.modelChanged {
		_ = tx.Rollback()
		return item, nil
	}
	if _, err := update.Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	item, err = r.Client.Upload.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "update", ObjectType: util.StringPointer("upload"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"changes": st.GetChangedFieldData()},
		})
	})
	return item, nil
}

// defaultTaggingIdleThreshold is the longest gap between two tagging actions
// that still counts as working time. Anything longer is a break and is dropped.
// Calibration knob (TAGGING_IDLE_THRESHOLD): the right value is the rhythm of a
// real photographer working a shoot, not a number derivable from first
// principles — tune it against measured uploads.
const defaultTaggingIdleThreshold = 2 * time.Minute

func (r *Repository) taggingIdleThreshold() time.Duration {
	if r.Options != nil && r.Options.TaggingIdleThreshold > 0 {
		return r.Options.TaggingIdleThreshold
	}
	return defaultTaggingIdleThreshold
}

// mutateUpload applies fn to a row-locked upload inside a transaction. fn
// reports whether it changed anything; false rolls back. Used by the metric
// accumulators, which must read-modify-write without losing concurrent updates.
func (r *Repository) mutateUpload(ctx context.Context, id string, fn func(*ent.Upload, *ent.UploadUpdateOne) bool) error {
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		return err
	}
	q := tx.Upload.Query().Where(upload.IDEQ(id))
	if r.isPostgres() {
		q = q.ForUpdate()
	}
	item, err := q.Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	update := tx.Upload.UpdateOneID(id)
	if !fn(item, update) {
		return tx.Rollback()
	}
	if _, err := update.Save(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// foldTaggingActivity folds a tagging action at `at` into the active-time
// accumulator: the gap since the previous action counts as work when it is
// shorter than idle, otherwise the photographer was on a break. Returns the new
// (taggingSeconds, lastTagActivityAt) and whether anything moved. An out-of-order
// action (older than the last recorded one) is ignored so async/retried callers
// cannot walk the clock backwards. Pure — the heuristic is unit-tested directly.
func foldTaggingActivity(seconds int, last *time.Time, at time.Time, idle time.Duration) (int, time.Time, bool) {
	if last == nil {
		return seconds, at, true
	}
	if !at.After(*last) {
		return seconds, *last, false
	}
	if gap := at.Sub(*last); gap <= idle {
		seconds += int(gap.Seconds())
	}
	return seconds, at, true
}

// RecordTaggingActivity folds one tagging action on an upload's images into the
// upload's active tagging time. Called for every manual tag assign/unassign the
// upload's own photographer performs; a reviewer's edits are not their working
// time and are not recorded.
func (r *Repository) RecordTaggingActivity(ctx context.Context, uploadID string, at time.Time) error {
	idle := r.taggingIdleThreshold()
	err := r.mutateUpload(ctx, uploadID, func(item *ent.Upload, update *ent.UploadUpdateOne) bool {
		seconds, last, changed := foldTaggingActivity(item.TaggingSeconds, item.LastTagActivityAt, at, idle)
		if !changed {
			return false
		}
		update.SetTaggingSeconds(seconds).SetLastTagActivityAt(last)
		return true
	})
	if err != nil {
		log.Error().Err(err).Str("upload", uploadID).Msg("error recording tagging activity")
	}
	return err
}

// TrackUploadTaggingError remembers that imageID carried the review error tag,
// so the count survives the tag being cleared and every later review cycle.
func (r *Repository) TrackUploadTaggingError(ctx context.Context, uploadID, imageID string) error {
	err := r.mutateUpload(ctx, uploadID, func(item *ent.Upload, update *ent.UploadUpdateOne) bool {
		if slices.Contains(item.ErrorImageIds, imageID) {
			return false
		}
		update.SetErrorImageIds(append(slices.Clone(item.ErrorImageIds), imageID))
		return true
	})
	if err != nil {
		log.Error().Err(err).Str("upload", uploadID).Msg("error tracking upload tagging error")
	}
	return err
}

// UploadMetrics is the per-upload tagging performance block (§4.9). Rates are
// derived, not stored, so they always match the current image/tag counts.
type UploadMetrics struct {
	ImageCount         int     `json:"imageCount"`
	TagCount           int     `json:"tagCount"`
	TagsPerImage       float64 `json:"tagsPerImage"`
	TaggingSeconds     int     `json:"taggingSeconds"`
	ImagesPerSecond    float64 `json:"imagesPerSecond"`
	TimeToReadySeconds int     `json:"timeToReadySeconds"`
	ReviewCycles       int     `json:"reviewCycles"`
	ErrorCount         int     `json:"errorCount"`
	// AI detection progress: counts of this upload's images by queue state
	// (in-flight = pending + processing). Zero-valued on AI-less projects.
	AiDone     int `json:"aiDone"`
	AiInFlight int `json:"aiInFlight"`
	AiError    int `json:"aiError"`
}

// GetUploadMetrics builds the metric block for the given uploads. Two grouped
// queries for the whole set (image count per upload, manual tag assignments per
// upload) — never per row.
func (r *Repository) GetUploadMetrics(ctx context.Context, uploads []*ent.Upload) (map[string]*UploadMetrics, error) {
	out := make(map[string]*UploadMetrics, len(uploads))
	ids := make([]string, 0, len(uploads))
	for _, up := range uploads {
		ids = append(ids, up.ID)
		out[up.ID] = &UploadMetrics{
			TaggingSeconds:     up.TaggingSeconds,
			TimeToReadySeconds: up.TimeToReadySeconds,
			ReviewCycles:       up.ReviewCycles,
			ErrorCount:         len(up.ErrorImageIds),
		}
	}
	if len(ids) == 0 {
		return out, nil
	}

	var imageCounts []struct {
		UploadID string `json:"upload_id"`
		Count    int    `json:"count"`
	}
	if err := r.Client.Image.Query().
		Where(image.UploadIDIn(ids...)).
		GroupBy(image.FieldUploadID).
		Aggregate(ent.Count()).
		Scan(ctx, &imageCounts); err != nil {
		log.Error().Err(err).Msg("error counting upload images")
		return nil, err
	}
	for _, row := range imageCounts {
		if m, ok := out[row.UploadID]; ok {
			m.ImageCount = row.Count
		}
	}

	tagCounts, err := r.manualTagCountsByUpload(ctx, ids)
	if err != nil {
		return nil, err
	}
	for id, count := range tagCounts {
		if m, ok := out[id]; ok {
			m.TagCount = count
		}
	}

	var aiCounts []struct {
		UploadID string `json:"upload_id"`
		AiStatus string `json:"ai_status"`
		Count    int    `json:"count"`
	}
	if err := r.Client.Image.Query().
		Where(image.UploadIDIn(ids...), image.AiStatusNotNil()).
		GroupBy(image.FieldUploadID, image.FieldAiStatus).
		Aggregate(ent.Count()).
		Scan(ctx, &aiCounts); err != nil {
		log.Error().Err(err).Msg("error counting upload AI statuses")
		return nil, err
	}
	for _, row := range aiCounts {
		m, ok := out[row.UploadID]
		if !ok {
			continue
		}
		switch row.AiStatus {
		case "done":
			m.AiDone += row.Count
		case "error":
			m.AiError += row.Count
		default: // pending | processing
			m.AiInFlight += row.Count
		}
	}

	for _, m := range out {
		if m.ImageCount > 0 {
			m.TagsPerImage = float64(m.TagCount) / float64(m.ImageCount)
		}
		if m.TaggingSeconds > 0 {
			m.ImagesPerSecond = float64(m.ImageCount) / float64(m.TaggingSeconds)
		}
	}
	return out, nil
}

// manualTagCountsByUpload counts manual tag assignments per upload in one query.
// ent's typed GroupBy cannot group by a joined table's column, so this drops to
// the ent SQL builder (dialect-aware placeholders, column names from the
// generated constants) rather than a hand-written string.
func (r *Repository) manualTagCountsByUpload(ctx context.Context, ids []string) (map[string]int, error) {
	d := dialect.SQLite
	if r.isPostgres() {
		d = dialect.Postgres
	}
	// Explicit aliases: the builder auto-aliases a joined table, but the select
	// list is captured BEFORE the Join call — an implicit alias then leaves the
	// projection pointing at a table name that is no longer in the FROM clause.
	assignments := sql.Table(imagetagassignment.Table).As("a")
	images := sql.Table(image.Table).As("i")
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	query, params := sql.Dialect(d).
		Select(images.C(image.FieldUploadID), sql.Count("*")).
		From(assignments).
		Join(images).On(assignments.C(imagetagassignment.FieldImageID), images.C(image.FieldID)).
		Where(sql.And(
			sql.In(images.C(image.FieldUploadID), args...),
			sql.EQ(assignments.C(imagetagassignment.FieldType), imagetagassignment.TypeManual),
		)).
		GroupBy(images.C(image.FieldUploadID)).
		Query()

	rows, err := r.Options.DatabaseConnection.DB.QueryContext(ctx, query, params...)
	if err != nil {
		log.Error().Err(err).Msg("error counting upload tag assignments")
		return nil, err
	}
	defer rows.Close()
	counts := make(map[string]int, len(ids))
	for rows.Next() {
		var id string
		var count int
		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}
		counts[id] = count
	}
	return counts, rows.Err()
}

func (r *Repository) DeleteUpload(ctx context.Context, id string) error {
	if err := r.Client.Upload.DeleteOneID(id).Exec(ctx); err != nil {
		log.Error().Err(err).Msg("error deleting upload")
		return err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "delete", ObjectType: util.StringPointer("upload"), ObjectId: util.StringPointer(id),
			Data: &map[string]any{},
		})
	})
	return nil
}
