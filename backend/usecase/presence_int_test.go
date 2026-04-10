//go:build integration

package usecase

import (
	"context"
	"testing"
	"time"

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

	prNow := time.Now().UTC()
	r1 := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("A"), CreatedBy: u1.ID, CreatedAt: prNow, UpdatedAt: prNow}
	r2 := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("B"), CreatedBy: u1.ID, CreatedAt: prNow, UpdatedAt: prNow}
	require.NoError(t, rooms.Create(ctx, r1))
	require.NoError(t, rooms.Create(ctx, r2))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: r1.ID, UserID: u1.ID, JoinedAt: prNow}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: r2.ID, UserID: u1.ID, JoinedAt: prNow}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: r2.ID, UserID: u2.ID, JoinedAt: prNow}))

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
	xNow := time.Now().UTC()
	r := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("X"), CreatedBy: u1.ID, CreatedAt: xNow, UpdatedAt: xNow}
	require.NoError(t, rooms.Create(ctx, r))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: r.ID, UserID: u1.ID, JoinedAt: xNow}))

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
