package controller

import "github.com/google/uuid"

type wsClientMessage struct {
	Type   string `json:"type"`
	RoomID string `json:"room_id"`
}

type wsServerMessage struct {
	Type    string `json:"type"`
	RoomID  string `json:"room_id,omitempty"`
	Message string `json:"message,omitempty"`
}

// presenceUpdatedPayload は WebSocket presence_updated イベント用 JSON。
type presenceUpdatedPayload struct {
	Type     string    `json:"type"`
	UserID   uuid.UUID `json:"user_id"`
	IsOnline bool      `json:"is_online"`
}
