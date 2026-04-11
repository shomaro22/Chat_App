package controller

import (
	"time"

	"chat-app/entity"

	"github.com/google/uuid"
)

type createMessageRequest struct {
	SenderID string  `json:"sender_id"`
	Content  *string `json:"content"`
	ImageURL *string `json:"image_url"`
}

type createMessageResponse struct {
	Message messageJSON `json:"message"`
}

type listMessagesResponse struct {
	Messages []messageJSON `json:"messages"`
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

// messageCreatedWSBroadcast は WebSocket で送る message_created イベント用 JSON。
type messageCreatedWSBroadcast struct {
	Type    string      `json:"type"`
	RoomID  uuid.UUID   `json:"room_id"`
	Message messageJSON `json:"message"`
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
