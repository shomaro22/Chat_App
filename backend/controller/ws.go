package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"chat-app/usecase"

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

const wsDBOpTimeout = 5 * time.Second

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
		return conn.SetReadDeadline(time.Now().Add(WSReadDeadline))
	})
	conn.SetReadDeadline(time.Now().Add(WSReadDeadline))

	connID, presOnConnect := w.Hub.Register(userID, conn)
	w.writeJSON(connID, wsSessionReadyOutbound{
		Type:          WSEventSessionReady,
		SchemaVersion: WSEventSchemaVersion,
		UserID:        userID,
	})
	if presOnConnect.Changed && presOnConnect.IsOnline {
		pctx, pcancel := context.WithTimeout(context.Background(), wsDBOpTimeout)
		BroadcastPresenceUpdated(w.Hub, w.Presence, pctx, userID, true)
		pcancel()
	}

	done := make(chan struct{})
	go w.pingLoop(connID, done)

	defer func() {
		close(done)
		ch := w.Hub.RemoveConn(connID)
		if ch.Changed {
			ctx, cancel := context.WithTimeout(context.Background(), wsDBOpTimeout)
			BroadcastPresenceUpdated(w.Hub, w.Presence, ctx, ch.UserID, ch.IsOnline)
			cancel()
		}
		_ = conn.Close()
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		conn.SetReadDeadline(time.Now().Add(WSReadDeadline))

		var in wsClientMessage
		if err := json.Unmarshal(data, &in); err != nil {
			w.writeWSError(connID, WSCodeInvalidJSON, "invalid json", nil)
			continue
		}
		if in.SchemaVersion != 0 && in.SchemaVersion != WSClientSchemaVersion {
			w.writeWSError(connID, WSCodeSchemaVersion, "unsupported schema_version", nil)
			continue
		}
		typ := strings.TrimSpace(in.Type)
		if typ == "" {
			w.writeWSError(connID, WSCodeMissingType, "type is required", nil)
			continue
		}
		if typ == WSClientPing {
			w.writeJSON(connID, wsPongOutbound{Type: WSEventPong, SchemaVersion: WSEventSchemaVersion})
			continue
		}
		roomRaw := strings.TrimSpace(in.RoomID)
		if roomRaw == "" {
			w.writeWSError(connID, WSCodeMissingRoomID, "room_id is required", nil)
			continue
		}
		roomID, err := uuid.Parse(roomRaw)
		if err != nil {
			w.writeWSError(connID, WSCodeInvalidRoomID, "invalid room_id", nil)
			continue
		}
		roomPtr := &roomID

		switch typ {
		case WSClientSubscribePrimary, WSClientSubscribeLegacy1, WSClientSubscribeLegacy2:
			ctx, cancel := context.WithTimeout(context.Background(), wsDBOpTimeout)
			err := w.Rooms.CanSubscribeWebSocket(ctx, roomID, userID)
			cancel()
			if err != nil {
				switch {
				case errors.Is(err, usecase.ErrRoomFrozen):
					w.writeWSError(connID, WSCodeRoomFrozen, "room is frozen", roomPtr)
				case errors.Is(err, usecase.ErrNotRoomMember):
					w.writeWSError(connID, WSCodeForbiddenNotMember, "not an active member of this room", roomPtr)
				case errors.Is(err, usecase.ErrInvalidRoomID), errors.Is(err, usecase.ErrInvalidUserID):
					w.writeWSError(connID, WSCodeInvalidRoomID, "invalid room or user", roomPtr)
				default:
					w.writeWSError(connID, WSCodeInternal, "internal error", roomPtr)
				}
				continue
			}
			w.Hub.Subscribe(connID, roomID)
			w.writeJSON(connID, wsRoomAckOutbound{Type: WSEventSubscribed, SchemaVersion: WSEventSchemaVersion, RoomID: roomID.String()})
		case WSClientUnsubscribePrimary, WSClientUnsubscribeLegacy1, WSClientUnsubscribeLegacy2:
			w.Hub.Unsubscribe(connID, roomID)
			w.writeJSON(connID, wsRoomAckOutbound{Type: WSEventUnsubscribed, SchemaVersion: WSEventSchemaVersion, RoomID: roomID.String()})
		default:
			w.writeWSError(connID, WSCodeUnknownType, "unknown type", roomPtr)
		}
	}
	return nil
}

func (w *WSController) pingLoop(connID uuid.UUID, done <-chan struct{}) {
	ticker := time.NewTicker(WSPingInterval)
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

func (w *WSController) writeWSError(connID uuid.UUID, code, msg string, roomID *uuid.UUID) {
	out := wsErrorOutbound{
		Type:          WSEventError,
		SchemaVersion: WSEventSchemaVersion,
		Code:          code,
		Message:       msg,
	}
	if roomID != nil {
		s := roomID.String()
		out.RoomID = &s
	}
	w.writeJSON(connID, out)
}

func (w *WSController) writeJSON(connID uuid.UUID, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	_ = w.Hub.WriteText(connID, b)
}

// BroadcastPresenceUpdated はユーザーのオンライン状態を、参加ルームの購読者へ presence_updated で配信する（WS 接続・管理者切断の共通処理）。
func BroadcastPresenceUpdated(hub PresenceBroadcastHub, rooms PresenceJoinedRooms, ctx context.Context, userID uuid.UUID, isOnline bool) {
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
		Type:          WSEventPresenceUpdated,
		SchemaVersion: WSEventSchemaVersion,
		UserID:        userID,
		IsOnline:      isOnline,
	})
	if err != nil {
		return
	}
	hub.BroadcastToUsersInRooms(roomIDs, b)
}
