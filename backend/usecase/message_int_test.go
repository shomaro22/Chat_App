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

type messageEventRecorder struct {
	created   []*entity.Message
	updated   []*entity.Message
	deleted   []struct {
		roomID    uuid.UUID
		messageID uuid.UUID
	}
}

func (r *messageEventRecorder) OnMessageCreated(_ context.Context, msg *entity.Message) {
	if msg == nil {
		return
	}
	cp := *msg
	r.created = append(r.created, &cp)
}

func (r *messageEventRecorder) OnMessageUpdated(_ context.Context, msg *entity.Message) {
	if msg == nil {
		return
	}
	cp := *msg
	r.updated = append(r.updated, &cp)
}

func (r *messageEventRecorder) OnMessageDeleted(_ context.Context, roomID, messageID, _ uuid.UUID) {
	r.deleted = append(r.deleted, struct {
		roomID    uuid.UUID
		messageID uuid.UUID
	}{
		roomID:    roomID,
		messageID: messageID,
	})
}

func newMessageUseCase(gormDB *gorm.DB) *MessageUseCase {
	return NewMessageUseCase(
		repository.NewMessageRepository(gormDB),
		repository.NewUserRepository(gormDB),
		repository.NewRoomRepository(gormDB),
		nil,
		nil,
	)
}

func setupGroupRoomWithMembers(t *testing.T, gormDB *gorm.DB) (*entity.User, *entity.User, *entity.Room) {
	t.Helper()
	ctx := context.Background()
	u1 := testUser(t, gormDB, "msgu1@example.com")
	u2 := testUser(t, gormDB, "msgu2@example.com")
	rooms := repository.NewRoomRepository(gormDB)
	members := repository.NewRoomMemberRepository(gormDB)
	rNow := time.Now().UTC()
	r := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("G"), CreatedBy: u1.ID, CreatedAt: rNow, UpdatedAt: rNow}
	require.NoError(t, rooms.Create(ctx, r))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: r.ID, UserID: u1.ID, JoinedAt: rNow}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: r.ID, UserID: u2.ID, JoinedAt: rNow}))
	return u1, u2, r
}

func TestMessageUseCase_SendMessage_EmptyRejected(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, _, r := setupGroupRoomWithMembers(t, gormDB)
	uc := newMessageUseCase(gormDB)

	_, err := uc.SendMessage(ctx, r.ID, u1.ID, strPtr("   "), nil)
	require.ErrorIs(t, err, ErrEmptyMessage)

	_, err = uc.SendMessage(ctx, r.ID, u1.ID, nil, nil)
	require.ErrorIs(t, err, ErrEmptyMessage)
}

func TestMessageUseCase_SendMessage_NotMember(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	_, _, r := setupGroupRoomWithMembers(t, gormDB)
	outsider := testUser(t, gormDB, "out@example.com")
	uc := newMessageUseCase(gormDB)

	_, err := uc.SendMessage(ctx, r.ID, outsider.ID, strPtr("hi"), nil)
	require.ErrorIs(t, err, ErrNotRoomMember)
}

func TestMessageUseCase_SendMessage_OK(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, _, r := setupGroupRoomWithMembers(t, gormDB)
	uc := newMessageUseCase(gormDB)

	got, err := uc.SendMessage(ctx, r.ID, u1.ID, strPtr("hello"), nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, r.ID, got.RoomID)
	require.Equal(t, u1.ID, got.SenderID)
	require.NotNil(t, got.Content)
	require.Equal(t, "hello", *got.Content)
	require.Nil(t, got.ImageURL)
	require.False(t, got.IsDeleted)

	img := "/static/uploads/e2e-test-image.png"
	got2, err := uc.SendMessage(ctx, r.ID, u1.ID, nil, &img)
	require.NoError(t, err)
	require.Nil(t, got2.Content)
	require.NotNil(t, got2.ImageURL)
	require.Equal(t, img, *got2.ImageURL)
}

func TestMessageUseCase_ListMessages_OK(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, u2, r := setupGroupRoomWithMembers(t, gormDB)
	uc := newMessageUseCase(gormDB)

	m1, err := uc.SendMessage(ctx, r.ID, u1.ID, strPtr("first"), nil)
	require.NoError(t, err)
	time.Sleep(5 * time.Millisecond)
	m2, err := uc.SendMessage(ctx, r.ID, u2.ID, strPtr("second"), nil)
	require.NoError(t, err)

	list, err := uc.ListMessages(ctx, r.ID, u1.ID, nil, nil, 10, true, false)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, m2.ID, list[0].Message.ID)
	require.Equal(t, m1.ID, list[1].Message.ID)

	page2, err := uc.ListMessages(ctx, r.ID, u1.ID, &m2.ID, nil, 10, true, false)
	require.NoError(t, err)
	require.Len(t, page2, 1)
	require.Equal(t, m1.ID, page2[0].Message.ID)
}

func TestMessageUseCase_ListMessages_DeletedIncluded(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, _, r := setupGroupRoomWithMembers(t, gormDB)
	uc := newMessageUseCase(gormDB)
	msgRepo := repository.NewMessageRepository(gormDB)

	m, err := uc.SendMessage(ctx, r.ID, u1.ID, strPtr("x"), nil)
	require.NoError(t, err)
	require.NoError(t, msgRepo.SoftDelete(ctx, m.ID, time.Now().UTC()))

	list, err := uc.ListMessages(ctx, r.ID, u1.ID, nil, nil, 10, true, false)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.True(t, list[0].Message.IsDeleted)
}

func TestMessageUseCase_ListMessages_NotMember(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	_, _, r := setupGroupRoomWithMembers(t, gormDB)
	outsider := testUser(t, gormDB, "outl@example.com")
	uc := newMessageUseCase(gormDB)

	_, err := uc.ListMessages(ctx, r.ID, outsider.ID, nil, nil, 10, true, false)
	require.ErrorIs(t, err, ErrNotRoomMember)
}

func TestMessageUseCase_DeleteMessage_Forbidden(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, u2, r := setupGroupRoomWithMembers(t, gormDB)
	uc := newMessageUseCase(gormDB)

	m, err := uc.SendMessage(ctx, r.ID, u1.ID, strPtr("mine"), nil)
	require.NoError(t, err)

	_, err = uc.DeleteMessage(ctx, m.ID, u2.ID)
	require.ErrorIs(t, err, ErrForbiddenMessageEdit)
}

func TestMessageUseCase_DeleteMessage_OK(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, _, r := setupGroupRoomWithMembers(t, gormDB)
	uc := newMessageUseCase(gormDB)

	m, err := uc.SendMessage(ctx, r.ID, u1.ID, strPtr("delme"), nil)
	require.NoError(t, err)

	_, err = uc.DeleteMessage(ctx, m.ID, u1.ID)
	require.NoError(t, err)

	again, err := repository.NewMessageRepository(gormDB).FindByID(ctx, m.ID)
	require.NoError(t, err)
	require.NotNil(t, again)
	require.True(t, again.IsDeleted)
	require.NotNil(t, again.DeletedAt)
}

func TestMessageUseCase_DeleteMessage_Idempotent(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, _, r := setupGroupRoomWithMembers(t, gormDB)
	uc := newMessageUseCase(gormDB)

	m, err := uc.SendMessage(ctx, r.ID, u1.ID, strPtr("x"), nil)
	require.NoError(t, err)
	_, err = uc.DeleteMessage(ctx, m.ID, u1.ID)
	require.NoError(t, err)
	_, err = uc.DeleteMessage(ctx, m.ID, u1.ID)
	require.NoError(t, err)
}

func TestMessageUseCase_DeleteMessage_NotFound(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, _, _ := setupGroupRoomWithMembers(t, gormDB)
	uc := newMessageUseCase(gormDB)

	_, err := uc.DeleteMessage(ctx, uuid.New(), u1.ID)
	require.ErrorIs(t, err, ErrMessageNotFound)
}

func TestMessageUseCase_Events_TriggeredOnCreateEditDelete(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, _, r := setupGroupRoomWithMembers(t, gormDB)
	rec := &messageEventRecorder{}
	uc := newMessageUseCase(gormDB).WithEventHook(rec)

	msg, err := uc.SendMessage(ctx, r.ID, u1.ID, strPtr("hello"), nil)
	require.NoError(t, err)
	require.Len(t, rec.created, 1)
	require.Equal(t, msg.ID, rec.created[0].ID)
	require.Empty(t, rec.updated)
	require.Empty(t, rec.deleted)

	edited, err := uc.EditMessage(ctx, msg.ID, u1.ID, strPtr("edited"))
	require.NoError(t, err)
	require.Len(t, rec.updated, 1)
	require.Equal(t, edited.ID, rec.updated[0].ID)
	require.NotNil(t, rec.updated[0].Content)
	require.Equal(t, "edited", *rec.updated[0].Content)
	require.Empty(t, rec.deleted)

	roomID, err := uc.DeleteMessage(ctx, msg.ID, u1.ID)
	require.NoError(t, err)
	require.Equal(t, r.ID, roomID)
	require.Len(t, rec.deleted, 1)
	require.Equal(t, r.ID, rec.deleted[0].roomID)
	require.Equal(t, msg.ID, rec.deleted[0].messageID)
}

func TestMessageUseCase_Events_NotTriggeredOnFailure(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, u2, r := setupGroupRoomWithMembers(t, gormDB)
	rec := &messageEventRecorder{}
	uc := newMessageUseCase(gormDB).WithEventHook(rec)

	_, err := uc.SendMessage(ctx, r.ID, u1.ID, strPtr("   "), nil)
	require.ErrorIs(t, err, ErrEmptyMessage)
	require.Empty(t, rec.created)
	require.Empty(t, rec.updated)
	require.Empty(t, rec.deleted)

	msg, err := uc.SendMessage(ctx, r.ID, u1.ID, strPtr("mine"), nil)
	require.NoError(t, err)
	require.Len(t, rec.created, 1)
	require.Empty(t, rec.updated)
	require.Empty(t, rec.deleted)

	_, err = uc.EditMessage(ctx, msg.ID, u2.ID, strPtr("hijack"))
	require.ErrorIs(t, err, ErrForbiddenMessageEdit)
	require.Len(t, rec.created, 1)
	require.Empty(t, rec.updated)
	require.Empty(t, rec.deleted)

	_, err = uc.DeleteMessage(ctx, msg.ID, u2.ID)
	require.ErrorIs(t, err, ErrForbiddenMessageEdit)
	require.Len(t, rec.created, 1)
	require.Empty(t, rec.updated)
	require.Empty(t, rec.deleted)
}
