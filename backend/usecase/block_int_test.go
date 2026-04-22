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

func TestBlockFlow_HideMessagesAndUnreadFromBlocker(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)

	users := repository.NewUserRepository(gormDB)
	rooms := repository.NewRoomRepository(gormDB)
	members := repository.NewRoomMemberRepository(gormDB)
	msgRepo := repository.NewMessageRepository(gormDB)
	readRepo := repository.NewMessageReadRepository(gormDB)
	blockRepo := repository.NewUserBlockRepository(gormDB)
	roomTx := repository.NewRoomUnitOfWork(gormDB)

	u1 := testUser(t, gormDB, "blk-a@example.com")
	u2 := testUser(t, gormDB, "blk-b@example.com")

	// Build direct room and join both users.
	pa, pb := entity.OrderedUserPair(u1.ID, u2.ID)
	now := time.Now().UTC()
	dm := &entity.Room{
		ID:            uuid.New(),
		Type:          entity.RoomTypeDirect,
		CreatedBy:     u1.ID,
		DirectUserAID: &pa,
		DirectUserBID: &pb,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, rooms.Create(ctx, dm))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: dm.ID, UserID: u1.ID, JoinedAt: now}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: dm.ID, UserID: u2.ID, JoinedAt: now}))

	msgUC := NewMessageUseCase(msgRepo, users, rooms, nil, blockRepo)
	readUC := NewMessageReadUseCase(msgRepo, readRepo, rooms, users, blockRepo)
	roomUC := NewRoomUseCase(roomTx, users, rooms, members, msgRepo, readRepo, nil, blockRepo)
	socialUC := NewSocialUseCase(msgRepo, nil, nil, nil, rooms, blockRepo)

	require.NoError(t, socialUC.BlockUser(ctx, u1.ID, u2.ID))

	_, err := msgUC.SendMessage(ctx, dm.ID, u2.ID, strPtr("from blocked user"), nil)
	require.NoError(t, err)
	_, err = msgUC.SendMessage(ctx, dm.ID, u1.ID, strPtr("from blocker"), nil)
	require.NoError(t, err)

	// Blocker should not see blocked user's messages.
	listForU1, err := msgUC.ListMessages(ctx, dm.ID, u1.ID, nil, nil, 50, true, false)
	require.NoError(t, err)
	require.Len(t, listForU1, 1)
	require.Equal(t, u1.ID, listForU1[0].Message.SenderID)

	// Blocked user still sees both messages.
	listForU2, err := msgUC.ListMessages(ctx, dm.ID, u2.ID, nil, nil, 50, true, false)
	require.NoError(t, err)
	require.Len(t, listForU2, 2)

	stateU1, err := readUC.GetRoomReadState(ctx, u1.ID, dm.ID)
	require.NoError(t, err)
	require.Equal(t, int64(0), stateU1.UnreadCount, "blocked user's message must not increment unread")

	sumsU1, err := roomUC.GetJoinedRoomSummaries(ctx, u1.ID, 20, 0, true)
	require.NoError(t, err)
	require.Len(t, sumsU1, 1)
	require.Equal(t, int64(0), sumsU1[0].UnreadCount)
	require.NotNil(t, sumsU1[0].LastMessage)
	require.Equal(t, u1.ID, sumsU1[0].LastMessage.SenderID)
}

func TestBlockFlow_SocialUseCaseBlockState(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)

	u1 := testUser(t, gormDB, "blk-state-a@example.com")
	u2 := testUser(t, gormDB, "blk-state-b@example.com")

	socialUC := NewSocialUseCase(
		repository.NewMessageRepository(gormDB),
		nil,
		nil,
		nil,
		repository.NewRoomRepository(gormDB),
		repository.NewUserBlockRepository(gormDB),
	)

	require.NoError(t, socialUC.BlockUser(ctx, u1.ID, u2.ID))
	blocked, err := socialUC.IsBlockedUser(ctx, u1.ID, u2.ID)
	require.NoError(t, err)
	require.True(t, blocked)

	require.NoError(t, socialUC.UnblockUser(ctx, u1.ID, u2.ID))
	blocked, err = socialUC.IsBlockedUser(ctx, u1.ID, u2.ID)
	require.NoError(t, err)
	require.False(t, blocked)
}
