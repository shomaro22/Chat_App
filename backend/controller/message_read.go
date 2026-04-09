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

type MessageReadController struct {
	Reads *usecase.MessageReadUseCase
	Hub   *wshub.ConnectionManager
}

func NewMessageReadController(reads *usecase.MessageReadUseCase, hub *wshub.ConnectionManager) *MessageReadController {
	return &MessageReadController{Reads: reads, Hub: hub}
}

type markAsReadRequest struct {
	UserID    string `json:"user_id"`
	MessageID string `json:"message_id"`
}

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

type readUpdatedPayload struct {
	Type              string    `json:"type"`
	RoomID            uuid.UUID `json:"room_id"`
	UserID            uuid.UUID `json:"user_id"`
	LastReadMessageID uuid.UUID `json:"last_read_message_id"`
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
