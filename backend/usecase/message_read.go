package usecase

import (
	"bytes"
	"context"
	"time"

	"chat-app/entity"
	"chat-app/repository"

	"github.com/google/uuid"
)

// MessageReadUseCase updates per-room last-read positions (forward-only).
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

// MarkAsRead advances the user's read cursor to messageID when it is strictly after the current position (by created_at, then id). Backward or equal positions succeed without updating.
func (u *MessageReadUseCase) MarkAsRead(ctx context.Context, userID, roomID, messageID uuid.UUID) error {
	if userID == uuid.Nil {
		return ErrInvalidMessageReadUserID
	}
	if roomID == uuid.Nil {
		return ErrInvalidMessageReadRoomID
	}
	if messageID == uuid.Nil {
		return ErrInvalidReadMessageID
	}

	ok, err := u.Members.IsActiveMember(ctx, roomID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrReadNotRoomMember
	}

	target, err := u.Messages.FindByID(ctx, messageID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrReadMessageNotFound
	}
	if target.RoomID != roomID {
		return ErrReadMessageRoomMismatch
	}

	existing, err := u.Reads.FindByRoomAndUser(ctx, roomID, userID)
	if err != nil {
		return err
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
		return u.Reads.Create(ctx, row)
	}

	if existing.LastReadMessageID == nil {
		return u.Reads.UpdateLastReadMessage(ctx, roomID, userID, messageID)
	}

	current, err := u.Messages.FindByID(ctx, *existing.LastReadMessageID)
	if err != nil {
		return err
	}
	if current == nil {
		return u.Reads.UpdateLastReadMessage(ctx, roomID, userID, messageID)
	}

	if !messageIsStrictlyAfter(target, current) {
		return nil
	}

	return u.Reads.UpdateLastReadMessage(ctx, roomID, userID, messageID)
}

// messageIsStrictlyAfter compares by (created_at ASC, id ASC) — UUID id order is not chronological; use byte order only as tie-breaker when timestamps match.
func messageIsStrictlyAfter(a, b *entity.Message) bool {
	if a.CreatedAt.After(b.CreatedAt) {
		return true
	}
	if a.CreatedAt.Before(b.CreatedAt) {
		return false
	}
	return bytes.Compare(a.ID[:], b.ID[:]) > 0
}
