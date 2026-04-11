package controller

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// MessageReadController は既読更新の HTTP と WS 通知。
type MessageReadController struct {
	Reads MessageReadUseCase
	Hub   RoomBroadcastHub
}

// NewMessageReadController は MessageReadController を生成する。
func NewMessageReadController(reads MessageReadUseCase, hub RoomBroadcastHub) *MessageReadController {
	return &MessageReadController{Reads: reads, Hub: hub}
}

// GetRoomReadState は GET /rooms/:id/read-state で未読数・既読概要を返す。
func (h *MessageReadController) GetRoomReadState(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid room id")
	}
	userID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	state, err := h.Reads.GetRoomReadState(c.Request().Context(), userID, roomID)
	if err != nil {
		return mapDomainError(c, err)
	}
	return c.JSON(http.StatusOK, roomReadStateToJSON(state))
}

// MarkAsRead は POST /rooms/:id/read で最終既読を更新する。
func (h *MessageReadController) MarkAsRead(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid room id")
	}
	userID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	var body markAsReadRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	messageID, err := uuid.Parse(body.MessageID)
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid message_id")
	}
	result, err := h.Reads.MarkAsRead(c.Request().Context(), userID, roomID, messageID)
	if err != nil {
		return mapDomainError(c, err)
	}
	if result.Updated {
		h.broadcastReadUpdated(roomID, userID, result.LastReadMessageID)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *MessageReadController) broadcastReadUpdated(roomID, userID, lastReadMessageID uuid.UUID) {
	if h.Hub == nil {
		return
	}
	b, err := json.Marshal(readUpdatedPayload{
		Type:              WSEventReadUpdated,
		SchemaVersion:     WSEventSchemaVersion,
		RoomID:            roomID,
		UserID:            userID,
		LastReadMessageID: lastReadMessageID,
	})
	if err != nil {
		return
	}
	h.Hub.BroadcastToRoom(roomID, b)
}
