package controller

import "github.com/google/uuid"

type markAsReadRequest struct {
	UserID    string `json:"user_id"`
	MessageID string `json:"message_id"`
}

// readUpdatedPayload は WebSocket read_updated イベント用 JSON。
type readUpdatedPayload struct {
	Type              string    `json:"type"`
	RoomID            uuid.UUID `json:"room_id"`
	UserID            uuid.UUID `json:"user_id"`
	LastReadMessageID uuid.UUID `json:"last_read_message_id"`
}
