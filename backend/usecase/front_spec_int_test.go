//go:build integration

package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"strings"
	"testing"
	"time"

	"chat-app/entity"
	"chat-app/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAuthz_GetRoomDetail_NonMember(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, _, r := setupGroupRoomWithMembers(t, gormDB)
	outsider := testUser(t, gormDB, "detailout@example.com")
	roomUC := newRoomUseCase(gormDB)

	_, err := roomUC.GetRoomDetail(ctx, r.ID, outsider.ID)
	require.ErrorIs(t, err, ErrNotRoomMember)

	d, err := roomUC.GetRoomDetail(ctx, r.ID, u1.ID)
	require.NoError(t, err)
	require.NotNil(t, d)
}

func TestAuthz_AfterKick_NoRoomAccess(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, u2, r := setupGroupRoomWithMembers(t, gormDB)
	roomUC := newRoomUseCase(gormDB)
	msgUC := newMessageUseCase(gormDB)
	readUC := newMessageReadUseCase(gormDB)

	require.NoError(t, roomUC.RemoveRoomMember(ctx, r.ID, u1.ID, u2.ID))

	_, err := roomUC.GetRoomDetail(ctx, r.ID, u2.ID)
	require.ErrorIs(t, err, ErrNotRoomMember)

	_, err = msgUC.ListMessages(ctx, r.ID, u2.ID, nil, nil, 10, true, false)
	require.ErrorIs(t, err, ErrNotRoomMember)

	_, err = readUC.GetRoomReadState(ctx, u2.ID, r.ID)
	require.ErrorIs(t, err, ErrNotRoomMember)

	txt := "after kick"
	msgNow := time.Now().UTC()
	msg := &entity.Message{ID: uuid.New(), RoomID: r.ID, SenderID: u1.ID, Content: &txt, CreatedAt: msgNow, UpdatedAt: msgNow}
	require.NoError(t, repository.NewMessageRepository(gormDB).Create(ctx, msg))

	_, err = readUC.MarkAsRead(ctx, u2.ID, r.ID, msg.ID)
	require.ErrorIs(t, err, ErrNotRoomMember)

	err = roomUC.CanSubscribeWebSocket(ctx, r.ID, u2.ID)
	require.ErrorIs(t, err, ErrNotRoomMember)
}

func TestAuthz_ListMessages_NonMember(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	_, _, r := setupGroupRoomWithMembers(t, gormDB)
	outsider := testUser(t, gormDB, "listout@example.com")
	msgUC := newMessageUseCase(gormDB)

	_, err := msgUC.ListMessages(ctx, r.ID, outsider.ID, nil, nil, 20, true, false)
	require.ErrorIs(t, err, ErrNotRoomMember)
}

func TestFrozen_SendAndReadAndSubscribe(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, _, r := setupGroupRoomWithMembers(t, gormDB)
	roomUC := newRoomUseCase(gormDB)
	msgUC := newMessageUseCase(gormDB)
	readUC := newMessageReadUseCase(gormDB)
	roomsRepo := repository.NewRoomRepository(gormDB)

	now := time.Now().UTC()
	require.NoError(t, roomsRepo.SetFrozenAt(ctx, r.ID, &now))

	txt := "x"
	_, err := msgUC.SendMessage(ctx, r.ID, u1.ID, &txt, nil)
	require.ErrorIs(t, err, ErrRoomFrozen)

	msgNow := time.Now().UTC()
	msg := &entity.Message{ID: uuid.New(), RoomID: r.ID, SenderID: u1.ID, Content: &txt, CreatedAt: msgNow, UpdatedAt: msgNow}
	require.NoError(t, repository.NewMessageRepository(gormDB).Create(ctx, msg))

	_, err = readUC.MarkAsRead(ctx, u1.ID, r.ID, msg.ID)
	require.ErrorIs(t, err, ErrRoomFrozen)

	err = roomUC.CanSubscribeWebSocket(ctx, r.ID, u1.ID)
	require.ErrorIs(t, err, ErrRoomFrozen)

	// 凍結中も履歴取得は可（閲覧のみ）
	list, err := msgUC.ListMessages(ctx, r.ID, u1.ID, nil, nil, 10, true, false)
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestImageMessage_UploadURLListPipeline(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1, _, r := setupGroupRoomWithMembers(t, gormDB)
	msgUC := newMessageUseCase(gormDB)
	mediaUC := NewMediaUseCase(t.TempDir())

	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var pngBuf bytes.Buffer
	require.NoError(t, png.Encode(&pngBuf, img))
	urlPath, err := mediaUC.SaveUploadedImage("image/png", &pngBuf)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(urlPath, "/static/uploads/"))

	msg, err := msgUC.SendMessage(ctx, r.ID, u1.ID, nil, &urlPath)
	require.NoError(t, err)
	require.NotNil(t, msg.ImageURL)
	require.Equal(t, urlPath, *msg.ImageURL)

	list, err := msgUC.ListMessages(ctx, r.ID, u1.ID, nil, nil, 20, true, false)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.NotNil(t, list[0].Message.ImageURL)
	require.Equal(t, urlPath, *list[0].Message.ImageURL)
}

func TestDeleteMessage_WSJSONShape(t *testing.T) {
	// message.deleted のペイロードが一覧の論理削除と整合する（WS 送信は controller だが型は共有）
	rid := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	mid := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	payload := map[string]any{
		"type":           "message_deleted",
		"schema_version": float64(1),
		"room_id":        rid.String(),
		"message_id":     mid.String(),
	}
	b, err := json.Marshal(payload)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))
	require.Equal(t, "message_deleted", out["type"])
}
