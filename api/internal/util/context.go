package util

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
)

type userContextKey struct{}

// UserKey is the context key under which the auth middleware stores the
// effective user (impersonated if active, else real — S8).
var UserKey = userContextKey{}

// GetUser returns the effective user from the context, or nil if unauthenticated.
func GetUser(ctx context.Context) *ent.User {
	if user, ok := ctx.Value(UserKey).(*ent.User); ok {
		return user
	}
	return nil
}

// GetActorID returns the effective user's id for createdBy/updatedBy/audit
// attribution, falling back to SystemUserID when there is no user in context
// (e.g. seeding, importer, background jobs).
func GetActorID(ctx context.Context) uuid.UUID {
	if user := GetUser(ctx); user != nil {
		return user.ID
	}
	log.Debug().Msg("no user in context, falling back to SystemUserID for actor attribution")
	return SystemUserID
}
