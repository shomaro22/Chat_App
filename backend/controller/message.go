package controller

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"chat-app/entity"
	"chat-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const maxMessageMultipartMemory = 12 << 20 // メディア上限 10MiB

// MessageController はメッセージ関連の HTTP ハンドラと作成時の WS 通知。
type MessageController struct {
	Messages MessageUseCase
	Msg      *usecase.MessageUseCase // 編集・検索・拡張送信（アダプタと同一インスタンス）
	Hub      RoomBroadcastHub
	Media    MediaUseCase // multipart 画像送信に使用。nil のときファイル付き multipart は不可。
}

// NewMessageController は MessageController を生成する。
func NewMessageController(msgUC *usecase.MessageUseCase, hub RoomBroadcastHub, media MediaUseCase) *MessageController {
	return &MessageController{
		Messages: NewMessageUseCaseAdapter(msgUC),
		Msg:      msgUC,
		Hub:      hub,
		Media:    media,
	}
}

// CreateMessage は POST /rooms/:id/messages でメッセージを作成する。
// Content-Type が application/json のときは JSON（content / image_url）。
// multipart/form-data のときはフィールド content（任意）と file（任意画像）。どちらか一方以上必須。
func (h *MessageController) CreateMessage(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid room id")
	}
	senderID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	ct := strings.ToLower(strings.TrimSpace(c.Request().Header.Get("Content-Type")))
	if strings.HasPrefix(ct, "multipart/form-data") {
		return h.createMessageFromMultipart(c, roomID, senderID)
	}
	var body createMessageRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	in := usecase.MessageSendInput{
		RoomID:           roomID,
		SenderID:         senderID,
		Content:          body.Content,
		ImageURL:         body.ImageURL,
		FileURL:          body.FileURL,
		FileName:         body.FileName,
		FileContentType:  body.FileContentType,
		ParentMessageID:  body.ParentMessageID,
	}
	msg, err := h.Msg.SendRoomMessage(c.Request().Context(), in)
	if err != nil {
		return mapDomainError(c, err)
	}
	h.broadcastMessageCreated(msg)
	return c.JSON(http.StatusCreated, createMessageResponse{Message: messageToJSON(msg)})
}

func (h *MessageController) createMessageFromMultipart(c echo.Context, roomID, senderID uuid.UUID) error {
	if err := c.Request().ParseMultipartForm(maxMessageMultipartMemory); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid multipart body")
	}
	defer func() {
		_ = c.Request().MultipartForm.RemoveAll()
	}()

	var content *string
	if v := strings.TrimSpace(c.FormValue("content")); v != "" {
		content = &v
	}

	var imageURL *string
	fhs := c.Request().MultipartForm.File["file"]
	if len(fhs) > 0 {
		if h.Media == nil {
			return jsonError(c, http.StatusServiceUnavailable, "image upload is not configured")
		}
		fh := fhs[0]
		src, err := fh.Open()
		if err != nil {
			return jsonError(c, http.StatusBadRequest, "cannot open upload")
		}
		defer src.Close()
		mimeType := fh.Header.Get("Content-Type")
		path, err := h.Media.SaveUploadedImage(mimeType, src)
		if err != nil {
			return mapMediaError(c, err)
		}
		imageURL = &path
	}

	if content == nil && imageURL == nil {
		return jsonError(c, http.StatusBadRequest, "content or file is required")
	}

	msg, err := h.Msg.SendRoomMessage(c.Request().Context(), usecase.MessageSendInput{
		RoomID: roomID, SenderID: senderID, Content: content, ImageURL: imageURL,
	})
	if err != nil {
		return mapDomainError(c, err)
	}
	h.broadcastMessageCreated(msg)
	return c.JSON(http.StatusCreated, createMessageResponse{Message: messageToJSON(msg)})
}

func (h *MessageController) broadcastMessageCreated(msg *entity.Message) {
	if h.Hub == nil || msg == nil {
		return
	}
	payload := messageCreatedWSBroadcast{
		Type:          WSEventMessageCreated,
		SchemaVersion: WSEventSchemaVersion,
		RoomID:        msg.RoomID,
		Message:       messageToJSON(msg),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	h.Hub.BroadcastToRoom(msg.RoomID, b)
}

func (h *MessageController) broadcastMessageUpdated(msg *entity.Message) {
	if h.Hub == nil || msg == nil {
		return
	}
	payload := messageUpdatedWSBroadcast{
		Type:          WSEventMessageUpdated,
		SchemaVersion: WSEventSchemaVersion,
		RoomID:        msg.RoomID,
		Message:       messageToJSON(msg),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	h.Hub.BroadcastToRoom(msg.RoomID, b)
}

// PatchMessage は PATCH /rooms/:id/messages/:messageId（本文のみ）。
func (h *MessageController) PatchMessage(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid room id")
	}
	messageID, err := uuid.Parse(c.Param("messageId"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid message id")
	}
	userID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	var body patchMessageRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	content := strings.TrimSpace(body.Content)
	msg, err := h.Msg.EditMessage(c.Request().Context(), messageID, userID, &content)
	if err != nil {
		return mapDomainError(c, err)
	}
	if msg.RoomID != roomID {
		return jsonError(c, http.StatusBadRequest, "message does not belong to room")
	}
	h.broadcastMessageUpdated(msg)
	return c.JSON(http.StatusOK, createMessageResponse{Message: messageToJSON(msg)})
}

// SearchMessages は GET /search/messages?q=…
func (h *MessageController) SearchMessages(c echo.Context) error {
	userID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	q := strings.TrimSpace(c.QueryParam("q"))
	limit := 0
	if raw := c.QueryParam("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return jsonError(c, http.StatusBadRequest, "invalid limit")
		}
		limit = n
	}
	list, err := h.Msg.SearchJoined(c.Request().Context(), userID, q, limit)
	if err != nil {
		return mapDomainError(c, err)
	}
	out := make([]listedMessageJSON, 0, len(list))
	for i := range list {
		out = append(out, listedMessageToJSON(ListedMessage{
			Message:    list[i].Message,
			SenderName: list[i].SenderName,
			ReadCount:  list[i].ReadCount,
		}))
	}
	return c.JSON(http.StatusOK, map[string]any{"messages": out, "count": len(out)})
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
	effectiveLimit := limit
	if effectiveLimit <= 0 {
		effectiveLimit = DefaultMessageListLimit
	}
	if effectiveLimit > MaxMessageListLimit {
		effectiveLimit = MaxMessageListLimit
	}
	list, err := h.Messages.ListMessages(c.Request().Context(), roomID, userID, beforeID, afterID, limit, orderDesc, includeRead)
	if err != nil {
		return mapDomainError(c, err)
	}
	out := make([]listedMessageJSON, 0, len(list))
	for i := range list {
		out = append(out, listedMessageToJSON(list[i]))
	}
	orderLabel := "desc"
	if !orderDesc {
		orderLabel = "asc"
	}
	return c.JSON(http.StatusOK, listMessagesResponse{
		RoomID:   roomID,
		Messages: out,
		Order:    orderLabel,
		Limit:    effectiveLimit,
		Count:    len(out),
	})
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
		return mapDomainError(c, err)
	}
	if h.Hub != nil && roomID != uuid.Nil {
		b, err := json.Marshal(messageDeletedWSBroadcast{
			Type:          WSEventMessageDeleted,
			SchemaVersion: WSEventSchemaVersion,
			RoomID:        roomID,
			MessageID:     messageID,
		})
		if err == nil {
			h.Hub.BroadcastToRoom(roomID, b)
		}
	}
	return c.NoContent(http.StatusNoContent)
}
