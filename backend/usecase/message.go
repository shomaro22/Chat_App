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
	defaultMessageListLimit = 50
	maxMessageListLimit     = 100
)

type MessageUseCase struct {
	MessageRepo    repository.MessageRepository
	RoomMemberRepo repository.RoomMemberRepository
}

func NewMessageUseCase(messages repository.MessageRepository, members repository.RoomMemberRepository) *MessageUseCase {
	return &MessageUseCase{
		MessageRepo:    messages,
		RoomMemberRepo: members,
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
	ok, err := u.RoomMemberRepo.IsActiveMember(ctx, roomID, senderID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotRoomMember
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
	return msg, nil
}

func (u *MessageUseCase) ListMessages(ctx context.Context, roomID, userID uuid.UUID, beforeID *uuid.UUID, limit int) ([]entity.Message, error) {
	if roomID == uuid.Nil {
		return nil, ErrInvalidRoomID
	}
	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	if beforeID != nil && *beforeID == uuid.Nil {
		beforeID = nil
	}
	limit = normalizeMessageListLimit(limit)
	ok, err := u.RoomMemberRepo.IsActiveMember(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotRoomMember
	}
	return u.MessageRepo.ListByRoomID(ctx, roomID, beforeID, limit)
}

func normalizeMessageListLimit(limit int) int {
	if limit <= 0 {
		return defaultMessageListLimit
	}
	if limit > maxMessageListLimit {
		return maxMessageListLimit
	}
	return limit
}

func (u *MessageUseCase) DeleteMessage(ctx context.Context, messageID, userID uuid.UUID) error {
	if messageID == uuid.Nil {
		return ErrInvalidMessageID
	}
	if userID == uuid.Nil {
		return ErrInvalidUserID
	}
	msg, err := u.MessageRepo.FindByID(ctx, messageID)
	if err != nil {
		return err
	}
	if msg == nil {
		return ErrMessageNotFound
	}
	if msg.IsDeleted || msg.DeletedAt != nil {
		return nil
	}
	if msg.SenderID != userID {
		return ErrForbiddenMessageEdit
	}
	return u.MessageRepo.SoftDelete(ctx, messageID, time.Now().UTC())
}
