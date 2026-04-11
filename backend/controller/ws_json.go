package controller

import (
	"time"

	"github.com/google/uuid"
)

// WebSocket イベント契約（サーバー ↔ クライアントで固定）
//
// # 共通
//   - 全フレームは TextMessage の JSON オブジェクト（サーバーからの WebSocket Ping は非 JSON）。
//   - サーバー → クライアントの各イベントに schema_version を付与（WSEventSchemaVersion）。
//   - クライアント → サーバーは schema_version を省略可（0 とみなす）。WSClientSchemaVersion のみ許可。
//
// # Presence（オンライン）
//   - online 定義: 当該ユーザーに登録済み WebSocket 接続が 1 本以上ある状態。
//   - 複数接続: 同一 user の接続数はハブで集約。接続追加で 0→1 になったときだけ「オンライン化」、
//     削除で 1→0 になったときだけ「オフライン化」とみなす（中間の本数変化では presence_updated を送らない）。
//   - presence_updated: 上記の遷移が起きたとき、当該ユーザーが参加しているルームの購読者へブロードキャスト
//     （同一接続が複数ルームを購読していても、その接続へは 1 回にまとめる）。
//
// # Keepalive
//   - サーバー: WSPingInterval ごとに WebSocket Ping フレームを送信。クライアントは Pong で応答すること。
//   - 読み取り: 最後に Pong を受信（またはメッセージ処理）から WSReadDeadline 以内に読み取りが無いと切断扱い。
//   - 任意: クライアントが JSON {"type":"ping","schema_version":1} を送ると {"type":"pong",...} を返す（room_id 不要）。

// WSEventSchemaVersion はサーバー → クライアントのイベント JSON の契約バージョン。
const WSEventSchemaVersion = 1

// WSClientSchemaVersion はクライアント → サーバーで許可する schema_version（0 は省略と同等）。
const WSClientSchemaVersion = 1

// WSPingInterval / WSReadDeadline は接続維持の実装値（本コメントと同期すること）。
var (
	WSPingInterval = 30 * time.Second
	WSReadDeadline = 90 * time.Second
)

// サーバー → クライアントの type（固定文字列）。
const (
	WSEventError           = "error"
	WSEventSessionReady    = "session_ready"
	WSEventMessageCreated  = "message_created"
	WSEventMessageDeleted  = "message_deleted"
	WSEventReadUpdated     = "read_updated"
	WSEventPresenceUpdated = "presence_updated"
	WSEventSubscribed      = "subscribed"
	WSEventUnsubscribed    = "unsubscribed"
	WSEventPong            = "pong"
)

// クライアント → サーバーの type（購読・解除・任意 JSON ping）。
const (
	WSClientPing = "ping"

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
	SchemaVersion int    `json:"schema_version,omitempty"`
}

// wsErrorOutbound は type=error の固定形。room_id は購読系エラー時に付く場合がある。
type wsErrorOutbound struct {
	Type          string  `json:"type"`
	SchemaVersion int     `json:"schema_version"`
	Code          string  `json:"code"`
	Message       string  `json:"message"`
	RoomID        *string `json:"room_id,omitempty"`
}

// wsSessionReadyOutbound は接続直後に 1 回だけ送るハンドシェイク用イベント。
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
