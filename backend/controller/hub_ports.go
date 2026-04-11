package controller

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"chat-app/wshub"
)

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
	Register(userID uuid.UUID, conn *websocket.Conn) (connID uuid.UUID, change wshub.PresenceChange)
	RemoveConn(connID uuid.UUID) (change wshub.PresenceChange)
	Subscribe(connID, roomID uuid.UUID)
	Unsubscribe(connID, roomID uuid.UUID)
	WriteText(connID uuid.UUID, payload []byte) bool
	SendPing(connID uuid.UUID) bool
	BroadcastToUsersInRooms(roomIDs []uuid.UUID, payload []byte)
}
