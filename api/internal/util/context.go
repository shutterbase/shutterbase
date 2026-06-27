package util

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
)

type userContextKey struct{}

type realUserContextKey struct{}

// UserKey is the context key under which the auth middleware stores the
// effective user (impersonated if active, else real — S8).
var UserKey = userContextKey{}

// RealUserKey is the context key under which the impersonation middleware stores
// the REAL logged-in user (S8). When impersonation is active it differs from the
// effective user under UserKey; otherwise it holds the same user.
var RealUserKey = realUserContextKey{}

// GetUser returns the effective user from the context, or nil if unauthenticated.
func GetUser(ctx context.Context) *ent.User {
	if user, ok := ctx.Value(UserKey).(*ent.User); ok {
		return user
	}
	return nil
}

// GetRealUser returns the real logged-in user, regardless of any active
// impersonation. Falls back to the effective user when no real user was stored
// (paths that never impersonate, e.g. background jobs).
func GetRealUser(ctx context.Context) *ent.User {
	if user, ok := ctx.Value(RealUserKey).(*ent.User); ok {
		return user
	}
	return GetUser(ctx)
}

// GetImpersonator returns the real admin IFF impersonation is active this
// request (effective user differs from the real user), else nil.
func GetImpersonator(ctx context.Context) *ent.User {
	real := GetRealUser(ctx)
	eff := GetUser(ctx)
	if real != nil && eff != nil && real.ID != eff.ID {
		return real
	}
	return nil
}

// GetImpersonatorID returns the real admin's id and true when impersonation is
// active, so CreateAuditLog can record impersonatedBy alongside actor=effective.
func GetImpersonatorID(ctx context.Context) (uuid.UUID, bool) {
	if imp := GetImpersonator(ctx); imp != nil {
		return imp.ID, true
	}
	return uuid.Nil, false
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
