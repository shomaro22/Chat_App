package usecase

import (
	"bytes"
	"context"
	"time"

	"chat-app/entity"
	"chat-app/repository"

	"github.com/google/uuid"
)

type MessageReadUseCase struct {
	Messages repository.MessageRepository
	Reads    repository.MessageReadRepository
	Members  repository.RoomMemberRepository
}

func NewMessageReadUseCase(messages repository.MessageRepository, reads repository.MessageReadRepository, members repository.RoomMemberRepository) *MessageReadUseCase {
	return &MessageReadUseCase{
		Messages: messages,
		Reads:    reads,
		Members:  members,
	}
}

type MarkAsReadResult struct {
	Updated           bool
	LastReadMessageID uuid.UUID
}

func (u *MessageReadUseCase) MarkAsRead(ctx context.Context, userID, roomID, messageID uuid.UUID) (*MarkAsReadResult, error) {
	if userID == uuid.Nil {
		return nil, ErrInvalidMessageReadUserID
	}
	if roomID == uuid.Nil {
		return nil, ErrInvalidMessageReadRoomID
	}
	if messageID == uuid.Nil {
		return nil, ErrInvalidReadMessageID
	}

	ok, err := u.Members.IsActiveMember(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrReadNotRoomMember
	}

	target, err := u.Messages.FindByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, ErrReadMessageNotFound
	}
	if target.RoomID != roomID {
		return nil, ErrReadMessageRoomMismatch
	}

	existing, err := u.Reads.FindByRoomAndUser(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		now := time.Now().UTC()
		row := &entity.MessageRead{
			ID:                uuid.New(),
			RoomID:            roomID,
			UserID:            userID,
			LastReadMessageID: &messageID,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		if err := u.Reads.Create(ctx, row); err != nil {
			return nil, err
		}
		return &MarkAsReadResult{Updated: true, LastReadMessageID: messageID}, nil
	}

	if existing.LastReadMessageID == nil {
		if err := u.Reads.UpdateLastReadMessage(ctx, roomID, userID, messageID); err != nil {
			return nil, err
		}
		return &MarkAsReadResult{Updated: true, LastReadMessageID: messageID}, nil
	}

	current, err := u.Messages.FindByID(ctx, *existing.LastReadMessageID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		if err := u.Reads.UpdateLastReadMessage(ctx, roomID, userID, messageID); err != nil {
			return nil, err
		}
		return &MarkAsReadResult{Updated: true, LastReadMessageID: messageID}, nil
	}

	if !messageIsStrictlyAfter(target, current) {
		return &MarkAsReadResult{Updated: false, LastReadMessageID: uuid.Nil}, nil
	}

	if err := u.Reads.UpdateLastReadMessage(ctx, roomID, userID, messageID); err != nil {
		return nil, err
	}
	return &MarkAsReadResult{Updated: true, LastReadMessageID: messageID}, nil
}

func messageIsStrictlyAfter(a, b *entity.Message) bool {
	if a.CreatedAt.After(b.CreatedAt) {
		return true
	}
	if a.CreatedAt.Before(b.CreatedAt) {
		return false
	}
	return bytes.Compare(a.ID[:], b.ID[:]) > 0
}
