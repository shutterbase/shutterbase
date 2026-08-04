package repository

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/scheduleitem"
	"github.com/shutterbase/shutterbase/ent/schema"
	"github.com/shutterbase/shutterbase/internal/util"
)

var (
	// ErrInvalidTimeline flags a structurally broken track (both/neither ids,
	// end not after start).
	ErrInvalidTimeline = errors.New("invalid_timeline")
	// ErrScheduleOverlap flags two enabled schedule-item tracks overlapping —
	// schedule tags are mutually exclusive by design.
	ErrScheduleOverlap = errors.New("schedule_track_overlap")
)

// TimelineApplyResult reports what one apply changed.
type TimelineApplyResult struct {
	Upload  *ent.Upload
	Created int
	Deleted int
}

// imageTime is the effective timeline position of an image: the time-sync
// corrected capture time, falling back to the raw one. ok=false when the image
// has neither and cannot be placed.
func imageTime(img *ent.Image) (time.Time, bool) {
	if img.CapturedAtCorrected != nil {
		return *img.CapturedAtCorrected, true
	}
	if img.CapturedAt != nil {
		return *img.CapturedAt, true
	}
	return time.Time{}, false
}

// ApplyUploadTimeline reconciles the upload's "scheduled" tag assignments with
// the given editor tracks and persists the tracks as the upload's timeline —
// atomically. The diff only ever touches scheduled-type rows: an existing
// manual/inferred/default assignment of the same (image, tag) pair is left
// alone (and never duplicated), so hand-set tags survive any handle-dragging.
func (r *Repository) ApplyUploadTimeline(ctx context.Context, uploadID string, tracks []schema.TimelineTrack) (*TimelineApplyResult, error) {
	up, err := r.Client.Upload.Get(ctx, uploadID)
	if err != nil {
		return nil, err
	}

	// Structural validation + id collection.
	itemIDs := make([]string, 0, len(tracks))
	tagIDs := make([]string, 0, len(tracks))
	for _, tr := range tracks {
		if (tr.ScheduleItemID == "") == (tr.TagID == "") {
			return nil, ErrInvalidTimeline // exactly one of the two ids
		}
		if !tr.End.After(tr.Start) {
			return nil, ErrInvalidTimeline
		}
		if tr.ScheduleItemID != "" {
			itemIDs = append(itemIDs, tr.ScheduleItemID)
		} else {
			tagIDs = append(tagIDs, tr.TagID)
		}
	}

	// Resolve schedule items (with their suggestion tags) — project-local only.
	items, err := r.Client.ScheduleItem.Query().
		Where(scheduleitem.IDIn(uniqueStrings(itemIDs)...), scheduleitem.ProjectID(up.ProjectID)).
		WithTags().All(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) != len(uniqueStrings(itemIDs)) {
		return nil, ErrInvalidTimeline // unknown or foreign schedule item
	}
	itemTags := make(map[string][]string, len(items))
	for _, it := range items {
		ids := make([]string, 0, len(it.Edges.Tags))
		for _, t := range it.Edges.Tags {
			ids = append(ids, t.ID)
		}
		itemTags[it.ID] = ids
	}
	// Free tag tracks must reference tags of the upload's project.
	if len(tagIDs) > 0 {
		n, err := r.Client.ImageTag.Query().
			Where(imagetag.IDIn(uniqueStrings(tagIDs)...), imagetag.ProjectID(up.ProjectID)).
			Count(ctx)
		if err != nil {
			return nil, err
		}
		if n != len(uniqueStrings(tagIDs)) {
			return nil, ErrTagProjectMismatch
		}
	}

	// Enabled schedule-item tracks are mutually exclusive: sorted by start, each
	// must end before (or exactly when) the next begins.
	scheduleTracks := make([]schema.TimelineTrack, 0, len(tracks))
	for _, tr := range tracks {
		if tr.ScheduleItemID != "" && tr.Enabled {
			scheduleTracks = append(scheduleTracks, tr)
		}
	}
	sort.Slice(scheduleTracks, func(i, j int) bool { return scheduleTracks[i].Start.Before(scheduleTracks[j].Start) })
	for i := 1; i < len(scheduleTracks); i++ {
		if scheduleTracks[i].Start.Before(scheduleTracks[i-1].End) {
			return nil, ErrScheduleOverlap
		}
	}

	// Desired state: image -> tag set, computed from every enabled track whose
	// window [start, end) contains the image's effective capture time.
	images, err := r.Client.Image.Query().Where(image.UploadID(uploadID)).
		Select(image.FieldID, image.FieldCapturedAt, image.FieldCapturedAtCorrected).
		All(ctx)
	if err != nil {
		return nil, err
	}
	desired := map[string]map[string]struct{}{}
	imageIDs := make([]string, 0, len(images))
	for _, img := range images {
		imageIDs = append(imageIDs, img.ID)
		t, ok := imageTime(img)
		if !ok {
			continue
		}
		for _, tr := range tracks {
			if !tr.Enabled || t.Before(tr.Start) || !t.Before(tr.End) {
				continue
			}
			tags := itemTags[tr.ScheduleItemID]
			if tr.TagID != "" {
				tags = []string{tr.TagID}
			}
			for _, tagID := range tags {
				if desired[img.ID] == nil {
					desired[img.ID] = map[string]struct{}{}
				}
				desired[img.ID][tagID] = struct{}{}
			}
		}
	}

	// Reconcile inside one tx.
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	existing, err := tx.ImageTagAssignment.Query().
		Where(imagetagassignment.ImageIDIn(imageIDs...)).All(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	type pair struct{ img, tag string }
	existingAny := map[pair]struct{}{}
	existingScheduled := map[pair]string{} // -> assignment id
	for _, a := range existing {
		p := pair{a.ImageID, a.ImageTagID}
		existingAny[p] = struct{}{}
		if a.Type == imagetagassignment.TypeScheduled {
			existingScheduled[p] = a.ID
		}
	}

	affected := map[string]struct{}{}
	created, deleted := 0, 0
	for imgID, tags := range desired {
		for tagID := range tags {
			p := pair{imgID, tagID}
			if _, ok := existingAny[p]; ok {
				continue // present in any type — leave untouched, never duplicate
			}
			if err := tx.ImageTagAssignment.Create().
				SetType(imagetagassignment.TypeScheduled).
				SetImageID(imgID).
				SetImageTagID(tagID).
				SetCreatedBy(util.GetActorID(ctx)).
				SetUpdatedBy(util.GetActorID(ctx)).
				Exec(ctx); err != nil {
				_ = tx.Rollback()
				return nil, err
			}
			created++
			affected[imgID] = struct{}{}
		}
	}
	for p, id := range existingScheduled {
		if _, ok := desired[p.img][p.tag]; ok {
			continue
		}
		if err := tx.ImageTagAssignment.DeleteOneID(id).Exec(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		deleted++
		affected[p.img] = struct{}{}
	}
	for imgID := range affected {
		if err := r.rebuildImageTags(ctx, tx, imgID); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if tracks == nil {
		tracks = []schema.TimelineTrack{}
	}
	upRow, err := tx.Upload.UpdateOneID(uploadID).
		SetTimeline(tracks).
		SetUpdatedBy(util.GetActorID(ctx)).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "timeline_apply", ObjectType: util.StringPointer("upload"), ObjectId: util.StringPointer(uploadID),
			Data: &map[string]any{"tracks": len(tracks), "created": created, "deleted": deleted},
		})
	})
	log.Debug().Str("upload", uploadID).Int("created", created).Int("deleted", deleted).Msg("timeline applied")
	return &TimelineApplyResult{Upload: upRow, Created: created, Deleted: deleted}, nil
}
