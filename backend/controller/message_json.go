package controller

import (
	"time"

	"chat-app/entity"
	"chat-app/usecase"

	"github.com/google/uuid"
)

// --- メッセージ CRUD / 一覧 ---

type createMessageRequest struct {
	Content  *string `json:"content"`
	ImageURL *string `json:"image_url"`
}

type createMessageResponse struct {
	Message messageJSON `json:"message"`
}

type listMessagesResponse struct {
	Messages []listedMessageJSON `json:"messages"`
	Order    string              `json:"order"`
}

type listedMessageJSON struct {
	messageJSON
	SenderName string `json:"sender_name"`
	ReadCount  *int64 `json:"read_count,omitempty"`
}

type messageJSON struct {
	ID        uuid.UUID `json:"id"`
	RoomID    uuid.UUID `json:"room_id"`
	SenderID  uuid.UUID `json:"sender_id"`
	Content   *string   `json:"content"`
	ImageURL  *string   `json:"image_url"`
	DeletedAt *string   `json:"deleted_at"`
	CreatedAt string    `json:"created_at"`
}

// messageDeletedWSBroadcast は message_deleted の WS イベント用。
type messageDeletedWSBroadcast struct {
	Type      string    `json:"type"`
	RoomID    uuid.UUID `json:"room_id"`
	MessageID uuid.UUID `json:"message_id"`
}

// messageCreatedWSBroadcast は WebSocket で送る message_created イベント用 JSON。
type messageCreatedWSBroadcast struct {
	Type    string      `json:"type"`
	RoomID  uuid.UUID   `json:"room_id"`
	Message messageJSON `json:"message"`
}

func listedMessageToJSON(m usecase.ListedMessage) listedMessageJSON {
	return listedMessageJSON{
		messageJSON: messageToJSON(&m.Message),
		SenderName:  m.SenderName,
		ReadCount:   m.ReadCount,
	}
}

func messageToJSON(m *entity.Message) messageJSON {
	var deleted *string
	if m.DeletedAt != nil {
		s := m.DeletedAt.UTC().Format(time.RFC3339Nano)
		deleted = &s
	}
	return messageJSON{
		ID:        m.ID,
		RoomID:    m.RoomID,
		SenderID:  m.SenderID,
		Content:   copyStrPtr(m.Content),
		ImageURL:  copyStrPtr(m.ImageURL),
		DeletedAt: deleted,
		CreatedAt: m.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func copyStrPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := *s
	return &v
}

// --- 既読 ---

type markAsReadRequest struct {
	MessageID string `json:"message_id"`
}

// readUpdatedPayload は WebSocket read_updated イベント用 JSON。
type readUpdatedPayload struct {
	Type              string    `json:"type"`
	RoomID            uuid.UUID `json:"room_id"`
	UserID            uuid.UUID `json:"user_id"`
	LastReadMessageID uuid.UUID `json:"last_read_message_id"`
}

type memberReadBriefJSON struct {
	UserID            uuid.UUID  `json:"user_id"`
	LastReadMessageID *uuid.UUID `json:"last_read_message_id,omitempty"`
}

type roomReadStateJSON struct {
	RoomID              uuid.UUID             `json:"room_id"`
	MyLastReadMessageID *uuid.UUID            `json:"my_last_read_message_id,omitempty"`
	UnreadCount         int64                 `json:"unread_count"`
	MemberReads         []memberReadBriefJSON `json:"member_reads"`
}

func roomReadStateToJSON(s *usecase.RoomReadState) roomReadStateJSON {
	if s == nil {
		return roomReadStateJSON{}
	}
	members := make([]memberReadBriefJSON, 0, len(s.MemberReads))
	for _, m := range s.MemberReads {
		members = append(members, memberReadBriefJSON{
			UserID:            m.UserID,
			LastReadMessageID: m.LastReadMessageID,
		})
	}
	return roomReadStateJSON{
		RoomID:              s.RoomID,
		MyLastReadMessageID: s.MyLastReadMessageID,
		UnreadCount:         s.UnreadCount,
		MemberReads:         members,
	}
}
