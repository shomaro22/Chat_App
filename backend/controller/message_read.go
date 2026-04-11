package controller

import (
	"encoding/json"
	"errors"
	"net/http"

	"chat-app/usecase"
	"chat-app/wshub"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// MessageReadController は既読更新の HTTP と WS 通知。
type MessageReadController struct {
	Reads MessageReadUseCase
	Hub   *wshub.ConnectionManager
}

// NewMessageReadController は MessageReadController を生成する。
func NewMessageReadController(reads MessageReadUseCase, hub *wshub.ConnectionManager) *MessageReadController {
	return &MessageReadController{Reads: reads, Hub: hub}
}

// MarkAsRead は POST /rooms/:id/read で最終既読を更新する。
func (h *MessageReadController) MarkAsRead(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid room id")
	}
	var body markAsReadRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid user_id")
	}
	messageID, err := uuid.Parse(body.MessageID)
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid message_id")
	}
	result, err := h.Reads.MarkAsRead(c.Request().Context(), userID, roomID, messageID)
	if err != nil {
		return mapMessageReadUsecaseError(c, err)
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
		Type:              "read_updated",
		RoomID:            roomID,
		UserID:            userID,
		LastReadMessageID: lastReadMessageID,
	})
	if err != nil {
		return
	}
	h.Hub.BroadcastToRoom(roomID, b)
}

func mapMessageReadUsecaseError(c echo.Context, err error) error {
	status, msg := messageReadUsecaseHTTP(err)
	return c.JSON(status, map[string]string{"error": msg})
}

func messageReadUsecaseHTTP(err error) (int, string) {
	switch {
	case errors.Is(err, usecase.ErrInvalidMessageReadUserID),
		errors.Is(err, usecase.ErrInvalidMessageReadRoomID),
		errors.Is(err, usecase.ErrInvalidReadMessageID),
		errors.Is(err, usecase.ErrReadMessageRoomMismatch):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, usecase.ErrReadNotRoomMember):
		return http.StatusForbidden, err.Error()
	case errors.Is(err, usecase.ErrReadMessageNotFound):
		return http.StatusNotFound, err.Error()
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
