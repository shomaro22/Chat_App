package controller

import "github.com/google/uuid"

// WSEventSchemaVersion はサーバー → クライアントのイベント JSON に付与する契約バージョン。
const WSEventSchemaVersion = 1

// サーバー → クライアントの type 文字列（フロント・テストの前提として固定）。
const (
	WSEventError            = "error"
	WSEventMessageCreated   = "message.created"
	WSEventMessageDeleted   = "message.deleted"
	WSEventMessageRead      = "message.read"
	WSEventPresenceChanged  = "presence.changed"
	WSEventRoomSubscribed   = "room.subscribed"
	WSEventRoomUnsubscribed = "room.unsubscribed"
)

// サーバー → クライアントの error イベントで使う code 定数。
const (
	WSCodeInvalidJSON        = "INVALID_JSON"
	WSCodeInvalidRoomID      = "INVALID_ROOM_ID"
	WSCodeForbiddenNotMember = "FORBIDDEN_NOT_MEMBER"
	WSCodeRoomFrozen         = "ROOM_FROZEN"
	WSCodeUnknownType        = "UNKNOWN_TYPE"
	WSCodeInternal           = "INTERNAL"
)

type wsClientMessage struct {
	Type          string `json:"type"`
	RoomID        string `json:"room_id"`
	SchemaVersion int    `json:"schema_version,omitempty"` // 将来用。未送信でもよい。
}

// wsErrorOutbound は type=error の固定形。
type wsErrorOutbound struct {
	Type          string `json:"type"`
	SchemaVersion int    `json:"schema_version"`
	Code          string `json:"code"`
	Message       string `json:"message"`
}

// wsRoomAckOutbound は subscribed / unsubscribed 共通形。
type wsRoomAckOutbound struct {
	Type          string `json:"type"`
	SchemaVersion int    `json:"schema_version"`
	RoomID        string `json:"room_id"`
}

// presenceUpdatedPayload は presence.changed イベント用 JSON。
type presenceUpdatedPayload struct {
	Type          string    `json:"type"`
	SchemaVersion int       `json:"schema_version"`
	UserID        uuid.UUID `json:"user_id"`
	IsOnline      bool      `json:"is_online"`
}
