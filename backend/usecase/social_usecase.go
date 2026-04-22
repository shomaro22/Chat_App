package usecase

import (
	"context"
	"strings"
	"unicode/utf8"

	"chat-app/entity"
	"chat-app/repository"

	"github.com/google/uuid"
)

// SocialUseCase はリアクション・ピン・ブックマーク。
type SocialUseCase struct {
	Messages  repository.MessageRepository
	Reactions repository.MessageReactionRepository
	Pins      repository.RoomPinRepository
	Bookmarks repository.UserBookmarkRepository
	Rooms     repository.RoomRepository
}

func NewSocialUseCase(
	msg repository.MessageRepository,
	re repository.MessageReactionRepository,
	pins repository.RoomPinRepository,
	bm repository.UserBookmarkRepository,
	rooms repository.RoomRepository,
) *SocialUseCase {
	return &SocialUseCase{
		Messages:  msg,
		Reactions: re,
		Pins:      pins,
		Bookmarks: bm,
		Rooms:     rooms,
	}
}

func normalizeEmoji(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" || utf8.RuneCountInString(s) > 32 {
		return "", ErrInvalidEmoji
	}
	return s, nil
}

func (s *SocialUseCase) AddReaction(ctx context.Context, roomID, userID, messageID uuid.UUID, emoji string) error {
	if _, err := RoomForActiveMemberWritable(ctx, s.Rooms, roomID, userID); err != nil {
		return err
	}
	m, err := s.Messages.FindByID(ctx, messageID)
	if err != nil {
		return err
	}
	if m == nil || m.RoomID != roomID || m.IsDeleted || m.DeletedAt != nil {
		return ErrMessageNotFound
	}
	emoji, err = normalizeEmoji(emoji)
	if err != nil {
		return err
	}
	return s.Reactions.Add(ctx, messageID, userID, emoji)
}

func (s *SocialUseCase) RemoveReaction(ctx context.Context, roomID, userID, messageID uuid.UUID, emoji string) error {
	if _, err := RoomForActiveMemberWritable(ctx, s.Rooms, roomID, userID); err != nil {
		return err
	}
	emoji, err := normalizeEmoji(emoji)
	if err != nil {
		return err
	}
	return s.Reactions.Remove(ctx, messageID, userID, emoji)
}

func (s *SocialUseCase) ListReactions(ctx context.Context, roomID, userID, messageID uuid.UUID) ([]repository.MessageReaction, error) {
	if _, err := RoomForActiveMember(ctx, s.Rooms, roomID, userID); err != nil {
		return nil, err
	}
	m, err := s.Messages.FindByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if m == nil || m.RoomID != roomID {
		return nil, ErrMessageNotFound
	}
	return s.Reactions.ListByMessage(ctx, messageID)
}

func (s *SocialUseCase) SetPin(ctx context.Context, roomID, actorID, messageID uuid.UUID) error {
	if _, err := RoomForActiveMemberWritable(ctx, s.Rooms, roomID, actorID); err != nil {
		return err
	}
	m, err := s.Messages.FindByID(ctx, messageID)
	if err != nil {
		return err
	}
	if m == nil || m.RoomID != roomID || m.IsDeleted || m.DeletedAt != nil {
		return ErrMessageNotFound
	}
	return s.Pins.Upsert(ctx, roomID, messageID, actorID)
}

func (s *SocialUseCase) ClearPin(ctx context.Context, roomID, actorID uuid.UUID) error {
	if _, err := RoomForActiveMemberWritable(ctx, s.Rooms, roomID, actorID); err != nil {
		return err
	}
	return s.Pins.Delete(ctx, roomID)
}

func (s *SocialUseCase) GetPin(ctx context.Context, roomID, userID uuid.UUID) (*repository.RoomPin, error) {
	if _, err := RoomForActiveMember(ctx, s.Rooms, roomID, userID); err != nil {
		return nil, err
	}
	return s.Pins.FindByRoom(ctx, roomID)
}

func (s *SocialUseCase) AddBookmark(ctx context.Context, userID, messageID uuid.UUID) error {
	m, err := s.Messages.FindByID(ctx, messageID)
	if err != nil {
		return err
	}
	if m == nil || m.IsDeleted || m.DeletedAt != nil {
		return ErrMessageNotFound
	}
	if _, err := RoomForActiveMember(ctx, s.Rooms, m.RoomID, userID); err != nil {
		return err
	}
	return s.Bookmarks.Add(ctx, userID, messageID, m.RoomID)
}

func (s *SocialUseCase) RemoveBookmark(ctx context.Context, userID, messageID uuid.UUID) error {
	return s.Bookmarks.Remove(ctx, userID, messageID)
}

func (s *SocialUseCase) ListBookmarks(ctx context.Context, userID uuid.UUID, limit, offset int) ([]repository.UserMessageBookmark, int64, error) {
	if userID == uuid.Nil {
		return nil, 0, ErrInvalidUserID
	}
	return s.Bookmarks.ListByUser(ctx, userID, limit, offset)
}

// BookmarkWithMessage はブックマーク一覧用。
type BookmarkWithMessage struct {
	Bookmark repository.UserMessageBookmark
	Message  entity.Message
}

func (s *SocialUseCase) ListBookmarksWithMessages(ctx context.Context, userID uuid.UUID, limit, offset int) ([]BookmarkWithMessage, int64, error) {
	rows, total, err := s.ListBookmarks(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]BookmarkWithMessage, 0, len(rows))
	for i := range rows {
		m, err := s.Messages.FindByID(ctx, rows[i].MessageID)
		if err != nil {
			return nil, 0, err
		}
		if m == nil {
			continue
		}
		out = append(out, BookmarkWithMessage{Bookmark: rows[i], Message: *m})
	}
	return out, total, nil
}
