package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/ent/scheduleitem"
	"github.com/shutterbase/shutterbase/ent/user"
	"github.com/shutterbase/shutterbase/internal/util"
)

// ErrTagProjectMismatch is returned when a schedule item references tags of
// another project (trust boundary: tag suggestions must stay project-local).
var ErrTagProjectMismatch = errors.New("tag_project_mismatch")

var scheduleItemSortFields = map[string]string{
	"start":     scheduleitem.FieldStart,
	"end":       scheduleitem.FieldEnd,
	"title":     scheduleitem.FieldTitle,
	"createdAt": scheduleitem.FieldCreatedAt,
	"updatedAt": scheduleitem.FieldUpdatedAt,
}

// scheduleItemQuery eager-loads what every serialization needs: assignees (for
// the avatar bubbles) and the suggestion tags.
func (r *Repository) scheduleItemQuery() *ent.ScheduleItemQuery {
	return r.Client.ScheduleItem.Query().WithAssignees().WithTags()
}

func (r *Repository) GetScheduleItem(ctx context.Context, id string) (*ent.ScheduleItem, error) {
	item, err := r.scheduleItemQuery().Where(scheduleitem.IDEQ(id)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Msg("error getting schedule item")
	}
	return item, err
}

type GetScheduleItemsParameters struct {
	ProjectID            string
	From                 *time.Time // overlap filter: item.end > From
	To                   *time.Time // overlap filter: item.start < To
	AssigneeID           *uuid.UUID
	PaginationParameters *PaginationParameters
}

func (r *Repository) GetScheduleItems(ctx context.Context, parameters *GetScheduleItemsParameters) ([]*ent.ScheduleItem, int, error) {
	predicates := []predicate.ScheduleItem{scheduleitem.ProjectID(parameters.ProjectID)}
	if parameters.From != nil {
		predicates = append(predicates, scheduleitem.EndGT(*parameters.From))
	}
	if parameters.To != nil {
		predicates = append(predicates, scheduleitem.StartLT(*parameters.To))
	}
	if parameters.AssigneeID != nil {
		predicates = append(predicates, scheduleitem.HasAssigneesWith(user.IDEQ(*parameters.AssigneeID)))
	}
	where := scheduleitem.And(predicates...)

	limit, offset, order, err := parameters.PaginationParameters.build(scheduleItemSortFields, "start")
	if err != nil {
		return nil, 0, err
	}
	items, err := r.scheduleItemQuery().Where(where).Limit(limit).Offset(offset).Order(order).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting schedule items")
		return nil, 0, err
	}
	total, err := r.Client.ScheduleItem.Query().Where(where).Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// validateTagsInProject guards the tags edge: every referenced tag must belong
// to the item's project. Takes the tag client so callers inside a transaction
// pass tx.ImageTag — a non-tx query while a tx holds SQLite's single
// connection deadlocks the process.
func validateTagsInProject(ctx context.Context, tagClient *ent.ImageTagClient, projectID string, tagIDs []string) error {
	if len(tagIDs) == 0 {
		return nil
	}
	n, err := tagClient.Query().
		Where(imagetag.IDIn(tagIDs...), imagetag.ProjectID(projectID)).
		Count(ctx)
	if err != nil {
		return err
	}
	if n != len(uniqueStrings(tagIDs)) {
		return ErrTagProjectMismatch
	}
	return nil
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

type CreateScheduleItemParameters struct {
	Title       string
	Description string
	Start       time.Time
	End         time.Time
	Cardinality int
	ProjectID   string
	TagIDs      []string
}

func (r *Repository) CreateScheduleItem(ctx context.Context, parameters *CreateScheduleItemParameters) (*ent.ScheduleItem, error) {
	if err := validateTagsInProject(ctx, r.Client.ImageTag, parameters.ProjectID, parameters.TagIDs); err != nil {
		return nil, err
	}
	create := r.Client.ScheduleItem.Create().
		SetTitle(parameters.Title).
		SetDescription(parameters.Description).
		SetStart(parameters.Start).
		SetEnd(parameters.End).
		SetProjectID(parameters.ProjectID).
		AddTagIDs(uniqueStrings(parameters.TagIDs)...).
		SetCreatedBy(util.GetActorID(ctx)).
		SetUpdatedBy(util.GetActorID(ctx))
	if parameters.Cardinality > 0 {
		create = create.SetCardinality(parameters.Cardinality)
	}
	item, err := create.Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error creating schedule item")
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "create", ObjectType: util.StringPointer("schedule_item"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"title": item.Title},
		})
	})
	return r.GetScheduleItem(ctx, item.ID)
}

type UpdateScheduleItemParameters struct {
	Title       *string
	Description *string
	Start       *time.Time
	End         *time.Time
	Cardinality *int
	TagIDs      *[]string
}

func (r *Repository) UpdateScheduleItem(ctx context.Context, id string, parameters *UpdateScheduleItemParameters) (*ent.ScheduleItem, error) {
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	q := tx.ScheduleItem.Query().Where(scheduleitem.IDEQ(id))
	if r.isPostgres() {
		q = q.ForUpdate()
	}
	item, err := q.Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	update := tx.ScheduleItem.UpdateOneID(id).SetUpdatedBy(util.GetActorID(ctx))
	st := modelUpdateStatus{}
	if parameters.Title != nil && item.Title != *parameters.Title {
		update.SetTitle(*parameters.Title)
		st.SetFieldChanged(scheduleitem.FieldTitle, item.Title, *parameters.Title)
	}
	if parameters.Description != nil && item.Description != *parameters.Description {
		update.SetDescription(*parameters.Description)
		st.SetFieldChanged(scheduleitem.FieldDescription, item.Description, *parameters.Description)
	}
	if parameters.Start != nil && !item.Start.Equal(*parameters.Start) {
		update.SetStart(*parameters.Start)
		st.SetFieldChanged(scheduleitem.FieldStart, item.Start, *parameters.Start)
	}
	if parameters.End != nil && !item.End.Equal(*parameters.End) {
		update.SetEnd(*parameters.End)
		st.SetFieldChanged(scheduleitem.FieldEnd, item.End, *parameters.End)
	}
	if parameters.Cardinality != nil && item.Cardinality != *parameters.Cardinality {
		update.SetCardinality(*parameters.Cardinality)
		st.SetFieldChanged(scheduleitem.FieldCardinality, item.Cardinality, *parameters.Cardinality)
	}
	if parameters.TagIDs != nil {
		if err := validateTagsInProject(ctx, tx.ImageTag, item.ProjectID, *parameters.TagIDs); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		update.ClearTags().AddTagIDs(uniqueStrings(*parameters.TagIDs)...)
		st.SetFieldChanged("tags", nil, *parameters.TagIDs)
	}
	if !st.modelChanged {
		_ = tx.Rollback()
		return r.GetScheduleItem(ctx, id)
	}
	if _, err := update.Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "update", ObjectType: util.StringPointer("schedule_item"), ObjectId: util.StringPointer(id),
			Data: &map[string]any{"changes": st.GetChangedFieldData()},
		})
	})
	return r.GetScheduleItem(ctx, id)
}

func (r *Repository) DeleteScheduleItem(ctx context.Context, id string) error {
	if err := r.Client.ScheduleItem.DeleteOneID(id).Exec(ctx); err != nil {
		return err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "delete", ObjectType: util.StringPointer("schedule_item"), ObjectId: util.StringPointer(id),
		})
	})
	return nil
}

// AssignScheduleItemUser adds userID to the item's assignees. Idempotent —
// assigning twice is a no-op, and there is deliberately NO cardinality cap:
// overbooking is allowed by design (violett, Maximum Power).
func (r *Repository) AssignScheduleItemUser(ctx context.Context, id string, userID uuid.UUID) (*ent.ScheduleItem, error) {
	exists, err := r.Client.ScheduleItem.Query().
		Where(scheduleitem.IDEQ(id), scheduleitem.HasAssigneesWith(user.IDEQ(userID))).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := r.Client.ScheduleItem.UpdateOneID(id).
			AddAssigneeIDs(userID).
			SetUpdatedBy(util.GetActorID(ctx)).
			Exec(ctx); err != nil {
			return nil, err
		}
		safeGo(func() {
			r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
				Action: "assign", ObjectType: util.StringPointer("schedule_item"), ObjectId: util.StringPointer(id),
				Data: &map[string]any{"userId": userID.String()},
			})
		})
	}
	return r.GetScheduleItem(ctx, id)
}

// UnassignScheduleItemUser removes userID from the item's assignees (no-op when
// not assigned).
func (r *Repository) UnassignScheduleItemUser(ctx context.Context, id string, userID uuid.UUID) (*ent.ScheduleItem, error) {
	item, err := r.Client.ScheduleItem.Query().
		Where(scheduleitem.IDEQ(id), scheduleitem.HasAssigneesWith(user.IDEQ(userID))).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return r.GetScheduleItem(ctx, id)
		}
		return nil, err
	}
	if err := r.Client.ScheduleItem.UpdateOneID(item.ID).
		RemoveAssigneeIDs(userID).
		SetUpdatedBy(util.GetActorID(ctx)).
		Exec(ctx); err != nil {
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "unassign", ObjectType: util.StringPointer("schedule_item"), ObjectId: util.StringPointer(id),
			Data: &map[string]any{"userId": userID.String()},
		})
	})
	return r.GetScheduleItem(ctx, id)
}
