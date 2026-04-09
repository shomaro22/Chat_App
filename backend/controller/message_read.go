package controller

import (
	"errors"
	"net/http"

	"chat-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// MessageReadController exposes read-position HTTP handlers.
// TODO(auth): remove user_id from body; derive from JWT / session.
type MessageReadController struct {
	Reads *usecase.MessageReadUseCase
}

func NewMessageReadController(reads *usecase.MessageReadUseCase) *MessageReadController {
	return &MessageReadController{Reads: reads}
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
	if err := h.Reads.MarkAsRead(c.Request().Context(), userID, roomID, messageID); err != nil {
		return mapMessageReadUsecaseError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
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
