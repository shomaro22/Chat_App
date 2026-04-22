package controller

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// HubPresenceChange は接続の増減に伴うオンライン状態の変化（ハブが Register / RemoveConn で返す境界用 DTO）。
type HubPresenceChange struct {
	Changed  bool
	UserID   uuid.UUID
	IsOnline bool
}

// RoomBroadcastHub はルーム単位の WebSocket 配信に必要な最小インターフェース。
type RoomBroadcastHub interface {
	BroadcastToRoom(roomID uuid.UUID, payload []byte)
}

// PresenceBroadcastHub は複数ルームの購読者へプレゼンス等を配信するときの最小インターフェース。
type PresenceBroadcastHub interface {
	BroadcastToUsersInRooms(roomIDs []uuid.UUID, payload []byte)
}

// WebSocketHub は接続登録・購読・書き込み・複数ルーム配信など WS ハブの操作を表す。
type WebSocketHub interface {
	Register(userID uuid.UUID, conn *websocket.Conn) (connID uuid.UUID, change HubPresenceChange)
	RemoveConn(connID uuid.UUID) (change HubPresenceChange)
	Subscribe(connID, roomID uuid.UUID)
	Unsubscribe(connID, roomID uuid.UUID)
	WriteText(connID uuid.UUID, payload []byte) bool
	SendPing(connID uuid.UUID) bool
	BroadcastToRoom(roomID uuid.UUID, payload []byte)
	BroadcastToUsersInRooms(roomIDs []uuid.UUID, payload []byte)
	CloseUserConnections(userID uuid.UUID) HubPresenceChange
}

// UserConnectionCloser は管理者による強制切断などに使う最小インターフェース。
type UserConnectionCloser interface {
	CloseUserConnections(userID uuid.UUID) HubPresenceChange
}
