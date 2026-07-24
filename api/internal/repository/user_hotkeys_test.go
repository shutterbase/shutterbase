package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/schema"
	"github.com/shutterbase/shutterbase/internal/repository"
)

func TestUserHotkeysRoundtrip(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)

	u, err := repo.CreateUser(ctx, &repository.CreateUserParameters{
		Username:  "hotkey-user",
		FirstName: "Hot",
		LastName:  "Key",
	})
	require.NoError(t, err)
	assert.Nil(t, u.Hotkeys)

	hotkeys := &schema.UserHotkeys{
		Bindings:    map[string][]string{"images.next-image": {"n"}, "images.toggle-view": {}},
		TagBindings: map[string]string{"p": "review", "shift+1": "podium"},
	}
	updated, err := repo.UpdateUser(ctx, u.ID, &repository.UpdateUserParameters{Hotkeys: hotkeys})
	require.NoError(t, err)
	require.NotNil(t, updated.Hotkeys)
	assert.Equal(t, hotkeys.Bindings, updated.Hotkeys.Bindings)
	assert.Equal(t, hotkeys.TagBindings, updated.Hotkeys.TagBindings)

	reloaded, err := repo.GetUser(ctx, u.ID)
	require.NoError(t, err)
	require.NotNil(t, reloaded.Hotkeys)
	assert.Equal(t, hotkeys.Bindings, reloaded.Hotkeys.Bindings)
	assert.Equal(t, hotkeys.TagBindings, reloaded.Hotkeys.TagBindings)

	// identical payload is a no-op (rolled back, updatedAt untouched)
	same, err := repo.UpdateUser(ctx, u.ID, &repository.UpdateUserParameters{Hotkeys: hotkeys})
	require.NoError(t, err)
	assert.Equal(t, reloaded.UpdatedAt, same.UpdatedAt)
}
