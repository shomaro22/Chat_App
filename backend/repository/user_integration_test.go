//go:build integration

package repository

import (
	"context"
	"testing"

	"chat-app/entity"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_FindByIDs(t *testing.T) {
	ctx := context.Background()
	gormDB := setupDB(t)
	users := NewUserRepository(gormDB)

	a := &entity.User{ID: uuid.New(), Name: "a", Email: "fa@example.com", PasswordHash: "x", Role: entity.UserRoleUser, Active: true}
	b := &entity.User{ID: uuid.New(), Name: "b", Email: "fb@example.com", PasswordHash: "x", Role: entity.UserRoleUser, Active: true}
	require.NoError(t, users.Create(ctx, a))
	require.NoError(t, users.Create(ctx, b))

	got, err := users.FindByIDs(ctx, []uuid.UUID{a.ID, b.ID, a.ID, uuid.Nil})
	require.NoError(t, err)
	require.Len(t, got, 2)
	ids := map[uuid.UUID]struct{}{got[0].ID: {}, got[1].ID: {}}
	_, okA := ids[a.ID]
	_, okB := ids[b.ID]
	require.True(t, okA && okB)

	partial, err := users.FindByIDs(ctx, []uuid.UUID{a.ID, uuid.New()})
	require.NoError(t, err)
	require.Len(t, partial, 1)
}
