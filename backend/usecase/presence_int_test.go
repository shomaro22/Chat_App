//go:build integration

package usecase

import (
	"context"
	"testing"

	"chat-app/entity"
	"chat-app/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPresenceUseCase_JoinedRoomIDs_ActiveMembersOnly(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1 := testUser(t, gormDB, "p1@example.com")
	u2 := testUser(t, gormDB, "p2@example.com")
	rooms := repository.NewRoomRepository(gormDB)
	members := repository.NewRoomMemberRepository(gormDB)

	r1 := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("A"), CreatedBy: u1.ID}
	r2 := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("B"), CreatedBy: u1.ID}
	require.NoError(t, rooms.Create(ctx, r1))
	require.NoError(t, rooms.Create(ctx, r2))
	require.NoError(t, members.AddMember(ctx, r1.ID, u1.ID))
	require.NoError(t, members.AddMember(ctx, r2.ID, u1.ID))
	require.NoError(t, members.AddMember(ctx, r2.ID, u2.ID))

	uc := NewPresenceUseCase(rooms)
	ids, err := uc.JoinedRoomIDs(ctx, u1.ID)
	require.NoError(t, err)
	require.Len(t, ids, 2)

	require.NoError(t, members.Leave(ctx, r1.ID, u1.ID))
	ids2, err := uc.JoinedRoomIDs(ctx, u1.ID)
	require.NoError(t, err)
	require.Len(t, ids2, 1)
	require.Equal(t, r2.ID, ids2[0])
}

func TestPresenceUseCase_JoinedRoomIDs_ExcludesNonMember(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1 := testUser(t, gormDB, "pnm@example.com")
	rooms := repository.NewRoomRepository(gormDB)
	members := repository.NewRoomMemberRepository(gormDB)
	r := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("X"), CreatedBy: u1.ID}
	require.NoError(t, rooms.Create(ctx, r))
	require.NoError(t, members.AddMember(ctx, r.ID, u1.ID))

	uc := NewPresenceUseCase(rooms)
	outsider := testUser(t, gormDB, "out@example.com")
	ids, err := uc.JoinedRoomIDs(ctx, outsider.ID)
	require.NoError(t, err)
	require.Empty(t, ids)
}

func TestPresenceUseCase_JoinedRoomIDs_RepositoryError(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	sqlDB, err := gormDB.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	uc := NewPresenceUseCase(repository.NewRoomRepository(gormDB))
	_, err = uc.JoinedRoomIDs(ctx, uuid.New())
	require.Error(t, err)
}
