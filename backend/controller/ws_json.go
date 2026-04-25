package controller

import (
	"time"

	"github.com/google/uuid"
)

// WebSocket イベント契約（サーバー ↔ クライアントで固定）

// WSEventSchemaVersion はサーバー → クライアントのイベント JSON の契約バージョン。
const WSEventSchemaVersion = 1

// WSClientSchemaVersion はクライアント → サーバーで許可する schema_version（0 は省略と同等）。
const WSClientSchemaVersion = 1

// WSPingInterval / WSReadDeadline は接続維持の実装値
var (
	WSPingInterval = 30 * time.Second
	WSReadDeadline = 90 * time.Second
)

// サーバー → クライアントの type（固定文字列）。
const (
	WSEventError           = "error"
	WSEventSessionReady    = "session_ready"
	WSEventMessageCreated  = "message_created"
	WSEventMessageUpdated  = "message_updated"
	WSEventMessageDeleted  = "message_deleted"
	WSEventReadUpdated     = "read_updated"
	WSEventPresenceUpdated = "presence_updated"
	WSEventTyping          = "typing"
	WSEventSubscribed      = "subscribed"
	WSEventUnsubscribed    = "unsubscribed"
	WSEventPong            = "pong"
)

// クライアント → サーバーの type（購読・解除・任意 JSON ping）。
const (
	WSClientPing   = "ping"
	WSClientTyping = "typing"

	WSClientSubscribePrimary   = "subscribe"
	WSClientSubscribeLegacy1   = "subscribe_room"
	WSClientSubscribeLegacy2   = "room.subscribe"
	WSClientUnsubscribePrimary = "unsubscribe"
	WSClientUnsubscribeLegacy1 = "unsubscribe_room"
	WSClientUnsubscribeLegacy2 = "room.unsubscribe"
)

// サーバー → クライアントの error イベント code。
const (
	WSCodeInvalidJSON        = "INVALID_JSON"
	WSCodeInvalidRoomID      = "INVALID_ROOM_ID"
	WSCodeForbiddenNotMember = "FORBIDDEN_NOT_MEMBER"
	WSCodeRoomFrozen         = "ROOM_FROZEN"
	WSCodeUnknownType        = "UNKNOWN_TYPE"
	WSCodeInternal           = "INTERNAL"
	WSCodeSchemaVersion      = "SCHEMA_VERSION_MISMATCH"
	WSCodeMissingType        = "MISSING_TYPE"
	WSCodeMissingRoomID      = "MISSING_ROOM_ID"
)

type wsClientMessage struct {
	Type          string `json:"type"`
	RoomID        string `json:"room_id"`
	IsTyping      *bool  `json:"is_typing,omitempty"`
	SchemaVersion int    `json:"schema_version,omitempty"`
}

// typingOutboundPayload は typing イベント（他ユーザー入力中）。
type typingOutboundPayload struct {
	Type          string    `json:"type"`
	SchemaVersion int       `json:"schema_version"`
	RoomID        uuid.UUID `json:"room_id"`
	UserID        uuid.UUID `json:"user_id"`
	IsTyping      bool      `json:"is_typing"`
}

// wsErrorOutbound は type=error の固定形。room_id は購読系エラー時に付く場合がある。
type wsErrorOutbound struct {
	Type          string  `json:"type"`
	SchemaVersion int     `json:"schema_version"`
	Code          string  `json:"code"`
	Message       string  `json:"message"`
	RoomID        *string `json:"room_id,omitempty"`
}

// wsSessionReadyOutbound は接続直後に1回だけ送るハンドシェイク用イベント。
type wsSessionReadyOutbound struct {
	Type          string    `json:"type"`
	SchemaVersion int       `json:"schema_version"`
	UserID        uuid.UUID `json:"user_id"`
}

// wsRoomAckOutbound は subscribed / unsubscribed 共通形（購読の結果 ACK）。
type wsRoomAckOutbound struct {
	Type          string `json:"type"`
	SchemaVersion int    `json:"schema_version"`
	RoomID        string `json:"room_id"`
}

// wsPongOutbound はクライアント JSON ping への応答。
type wsPongOutbound struct {
	Type          string `json:"type"`
	SchemaVersion int    `json:"schema_version"`
}

// presenceUpdatedPayload は presence_updated イベント用 JSON。
type presenceUpdatedPayload struct {
	Type          string    `json:"type"`
	SchemaVersion int       `json:"schema_version"`
	UserID        uuid.UUID `json:"user_id"`
	IsOnline      bool      `json:"is_online"`
}
