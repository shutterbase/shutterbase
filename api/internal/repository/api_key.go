package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/apikey"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/internal/util"
)

var apiKeySortFields = map[string]string{
	"name":      apikey.FieldName,
	"createdAt": apikey.FieldCreatedAt,
	"updatedAt": apikey.FieldUpdatedAt,
}

func (r *Repository) GetApiKey(ctx context.Context, id string) (*ent.ApiKey, error) {
	item, err := r.Client.ApiKey.Query().Where(apikey.IDEQ(id)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Msg("error getting api key")
	}
	return item, err
}

// GetApiKeyByKeyId looks up an active (non-revoked) key by its public keyId — the
// hot path for API-key auth.
func (r *Repository) GetApiKeyByKeyId(ctx context.Context, keyId string) (*ent.ApiKey, error) {
	item, err := r.Client.ApiKey.Query().
		Where(apikey.KeyIdEQ(keyId), apikey.RevokedEQ(false)).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Msg("error getting api key by keyId")
	}
	return item, err
}

type GetApiKeyParameters struct {
	UserID               *uuid.UUID
	PaginationParameters *PaginationParameters
}

func (r *Repository) GetApiKeys(ctx context.Context, parameters *GetApiKeyParameters) ([]*ent.ApiKey, int, error) {
	predicates := []predicate.ApiKey{}
	if parameters.UserID != nil {
		predicates = append(predicates, apikey.UserID(*parameters.UserID))
	}
	where := apikey.And(predicates...)

	limit, offset, order, err := parameters.PaginationParameters.build(apiKeySortFields, "createdAt")
	if err != nil {
		return nil, 0, err
	}
	items, err := r.Client.ApiKey.Query().Where(where).Limit(limit).Offset(offset).Order(order).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting api keys")
		return nil, 0, err
	}
	total, err := r.Client.ApiKey.Query().Where(where).Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type CreateApiKeyParameters struct {
	KeyID      string
	SecretHash string
	Name       string
	UserID     uuid.UUID
}

func (r *Repository) CreateApiKey(ctx context.Context, parameters *CreateApiKeyParameters) (*ent.ApiKey, error) {
	item, err := r.Client.ApiKey.Create().
		SetKeyId(parameters.KeyID).
		SetSecretHash(parameters.SecretHash).
		SetName(parameters.Name).
		SetUserID(parameters.UserID).
		SetCreatedBy(util.GetActorID(ctx)).
		SetUpdatedBy(util.GetActorID(ctx)).
		Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error creating api key")
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "create", ObjectType: util.StringPointer("api_key"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"name": item.Name, "keyId": item.KeyId},
		})
	})
	return item, nil
}

// RevokeApiKey flips revoked=true (soft delete — the row stays for audit/history).
func (r *Repository) RevokeApiKey(ctx context.Context, id string) error {
	if err := r.Client.ApiKey.UpdateOneID(id).
		SetRevoked(true).
		SetUpdatedBy(util.GetActorID(ctx)).
		Exec(ctx); err != nil {
		log.Error().Err(err).Msg("error revoking api key")
		return err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "revoke", ObjectType: util.StringPointer("api_key"), ObjectId: util.StringPointer(id),
			Data: &map[string]any{},
		})
	})
	return nil
}

// TouchApiKey best-effort stamps lastUsedAt. Errors are swallowed — a failed
// timestamp update must never fail the authenticated request.
func (r *Repository) TouchApiKey(ctx context.Context, id string) {
	if err := r.Client.ApiKey.UpdateOneID(id).SetLastUsedAt(time.Now()).Exec(ctx); err != nil {
		log.Debug().Err(err).Str("id", id).Msg("failed to update api key lastUsedAt")
	}
}
