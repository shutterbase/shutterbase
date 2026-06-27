package repository

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/util"
)

// CreateAuditLogParameters describes one audit row. ObjectId is a string so it
// holds both string PKs and the stringified uuid User PK.
type CreateAuditLogParameters struct {
	Action     string
	ObjectType *string
	ObjectId   *string
	Data       *map[string]any
}

// CreateAuditLog writes one immutable audit row. Called from mutation methods via
// safeGo(context.WithoutCancel(ctx), ...). No WS broadcast (deferred, §1).
func (r *Repository) CreateAuditLog(ctx context.Context, parameters *CreateAuditLogParameters) (*ent.AuditLog, error) {
	create := r.Client.AuditLog.Create().
		SetAction(parameters.Action).
		SetActor(util.GetActorID(ctx)).
		SetCreatedBy(util.GetActorID(ctx)).
		SetUpdatedBy(util.GetActorID(ctx))

	if parameters.ObjectType != nil {
		create = create.SetObjectType(*parameters.ObjectType)
	}
	if parameters.ObjectId != nil {
		create = create.SetObjectId(*parameters.ObjectId)
	}
	if parameters.Data != nil {
		create = create.SetData(*parameters.Data)
	}

	item, err := create.Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error creating audit log")
		return nil, err
	}
	return item, nil
}
