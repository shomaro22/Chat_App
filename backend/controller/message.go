package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"chat-app/entity"
	"chat-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// MessageController はメッセージ関連の HTTP ハンドラと作成時の WS 通知。
type MessageController struct {
	Messages MessageUseCase
	Hub      RoomBroadcastHub
}

// NewMessageController は MessageController を生成する。
func NewMessageController(messages MessageUseCase, hub RoomBroadcastHub) *MessageController {
	return &MessageController{Messages: messages, Hub: hub}
}

// CreateMessage は POST /rooms/:id/messages でメッセージを作成する。
func (h *MessageController) CreateMessage(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid room id")
	}
	senderID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	var body createMessageRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
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
	payload := messageCreatedWSBroadcast{
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

// ListMessages は GET /rooms/:id/messages で履歴を返す。order=asc|desc、include_read_count で既読人数を付与。
func (h *MessageController) ListMessages(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid room id")
	}
	userID, err := CurrentUserID(c)
	if err != nil {
		return err
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
	var afterID *uuid.UUID
	if raw := c.QueryParam("after_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return jsonError(c, http.StatusBadRequest, "invalid after_id")
		}
		afterID = &id
	}
	orderDesc := true
	if raw := strings.ToLower(strings.TrimSpace(c.QueryParam("order"))); raw == "asc" {
		orderDesc = false
	} else if raw == "desc" || raw == "" {
		orderDesc = true
	} else if raw != "" {
		return jsonError(c, http.StatusBadRequest, "invalid order (use asc or desc)")
	}
	includeRead := false
	if raw := strings.ToLower(c.QueryParam("include_read_count")); raw == "1" || raw == "true" {
		includeRead = true
	}
	list, err := h.Messages.ListMessages(c.Request().Context(), roomID, userID, beforeID, afterID, limit, orderDesc, includeRead)
	if err != nil {
		return mapMessageUsecaseError(c, err)
	}
	out := make([]listedMessageJSON, 0, len(list))
	for i := range list {
		out = append(out, listedMessageToJSON(list[i]))
	}
	orderLabel := "desc"
	if !orderDesc {
		orderLabel = "asc"
	}
	return c.JSON(http.StatusOK, listMessagesResponse{Messages: out, Order: orderLabel})
}

// DeleteMessage は DELETE /messages/:id で論理削除し、WS で通知する。
func (h *MessageController) DeleteMessage(c echo.Context) error {
	messageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid message id")
	}
	userID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	roomID, err := h.Messages.DeleteMessage(c.Request().Context(), messageID, userID)
	if err != nil {
		return mapMessageUsecaseError(c, err)
	}
	if h.Hub != nil && roomID != uuid.Nil {
		b, err := json.Marshal(messageDeletedWSBroadcast{
			Type:      "message_deleted",
			RoomID:    roomID,
			MessageID: messageID,
		})
		if err == nil {
			h.Hub.BroadcastToRoom(roomID, b)
		}
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
		errors.Is(err, usecase.ErrForbiddenMessageEdit),
		errors.Is(err, usecase.ErrRoomFrozen):
		return http.StatusForbidden, err.Error()
	case errors.Is(err, usecase.ErrMessageNotFound):
		return http.StatusNotFound, err.Error()
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
