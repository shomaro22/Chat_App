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
	"gorm.io/gorm"
)

func newMessageReadUseCase(gormDB *gorm.DB) *MessageReadUseCase {
	return NewMessageReadUseCase(
		repository.NewMessageRepository(gormDB),
		repository.NewMessageReadRepository(gormDB),
		repository.NewRoomMemberRepository(gormDB),
	)
}

func TestMessageReadUseCase_NotMember(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, _, r := setupGroupRoomWithMembers(t, gormDB)
	outsider := testUser(t, gormDB, "readout@example.com")
	uc := newMessageReadUseCase(gormDB)
	txt := "m"
	msg := &entity.Message{ID: uuid.New(), RoomID: r.ID, SenderID: u1.ID, Content: &txt}
	require.NoError(t, repository.NewMessageRepository(gormDB).Create(ctx, msg))

	err := uc.MarkAsRead(ctx, outsider.ID, r.ID, msg.ID)
	require.ErrorIs(t, err, ErrReadNotRoomMember)
}

func TestMessageReadUseCase_MessageNotFound(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, _, r := setupGroupRoomWithMembers(t, gormDB)
	uc := newMessageReadUseCase(gormDB)

	err := uc.MarkAsRead(ctx, u1.ID, r.ID, uuid.New())
	require.ErrorIs(t, err, ErrReadMessageNotFound)
}

func TestMessageReadUseCase_RoomMismatch(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, u2, r1 := setupGroupRoomWithMembers(t, gormDB)
	r2 := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("R2"), CreatedBy: u1.ID}
	require.NoError(t, repository.NewRoomRepository(gormDB).Create(ctx, r2))
	members := repository.NewRoomMemberRepository(gormDB)
	require.NoError(t, members.AddMember(ctx, r2.ID, u1.ID))
	require.NoError(t, members.AddMember(ctx, r2.ID, u2.ID))

	txt := "x"
	msg := &entity.Message{ID: uuid.New(), RoomID: r2.ID, SenderID: u1.ID, Content: &txt}
	require.NoError(t, repository.NewMessageRepository(gormDB).Create(ctx, msg))

	uc := newMessageReadUseCase(gormDB)
	err := uc.MarkAsRead(ctx, u1.ID, r1.ID, msg.ID)
	require.ErrorIs(t, err, ErrReadMessageRoomMismatch)
}

func TestMessageReadUseCase_FirstRead(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, _, r := setupGroupRoomWithMembers(t, gormDB)
	uc := newMessageReadUseCase(gormDB)
	txt := "a"
	msg := &entity.Message{ID: uuid.New(), RoomID: r.ID, SenderID: u1.ID, Content: &txt}
	require.NoError(t, repository.NewMessageRepository(gormDB).Create(ctx, msg))

	require.NoError(t, uc.MarkAsRead(ctx, u1.ID, r.ID, msg.ID))
	got, err := repository.NewMessageReadRepository(gormDB).FindByRoomAndUser(ctx, r.ID, u1.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, msg.ID, *got.LastReadMessageID)
}

func TestMessageReadUseCase_ForwardThenIdempotent(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, u2, r := setupGroupRoomWithMembers(t, gormDB)
	uc := newMessageReadUseCase(gormDB)
	msgs := repository.NewMessageRepository(gormDB)
	a, b, c := "a", "b", "c"
	m1 := &entity.Message{ID: uuid.New(), RoomID: r.ID, SenderID: u1.ID, Content: &a}
	require.NoError(t, msgs.Create(ctx, m1))
	time.Sleep(5 * time.Millisecond)
	m2 := &entity.Message{ID: uuid.New(), RoomID: r.ID, SenderID: u2.ID, Content: &b}
	require.NoError(t, msgs.Create(ctx, m2))
	time.Sleep(5 * time.Millisecond)
	m3 := &entity.Message{ID: uuid.New(), RoomID: r.ID, SenderID: u1.ID, Content: &c}
	require.NoError(t, msgs.Create(ctx, m3))

	require.NoError(t, uc.MarkAsRead(ctx, u1.ID, r.ID, m1.ID))
	require.NoError(t, uc.MarkAsRead(ctx, u1.ID, r.ID, m3.ID))
	got, err := repository.NewMessageReadRepository(gormDB).FindByRoomAndUser(ctx, r.ID, u1.ID)
	require.NoError(t, err)
	require.Equal(t, m3.ID, *got.LastReadMessageID)

	require.NoError(t, uc.MarkAsRead(ctx, u1.ID, r.ID, m3.ID))
	require.NoError(t, uc.MarkAsRead(ctx, u1.ID, r.ID, m2.ID))
	got2, err := repository.NewMessageReadRepository(gormDB).FindByRoomAndUser(ctx, r.ID, u1.ID)
	require.NoError(t, err)
	require.Equal(t, m3.ID, *got2.LastReadMessageID)
}
