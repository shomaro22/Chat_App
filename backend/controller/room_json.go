package controller

import (
	"time"

	"chat-app/entity"
	"chat-app/usecase"

	"github.com/google/uuid"
)

// --- 作成・更新リクエスト ---

type createDirectRoomRequest struct {
	TargetUserID string `json:"target_user_id"`
}

type createGroupRoomRequest struct {
	Name      string   `json:"name"`
	MemberIDs []string `json:"member_ids"`
}

type patchGroupRoomRequest struct {
	Name string `json:"name"`
}

type addRoomMemberRequest struct {
	MemberID string `json:"member_id"`
}

// --- ルーム本体 ---

type roomJSON struct {
	ID            uuid.UUID  `json:"id"`
	Type          string     `json:"type"`
	Name          *string    `json:"name,omitempty"`
	DirectUserAID *uuid.UUID `json:"direct_user_a_id,omitempty"`
	DirectUserBID *uuid.UUID `json:"direct_user_b_id,omitempty"`
	CreatedBy     uuid.UUID  `json:"created_by"`
	CreatedAt     string     `json:"created_at"`
	UpdatedAt     string     `json:"updated_at"`
	IsFrozen      bool       `json:"is_frozen"`
	FrozenAt      *string    `json:"frozen_at,omitempty"`
}

func roomToJSON(r *entity.Room) roomJSON {
	j := roomJSON{
		ID:            r.ID,
		Type:          string(r.Type),
		Name:          r.Name,
		DirectUserAID: r.DirectUserAID,
		DirectUserBID: r.DirectUserBID,
		CreatedBy:     r.CreatedBy,
		CreatedAt:     r.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:     r.UpdatedAt.UTC().Format(time.RFC3339Nano),
		IsFrozen:      r.FrozenAt != nil,
	}
	if r.FrozenAt != nil {
		s := r.FrozenAt.UTC().Format(time.RFC3339Nano)
		j.FrozenAt = &s
	}
	return j
}

// --- ルーム詳細 ---

type roomMemberJSON struct {
	UserID   uuid.UUID `json:"user_id"`
	Name     string    `json:"name"`
	JoinedAt string    `json:"joined_at"`
	Online   bool      `json:"online"`
}

type roomDetailResponse struct {
	Room                roomJSON         `json:"room"`
	Members             []roomMemberJSON `json:"members"`
	DirectPeer          *roomMemberJSON  `json:"direct_peer,omitempty"`
	MyLastReadMessageID *uuid.UUID       `json:"my_last_read_message_id,omitempty"`
	UnreadCount         int64            `json:"unread_count"`
}

func roomDetailToJSON(d *usecase.RoomDetail) roomDetailResponse {
	if d == nil {
		return roomDetailResponse{}
	}
	members := make([]roomMemberJSON, 0, len(d.Members))
	for _, m := range d.Members {
		members = append(members, roomMemberJSON{
			UserID:   m.UserID,
			Name:     m.Name,
			JoinedAt: m.JoinedAt.UTC().Format(time.RFC3339Nano),
			Online:   m.Online,
		})
	}
	var peer *roomMemberJSON
	if d.DirectPeer != nil {
		p := roomMemberJSON{
			UserID:   d.DirectPeer.UserID,
			Name:     d.DirectPeer.Name,
			JoinedAt: d.DirectPeer.JoinedAt.UTC().Format(time.RFC3339Nano),
			Online:   d.DirectPeer.Online,
		}
		peer = &p
	}
	return roomDetailResponse{
		Room:                roomToJSON(&d.Room),
		Members:             members,
		DirectPeer:          peer,
		MyLastReadMessageID: d.MyLastReadMessageID,
		UnreadCount:         d.UnreadCount,
	}
}

// --- ルーム一覧（付加情報付き） ---

type lastMessageSummaryJSON struct {
	ID         uuid.UUID `json:"id"`
	SenderID   uuid.UUID `json:"sender_id"`
	SenderName string    `json:"sender_name,omitempty"`
	Preview    *string   `json:"preview,omitempty"`
	CreatedAt  string    `json:"created_at"`
	IsDeleted  bool      `json:"is_deleted"`
}

type roomListItemJSON struct {
	roomJSON
	LastMessage    *lastMessageSummaryJSON `json:"last_message,omitempty"`
	UnreadCount    int64                   `json:"unread_count"`
	PeerOnline     *bool                   `json:"peer_online,omitempty"`
	DirectPeerID   *uuid.UUID              `json:"direct_peer_id,omitempty"`
	DirectPeerName *string                 `json:"direct_peer_name,omitempty"`
}

type listRoomsEnrichedResponse struct {
	Rooms  []roomListItemJSON `json:"rooms"`
	Sort   string             `json:"sort"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}

func joinedRoomSummaryToJSON(s usecase.JoinedRoomSummary) roomListItemJSON {
	item := roomListItemJSON{
		roomJSON:     roomToJSON(&s.Room),
		UnreadCount:  s.UnreadCount,
		PeerOnline:   s.PeerOnline,
		DirectPeerID: s.DirectPeerID,
	}
	item.DirectPeerName = s.DirectPeerName
	if s.LastMessage != nil {
		lm := lastMessageSummaryJSON{
			ID:        s.LastMessage.ID,
			SenderID:  s.LastMessage.SenderID,
			Preview:   s.LastPreview,
			CreatedAt: s.LastMessage.CreatedAt.UTC().Format(time.RFC3339Nano),
			IsDeleted: s.LastMessage.IsDeleted || s.LastMessage.DeletedAt != nil,
		}
		if s.LastMessageSenderName != nil {
			lm.SenderName = *s.LastMessageSenderName
		}
		item.LastMessage = &lm
	}
	return item
}
