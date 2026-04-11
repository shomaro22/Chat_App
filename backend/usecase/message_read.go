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
	Rooms    repository.RoomRepository
	Users    repository.UserRepository
}

func NewMessageReadUseCase(messages repository.MessageRepository, reads repository.MessageReadRepository, rooms repository.RoomRepository, users repository.UserRepository) *MessageReadUseCase {
	return &MessageReadUseCase{
		Messages: messages,
		Reads:    reads,
		Rooms:    rooms,
		Users:    users,
	}
}

// MarkAsReadResult は既読更新の結果（更新の有無と最終既読メッセージ ID）。
type MarkAsReadResult struct {
	Updated           bool
	LastReadMessageID uuid.UUID
}

// MarkAsRead はメンバー検証後、最終既読メッセージを更新または新規作成する。
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

	roomRow, err := readRoomForActiveMember(ctx, u.Rooms, roomID, userID)
	if err != nil {
		return nil, err
	}
	if roomRow.FrozenAt != nil {
		return nil, ErrRoomFrozen
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
			ReadAt:            &now,
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

// MemberReadBrief はルーム内メンバーの最終既読位置（集計 API 用）。
type MemberReadBrief struct {
	UserID            uuid.UUID
	UserName          string
	LastReadMessageID *uuid.UUID
}

// RoomReadState はルームの未読数・自分の既読・メンバー別既読概要。
type RoomReadState struct {
	RoomID              uuid.UUID
	MyLastReadMessageID *uuid.UUID
	UnreadCount         int64
	MemberReads         []MemberReadBrief
}

// GetRoomReadState は一覧・詳細向けの既読／未読集計を返す。
func (u *MessageReadUseCase) GetRoomReadState(ctx context.Context, userID, roomID uuid.UUID) (*RoomReadState, error) {
	if userID == uuid.Nil {
		return nil, ErrInvalidMessageReadUserID
	}
	if roomID == uuid.Nil {
		return nil, ErrInvalidMessageReadRoomID
	}
	if _, err := readRoomForActiveMember(ctx, u.Rooms, roomID, userID); err != nil {
		return nil, err
	}
	readRow, err := u.Reads.FindByRoomAndUser(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	var lastRead *uuid.UUID
	if readRow != nil {
		lastRead = readRow.LastReadMessageID
	}
	unread, err := u.Messages.CountUnreadForViewer(ctx, roomID, userID, lastRead)
	if err != nil {
		return nil, err
	}
	rows, err := u.Reads.ListByRoomForActiveMembers(ctx, roomID)
	if err != nil {
		return nil, err
	}
	readUserIDs := make([]uuid.UUID, 0, len(rows))
	for i := range rows {
		readUserIDs = append(readUserIDs, rows[i].UserID)
	}
	var readUsers []entity.User
	if len(readUserIDs) > 0 {
		readUsers, err = u.Users.FindByIDs(ctx, readUserIDs)
		if err != nil {
			return nil, err
		}
	}
	nameBy := make(map[uuid.UUID]string, len(readUsers))
	for i := range readUsers {
		nameBy[readUsers[i].ID] = readUsers[i].Name
	}
	members := make([]MemberReadBrief, 0, len(rows))
	for i := range rows {
		members = append(members, MemberReadBrief{
			UserID:            rows[i].UserID,
			UserName:          nameBy[rows[i].UserID],
			LastReadMessageID: rows[i].LastReadMessageID,
		})
	}
	return &RoomReadState{
		RoomID:              roomID,
		MyLastReadMessageID: lastRead,
		UnreadCount:         unread,
		MemberReads:         members,
	}, nil
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
