package usecase

import (
	"context"
	"strings"
	"time"

	"chat-app/entity"
	"chat-app/repository"

	"github.com/google/uuid"
)

const (
	// DefaultMessageListLimit は limit 省略時のメッセージ取得件数。
	DefaultMessageListLimit = 50
	// MaxMessageListLimit は limit の上限。
	MaxMessageListLimit = 100
)

// ListedMessage は一覧 API 向け（送信者名・任意で既読数）。
type ListedMessage struct {
	Message    entity.Message
	SenderName string
	ReadCount  *int64
}

type MessageUseCase struct {
	MessageRepo repository.MessageRepository
	Users       repository.UserRepository
	Rooms       repository.RoomRepository
}

func NewMessageUseCase(messages repository.MessageRepository, users repository.UserRepository, rooms repository.RoomRepository) *MessageUseCase {
	return &MessageUseCase{
		MessageRepo: messages,
		Users:       users,
		Rooms:       rooms,
	}
}

func normalizeOptionalString(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// SendMessage はメンバー検証のうえでメッセージを保存する。
func (u *MessageUseCase) SendMessage(ctx context.Context, roomID, senderID uuid.UUID, content, imageURL *string) (*entity.Message, error) {
	if roomID == uuid.Nil {
		return nil, ErrInvalidRoomID
	}
	if senderID == uuid.Nil {
		return nil, ErrInvalidSenderID
	}
	content = normalizeOptionalString(content)
	imageURL = normalizeOptionalString(imageURL)
	if content == nil && imageURL == nil {
		return nil, ErrEmptyMessage
	}
	roomRow, err := RoomForActiveMember(ctx, u.Rooms, roomID, senderID)
	if err != nil {
		return nil, err
	}
	if roomRow.FrozenAt != nil {
		return nil, ErrRoomFrozen
	}
	now := time.Now().UTC()
	msg := &entity.Message{
		ID:        uuid.New(),
		RoomID:    roomID,
		SenderID:  senderID,
		Content:   content,
		ImageURL:  imageURL,
		IsDeleted: false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := u.MessageRepo.Create(ctx, msg); err != nil {
		return nil, err
	}
	if u.Rooms != nil {
		_ = u.Rooms.UpdateUpdatedAt(ctx, roomID, now)
	}
	return msg, nil
}

// ListMessages は参加メンバーのみメッセージ履歴を取得する。order_desc=false で昇順（古い順）。
func (u *MessageUseCase) ListMessages(ctx context.Context, roomID, userID uuid.UUID, beforeID, afterID *uuid.UUID, limit int, orderDesc, includeReadCount bool) ([]ListedMessage, error) {
	if roomID == uuid.Nil {
		return nil, ErrInvalidRoomID
	}
	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	if beforeID != nil && *beforeID == uuid.Nil {
		beforeID = nil
	}
	if afterID != nil && *afterID == uuid.Nil {
		afterID = nil
	}
	limit = normalizeMessageListLimit(limit)
	if _, err := RoomForActiveMember(ctx, u.Rooms, roomID, userID); err != nil {
		return nil, err
	}
	rows, err := u.MessageRepo.ListByRoomID(ctx, roomID, beforeID, afterID, limit, orderDesc)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	senderIDs := make([]uuid.UUID, 0, len(rows))
	seen := make(map[uuid.UUID]struct{})
	for i := range rows {
		id := rows[i].SenderID
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		senderIDs = append(senderIDs, id)
	}
	var users []entity.User
	if u.Users != nil {
		users, err = u.Users.FindByIDs(ctx, senderIDs)
		if err != nil {
			return nil, err
		}
	}
	nameBy := make(map[uuid.UUID]string, len(users))
	for i := range users {
		nameBy[users[i].ID] = users[i].Name
	}
	out := make([]ListedMessage, 0, len(rows))
	for i := range rows {
		item := ListedMessage{
			Message:    rows[i],
			SenderName: nameBy[rows[i].SenderID],
		}
		if includeReadCount {
			n, err := u.MessageRepo.CountReadersAtOrAfterMessage(ctx, roomID, rows[i].ID, rows[i].SenderID)
			if err != nil {
				return nil, err
			}
			v := n
			item.ReadCount = &v
		}
		out = append(out, item)
	}
	return out, nil
}

func normalizeMessageListLimit(limit int) int {
	if limit <= 0 {
		return DefaultMessageListLimit
	}
	if limit > MaxMessageListLimit {
		return MaxMessageListLimit
	}
	return limit
}

// DeleteMessage は送信者本人によるメッセージの論理削除を行う。戻り値の room_id は WS 通知用（既に削除済みの場合もルーム ID を返す）。
func (u *MessageUseCase) DeleteMessage(ctx context.Context, messageID, userID uuid.UUID) (uuid.UUID, error) {
	if messageID == uuid.Nil {
		return uuid.Nil, ErrInvalidMessageID
	}
	if userID == uuid.Nil {
		return uuid.Nil, ErrInvalidUserID
	}
	msg, err := u.MessageRepo.FindByID(ctx, messageID)
	if err != nil {
		return uuid.Nil, err
	}
	if msg == nil {
		return uuid.Nil, ErrMessageNotFound
	}
	if msg.IsDeleted || msg.DeletedAt != nil {
		return msg.RoomID, nil
	}
	if msg.SenderID != userID {
		return uuid.Nil, ErrForbiddenMessageEdit
	}
	roomRow, err := RoomForActiveMember(ctx, u.Rooms, msg.RoomID, userID)
	if err != nil {
		return uuid.Nil, err
	}
	if roomRow.FrozenAt != nil {
		return uuid.Nil, ErrRoomFrozen
	}
	if err := u.MessageRepo.SoftDelete(ctx, messageID, time.Now().UTC()); err != nil {
		return uuid.Nil, err
	}
	return msg.RoomID, nil
}
