package controller

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

// WSController は WebSocket 接続・ルーム購読・プレゼンス通知を扱う。
type WSController struct {
	Hub      WebSocketHub
	Rooms    RoomUseCase
	Presence PresenceJoinedRooms
}

// NewWSController は WSController を生成する。Rooms は購読可否判定、Presence は配信先ルーム解決に使う。
func NewWSController(hub WebSocketHub, rooms RoomUseCase, presence PresenceJoinedRooms) *WSController {
	return &WSController{Hub: hub, Rooms: rooms, Presence: presence}
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

const (
	wsReadDeadline = 90 * time.Second
	wsPingInterval = 30 * time.Second
	wsDBOpTimeout  = 5 * time.Second
)

// Handle は GET /ws で WebSocket を確立し、メッセージループを回す。認証は JWT ミドルウェア（Bearer または access_token）で行う。
func (w *WSController) Handle(c echo.Context) error {
	if w.Hub == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "websocket hub not configured")
	}
	userID, err := CurrentUserID(c)
	if err != nil {
		return err
	}

	conn, err := wsUpgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsReadDeadline))
	})
	conn.SetReadDeadline(time.Now().Add(wsReadDeadline))

	connID, presOnConnect := w.Hub.Register(userID, conn)
	if presOnConnect.Changed && presOnConnect.IsOnline {
		pctx, pcancel := context.WithTimeout(context.Background(), wsDBOpTimeout)
		broadcastPresenceUpdated(w.Hub, w.Presence, pctx, userID, true)
		pcancel()
	}

	done := make(chan struct{})
	go w.pingLoop(connID, done)

	defer func() {
		close(done)
		ch := w.Hub.RemoveConn(connID)
		if ch.Changed {
			ctx, cancel := context.WithTimeout(context.Background(), wsDBOpTimeout)
			broadcastPresenceUpdated(w.Hub, w.Presence, ctx, ch.UserID, ch.IsOnline)
			cancel()
		}
		_ = conn.Close()
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		conn.SetReadDeadline(time.Now().Add(wsReadDeadline))

		var in wsClientMessage
		if err := json.Unmarshal(data, &in); err != nil {
			w.writeWSError(connID, "invalid json")
			continue
		}
		roomID, err := uuid.Parse(in.RoomID)
		if err != nil {
			w.writeWSError(connID, "invalid room_id")
			continue
		}

		switch in.Type {
		case "subscribe_room":
			ctx, cancel := context.WithTimeout(context.Background(), wsDBOpTimeout)
			ok, err := w.Rooms.IsActiveRoomMember(ctx, roomID, userID)
			cancel()
			if err != nil {
				w.writeWSError(connID, "internal error")
				continue
			}
			if !ok {
				w.writeWSError(connID, "forbidden: not an active member of this room")
				continue
			}
			w.Hub.Subscribe(connID, roomID)
			w.writeJSON(connID, wsServerMessage{Type: "subscribed", RoomID: roomID.String()})
		case "unsubscribe_room":
			w.Hub.Unsubscribe(connID, roomID)
			w.writeJSON(connID, wsServerMessage{Type: "unsubscribed", RoomID: roomID.String()})
		default:
			w.writeWSError(connID, "unknown type")
		}
	}
	return nil
}

func (w *WSController) pingLoop(connID uuid.UUID, done <-chan struct{}) {
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if !w.Hub.SendPing(connID) {
				return
			}
		}
	}
}

func (w *WSController) writeWSError(connID uuid.UUID, msg string) {
	w.writeJSON(connID, wsServerMessage{Type: "error", Message: msg})
}

func (w *WSController) writeJSON(connID uuid.UUID, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	_ = w.Hub.WriteText(connID, b)
}

// broadcastPresenceUpdated はユーザーのオンライン状態を、参加ルームの購読者へ配信する。
func broadcastPresenceUpdated(hub PresenceBroadcastHub, rooms PresenceJoinedRooms, ctx context.Context, userID uuid.UUID, isOnline bool) {
	if hub == nil || rooms == nil {
		return
	}
	roomIDs, err := rooms.JoinedRoomIDs(ctx, userID)
	if err != nil {
		log.Printf("presence: JoinedRoomIDs user_id=%s: %v", userID, err)
		return
	}
	if len(roomIDs) == 0 {
		return
	}
	b, err := json.Marshal(presenceUpdatedPayload{
		Type:     "presence_updated",
		UserID:   userID,
		IsOnline: isOnline,
	})
	if err != nil {
		return
	}
	hub.BroadcastToUsersInRooms(roomIDs, b)
}
