package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"chat-app/entity"
	"chat-app/usecase"
	"chat-app/wshub"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// MessageController exposes message HTTP handlers.
// Hub is optional; when set, successful sends are broadcast to subscribed WebSocket clients for that room.
// TODO(auth): remove sender_id from POST body and user_id from query params; derive from JWT / session.
type MessageController struct {
	Messages *usecase.MessageUseCase
	Hub      *wshub.ConnectionManager
}

func NewMessageController(messages *usecase.MessageUseCase, hub *wshub.ConnectionManager) *MessageController {
	return &MessageController{Messages: messages, Hub: hub}
}

// --- POST /rooms/:id/messages ---

type createMessageRequest struct {
	SenderID string  `json:"sender_id"`
	Content  *string `json:"content"`
	ImageURL *string `json:"image_url"`
}

type createMessageResponse struct {
	Message messageJSON `json:"message"`
}

// --- GET /rooms/:id/messages ---

type listMessagesResponse struct {
	Messages []messageJSON `json:"messages"`
}

type messageJSON struct {
	ID        uuid.UUID `json:"id"`
	RoomID    uuid.UUID `json:"room_id"`
	SenderID  uuid.UUID `json:"sender_id"`
	Content   *string   `json:"content"`
	ImageURL  *string   `json:"image_url"`
	DeletedAt *string   `json:"deleted_at"`
	CreatedAt string    `json:"created_at"`
}

func messageToJSON(m *entity.Message) messageJSON {
	var deleted *string
	if m.DeletedAt != nil {
		s := m.DeletedAt.UTC().Format(time.RFC3339Nano)
		deleted = &s
	}
	return messageJSON{
		ID:        m.ID,
		RoomID:    m.RoomID,
		SenderID:  m.SenderID,
		Content:   copyStrPtr(m.Content),
		ImageURL:  copyStrPtr(m.ImageURL),
		DeletedAt: deleted,
		CreatedAt: m.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func copyStrPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := *s
	return &v
}

func (h *MessageController) CreateMessage(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid room id")
	}
	var body createMessageRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	senderID, err := uuid.Parse(body.SenderID)
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid sender_id")
	}
	msg, err := h.Messages.SendMessage(c.Request().Context(), roomID, senderID, body.Content, body.ImageURL)
	if err != nil {
		return mapMessageUsecaseError(c, err)
	}
	h.broadcastMessageCreated(msg)
	return c.JSON(http.StatusCreated, createMessageResponse{Message: messageToJSON(msg)})
}

func (h *MessageController) broadcastMessageCreated(msg *entity.Message) {
	if h.Hub == nil || msg == nil {
		return
	}
	payload := struct {
		Type    string      `json:"type"`
		RoomID  uuid.UUID   `json:"room_id"`
		Message messageJSON `json:"message"`
	}{
		Type:    "message_created",
		RoomID:  msg.RoomID,
		Message: messageToJSON(msg),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	h.Hub.BroadcastToRoom(msg.RoomID, b)
}

func (h *MessageController) ListMessages(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid room id")
	}
	rawUser := c.QueryParam("user_id")
	if rawUser == "" {
		return jsonError(c, http.StatusBadRequest, "user_id is required")
	}
	userID, err := uuid.Parse(rawUser)
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid user_id")
	}
	limit := 0
	if raw := c.QueryParam("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return jsonError(c, http.StatusBadRequest, "invalid limit")
		}
		limit = n
	}
	var beforeID *uuid.UUID
	if raw := c.QueryParam("before_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return jsonError(c, http.StatusBadRequest, "invalid before_id")
		}
		beforeID = &id
	}
	list, err := h.Messages.ListMessages(c.Request().Context(), roomID, userID, beforeID, limit)
	if err != nil {
		return mapMessageUsecaseError(c, err)
	}
	out := make([]messageJSON, 0, len(list))
	for i := range list {
		out = append(out, messageToJSON(&list[i]))
	}
	return c.JSON(http.StatusOK, listMessagesResponse{Messages: out})
}

func (h *MessageController) DeleteMessage(c echo.Context) error {
	messageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid message id")
	}
	rawUser := c.QueryParam("user_id")
	if rawUser == "" {
		return jsonError(c, http.StatusBadRequest, "user_id is required")
	}
	userID, err := uuid.Parse(rawUser)
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid user_id")
	}
	if err := h.Messages.DeleteMessage(c.Request().Context(), messageID, userID); err != nil {
		return mapMessageUsecaseError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func mapMessageUsecaseError(c echo.Context, err error) error {
	status, msg := messageUsecaseHTTP(err)
	return c.JSON(status, map[string]string{"error": msg})
}

func messageUsecaseHTTP(err error) (int, string) {
	switch {
	case errors.Is(err, usecase.ErrInvalidRoomID),
		errors.Is(err, usecase.ErrInvalidMessageID),
		errors.Is(err, usecase.ErrInvalidSenderID),
		errors.Is(err, usecase.ErrEmptyMessage),
		errors.Is(err, usecase.ErrInvalidUserID):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, usecase.ErrNotRoomMember),
		errors.Is(err, usecase.ErrForbiddenMessageEdit):
		return http.StatusForbidden, err.Error()
	case errors.Is(err, usecase.ErrMessageNotFound):
		return http.StatusNotFound, err.Error()
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
