package controller

import (
	"context"

	"chat-app/entity"
	"chat-app/usecase"

	"github.com/google/uuid"
)

// --- Room ---

type roomUseCaseAdapter struct{ inner *usecase.RoomUseCase }

// NewRoomUseCaseAdapter は *usecase.RoomUseCase を controller.RoomUseCase としてラップする。
func NewRoomUseCaseAdapter(u *usecase.RoomUseCase) RoomUseCase {
	return &roomUseCaseAdapter{inner: u}
}

func (a *roomUseCaseAdapter) GetJoinedRooms(ctx context.Context, userID uuid.UUID) ([]entity.Room, error) {
	return a.inner.GetJoinedRooms(ctx, userID)
}

func (a *roomUseCaseAdapter) GetJoinedRoomSummaries(ctx context.Context, userID uuid.UUID, limit, offset int, updatedDesc bool) ([]JoinedRoomSummary, error) {
	rows, err := a.inner.GetJoinedRoomSummaries(ctx, userID, limit, offset, updatedDesc)
	if err != nil {
		return nil, err
	}
	out := make([]JoinedRoomSummary, len(rows))
	for i := range rows {
		out[i] = JoinedRoomSummary{
			Room:                  rows[i].Room,
			LastMessage:           rows[i].LastMessage,
			LastPreview:           rows[i].LastPreview,
			LastMessageSenderName: rows[i].LastMessageSenderName,
			UnreadCount:           rows[i].UnreadCount,
			PeerOnline:            rows[i].PeerOnline,
			DirectPeerName:        rows[i].DirectPeerName,
			DirectPeerID:          rows[i].DirectPeerID,
		}
	}
	return out, nil
}

func (a *roomUseCaseAdapter) GetRoomDetail(ctx context.Context, roomID, requesterID uuid.UUID) (*RoomDetail, error) {
	d, err := a.inner.GetRoomDetail(ctx, roomID, requesterID)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, nil
	}
	members := make([]RoomMemberBrief, len(d.Members))
	for i := range d.Members {
		members[i] = RoomMemberBrief{
			UserID:   d.Members[i].UserID,
			Name:     d.Members[i].Name,
			JoinedAt: d.Members[i].JoinedAt,
			Online:   d.Members[i].Online,
		}
	}
	var peer *RoomMemberBrief
	if d.DirectPeer != nil {
		p := RoomMemberBrief{
			UserID:   d.DirectPeer.UserID,
			Name:     d.DirectPeer.Name,
			JoinedAt: d.DirectPeer.JoinedAt,
			Online:   d.DirectPeer.Online,
		}
		peer = &p
	}
	return &RoomDetail{
		Room:                d.Room,
		Members:             members,
		DirectPeer:          peer,
		MyLastReadMessageID: d.MyLastReadMessageID,
		UnreadCount:         d.UnreadCount,
	}, nil
}

func (a *roomUseCaseAdapter) AddGroupMember(ctx context.Context, roomID, actorID, newUserID uuid.UUID) error {
	return a.inner.AddGroupMember(ctx, roomID, actorID, newUserID)
}

func (a *roomUseCaseAdapter) RemoveRoomMember(ctx context.Context, roomID, actorID, targetUserID uuid.UUID) error {
	return a.inner.RemoveRoomMember(ctx, roomID, actorID, targetUserID)
}

func (a *roomUseCaseAdapter) CreateOrGetDirectRoom(ctx context.Context, userID, targetUserID uuid.UUID) (*entity.Room, error) {
	return a.inner.CreateOrGetDirectRoom(ctx, userID, targetUserID)
}

func (a *roomUseCaseAdapter) CreateGroupRoom(ctx context.Context, createdBy uuid.UUID, name string, memberIDs []uuid.UUID) (*entity.Room, error) {
	return a.inner.CreateGroupRoom(ctx, createdBy, name, memberIDs)
}

func (a *roomUseCaseAdapter) ResolveMemberRef(ctx context.Context, ref string) (uuid.UUID, error) {
	return a.inner.ResolveMemberRef(ctx, ref)
}

func (a *roomUseCaseAdapter) UpdateGroupName(ctx context.Context, roomID, actorID uuid.UUID, name string) error {
	return a.inner.UpdateGroupName(ctx, roomID, actorID, name)
}

func (a *roomUseCaseAdapter) CloseRoom(ctx context.Context, roomID, actorID uuid.UUID) error {
	return a.inner.CloseRoom(ctx, roomID, actorID)
}

func (a *roomUseCaseAdapter) IsActiveRoomMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	return a.inner.IsActiveRoomMember(ctx, roomID, userID)
}

func (a *roomUseCaseAdapter) CanSubscribeWebSocket(ctx context.Context, roomID, userID uuid.UUID) error {
	return a.inner.CanSubscribeWebSocket(ctx, roomID, userID)
}

// --- Message ---

type messageUseCaseAdapter struct{ inner *usecase.MessageUseCase }

// NewMessageUseCaseAdapter は *usecase.MessageUseCase を controller.MessageUseCase としてラップする。
func NewMessageUseCaseAdapter(u *usecase.MessageUseCase) MessageUseCase {
	return &messageUseCaseAdapter{inner: u}
}

func (a *messageUseCaseAdapter) SendMessage(ctx context.Context, roomID, senderID uuid.UUID, content, imageURL *string) (*entity.Message, error) {
	return a.inner.SendMessage(ctx, roomID, senderID, content, imageURL)
}

func (a *messageUseCaseAdapter) ListMessages(ctx context.Context, roomID, userID uuid.UUID, beforeID, afterID *uuid.UUID, limit int, orderDesc, includeReadCount bool) ([]ListedMessage, error) {
	rows, err := a.inner.ListMessages(ctx, roomID, userID, beforeID, afterID, limit, orderDesc, includeReadCount)
	if err != nil {
		return nil, err
	}
	out := make([]ListedMessage, len(rows))
	for i := range rows {
		out[i] = ListedMessage{
			Message:    rows[i].Message,
			SenderName: rows[i].SenderName,
			ReadCount:  rows[i].ReadCount,
		}
	}
	return out, nil
}

func (a *messageUseCaseAdapter) DeleteMessage(ctx context.Context, messageID, userID uuid.UUID) (uuid.UUID, error) {
	return a.inner.DeleteMessage(ctx, messageID, userID)
}

// --- Message read ---

type messageReadUseCaseAdapter struct{ inner *usecase.MessageReadUseCase }

// NewMessageReadUseCaseAdapter は *usecase.MessageReadUseCase を controller.MessageReadUseCase としてラップする。
func NewMessageReadUseCaseAdapter(u *usecase.MessageReadUseCase) MessageReadUseCase {
	return &messageReadUseCaseAdapter{inner: u}
}

func (a *messageReadUseCaseAdapter) MarkAsRead(ctx context.Context, userID, roomID, messageID uuid.UUID) (*MarkAsReadResult, error) {
	r, err := a.inner.MarkAsRead(ctx, userID, roomID, messageID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, nil
	}
	return &MarkAsReadResult{Updated: r.Updated, LastReadMessageID: r.LastReadMessageID}, nil
}

func (a *messageReadUseCaseAdapter) GetRoomReadState(ctx context.Context, userID, roomID uuid.UUID) (*RoomReadState, error) {
	s, err := a.inner.GetRoomReadState(ctx, userID, roomID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	members := make([]MemberReadBrief, len(s.MemberReads))
	for i := range s.MemberReads {
		members[i] = MemberReadBrief{
			UserID:            s.MemberReads[i].UserID,
			UserName:          s.MemberReads[i].UserName,
			LastReadMessageID: s.MemberReads[i].LastReadMessageID,
		}
	}
	return &RoomReadState{
		RoomID:              s.RoomID,
		MyLastReadMessageID: s.MyLastReadMessageID,
		UnreadCount:         s.UnreadCount,
		MemberReads:         members,
	}, nil
}
