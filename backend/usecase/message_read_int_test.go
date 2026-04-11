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
		repository.NewRoomRepository(gormDB),
		repository.NewUserRepository(gormDB),
	)
}

func TestMessageReadUseCase_NotMember(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, _, r := setupGroupRoomWithMembers(t, gormDB)
	outsider := testUser(t, gormDB, "readout@example.com")
	uc := newMessageReadUseCase(gormDB)
	txt := "m"
	msgNow := time.Now().UTC()
	msg := &entity.Message{ID: uuid.New(), RoomID: r.ID, SenderID: u1.ID, Content: &txt, CreatedAt: msgNow, UpdatedAt: msgNow}
	require.NoError(t, repository.NewMessageRepository(gormDB).Create(ctx, msg))

	_, err := uc.MarkAsRead(ctx, outsider.ID, r.ID, msg.ID)
	require.ErrorIs(t, err, ErrReadNotRoomMember)
}

func TestMessageReadUseCase_GetRoomReadState_NotMember(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, _, r := setupGroupRoomWithMembers(t, gormDB)
	outsider := testUser(t, gormDB, "readstateout@example.com")
	uc := newMessageReadUseCase(gormDB)

	_, err := uc.GetRoomReadState(ctx, outsider.ID, r.ID)
	require.ErrorIs(t, err, ErrReadNotRoomMember)
	_, err = uc.GetRoomReadState(ctx, u1.ID, r.ID)
	require.NoError(t, err)
}

func TestMessageReadUseCase_MessageNotFound(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, _, r := setupGroupRoomWithMembers(t, gormDB)
	uc := newMessageReadUseCase(gormDB)

	_, err := uc.MarkAsRead(ctx, u1.ID, r.ID, uuid.New())
	require.ErrorIs(t, err, ErrReadMessageNotFound)
}

func TestMessageReadUseCase_RoomMismatch(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, u2, r1 := setupGroupRoomWithMembers(t, gormDB)
	r2Now := time.Now().UTC()
	r2 := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("R2"), CreatedBy: u1.ID, CreatedAt: r2Now, UpdatedAt: r2Now}
	require.NoError(t, repository.NewRoomRepository(gormDB).Create(ctx, r2))
	members := repository.NewRoomMemberRepository(gormDB)
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: r2.ID, UserID: u1.ID, JoinedAt: r2Now}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: r2.ID, UserID: u2.ID, JoinedAt: r2Now}))

	txt := "x"
	msgT := time.Now().UTC()
	msg := &entity.Message{ID: uuid.New(), RoomID: r2.ID, SenderID: u1.ID, Content: &txt, CreatedAt: msgT, UpdatedAt: msgT}
	require.NoError(t, repository.NewMessageRepository(gormDB).Create(ctx, msg))

	uc := newMessageReadUseCase(gormDB)
	_, err := uc.MarkAsRead(ctx, u1.ID, r1.ID, msg.ID)
	require.ErrorIs(t, err, ErrReadMessageRoomMismatch)
}

func TestMessageReadUseCase_FirstRead(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, _, r := setupGroupRoomWithMembers(t, gormDB)
	uc := newMessageReadUseCase(gormDB)
	txt := "a"
	tA := time.Now().UTC()
	msg := &entity.Message{ID: uuid.New(), RoomID: r.ID, SenderID: u1.ID, Content: &txt, CreatedAt: tA, UpdatedAt: tA}
	require.NoError(t, repository.NewMessageRepository(gormDB).Create(ctx, msg))

	res, err := uc.MarkAsRead(ctx, u1.ID, r.ID, msg.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.True(t, res.Updated)
	require.Equal(t, msg.ID, res.LastReadMessageID)
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
	t1 := time.Now().UTC()
	m1 := &entity.Message{ID: uuid.New(), RoomID: r.ID, SenderID: u1.ID, Content: &a, CreatedAt: t1, UpdatedAt: t1}
	require.NoError(t, msgs.Create(ctx, m1))
	time.Sleep(5 * time.Millisecond)
	t2 := time.Now().UTC()
	m2 := &entity.Message{ID: uuid.New(), RoomID: r.ID, SenderID: u2.ID, Content: &b, CreatedAt: t2, UpdatedAt: t2}
	require.NoError(t, msgs.Create(ctx, m2))
	time.Sleep(5 * time.Millisecond)
	t3 := time.Now().UTC()
	m3 := &entity.Message{ID: uuid.New(), RoomID: r.ID, SenderID: u1.ID, Content: &c, CreatedAt: t3, UpdatedAt: t3}
	require.NoError(t, msgs.Create(ctx, m3))

	res1, err := uc.MarkAsRead(ctx, u1.ID, r.ID, m1.ID)
	require.NoError(t, err)
	require.True(t, res1.Updated)
	res2, err := uc.MarkAsRead(ctx, u1.ID, r.ID, m3.ID)
	require.NoError(t, err)
	require.True(t, res2.Updated)
	require.Equal(t, m3.ID, res2.LastReadMessageID)
	got, err := repository.NewMessageReadRepository(gormDB).FindByRoomAndUser(ctx, r.ID, u1.ID)
	require.NoError(t, err)
	require.Equal(t, m3.ID, *got.LastReadMessageID)

	res3, err := uc.MarkAsRead(ctx, u1.ID, r.ID, m3.ID)
	require.NoError(t, err)
	require.NotNil(t, res3)
	require.False(t, res3.Updated)
	res4, err := uc.MarkAsRead(ctx, u1.ID, r.ID, m2.ID)
	require.NoError(t, err)
	require.False(t, res4.Updated)
	got2, err := repository.NewMessageReadRepository(gormDB).FindByRoomAndUser(ctx, r.ID, u1.ID)
	require.NoError(t, err)
	require.Equal(t, m3.ID, *got2.LastReadMessageID)
}
