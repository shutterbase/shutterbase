package util

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/shutterbase/shutterbase/ent"
)

// S8: GetRealUser/GetImpersonator resolve the real-vs-effective identity.
func TestImpersonationContextResolution(t *testing.T) {
	admin := &ent.User{ID: uuid.New()}
	viewer := &ent.User{ID: uuid.New()}

	// No user at all.
	bare := context.Background()
	assert.Nil(t, GetUser(bare))
	assert.Nil(t, GetRealUser(bare))
	assert.Nil(t, GetImpersonator(bare))
	_, ok := GetImpersonatorID(bare)
	assert.False(t, ok)

	// Plain auth (no impersonation): effective == real, no RealUserKey stored.
	plain := context.WithValue(context.Background(), UserKey, viewer)
	assert.Equal(t, viewer, GetUser(plain))
	assert.Equal(t, viewer, GetRealUser(plain), "GetRealUser falls back to effective")
	assert.Nil(t, GetImpersonator(plain), "no impersonation when effective == real")
	_, ok = GetImpersonatorID(plain)
	assert.False(t, ok)

	// Impersonation active: effective=viewer, real=admin.
	imp := context.WithValue(context.WithValue(context.Background(), UserKey, viewer), RealUserKey, admin)
	assert.Equal(t, viewer, GetUser(imp), "effective is the impersonated target")
	assert.Equal(t, admin, GetRealUser(imp))
	assert.Equal(t, admin, GetImpersonator(imp), "impersonator is the real admin")
	id, ok := GetImpersonatorID(imp)
	assert.True(t, ok)
	assert.Equal(t, admin.ID, id)

	// Same user under both keys (defensive): not treated as impersonation.
	same := context.WithValue(context.WithValue(context.Background(), UserKey, admin), RealUserKey, admin)
	assert.Nil(t, GetImpersonator(same))
}
