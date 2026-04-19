package controller

import (
	"net/http"
	"strconv"
	"time"

	"chat-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// SocialController はリアクション・ピン・ブックマーク。
type SocialController struct {
	Social *usecase.SocialUseCase
}

func NewSocialController(s *usecase.SocialUseCase) *SocialController {
	return &SocialController{Social: s}
}

type reactionRequest struct {
	Emoji string `json:"emoji"`
}

type reactionRowJSON struct {
	UserID    uuid.UUID `json:"user_id"`
	Emoji     string    `json:"emoji"`
	CreatedAt string    `json:"created_at"`
}

type pinRequest struct {
	MessageID uuid.UUID `json:"message_id"`
}

type bookmarkRowJSON struct {
	RoomID    uuid.UUID   `json:"room_id"`
	MessageID uuid.UUID   `json:"message_id"`
	CreatedAt string      `json:"created_at"`
	Message   messageJSON `json:"message"`
}

// PostReaction は POST /rooms/:id/messages/:messageId/reactions
func (h *SocialController) PostReaction(c echo.Context) error {
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
	var body reactionRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	if err := h.Social.AddReaction(c.Request().Context(), roomID, userID, messageID, body.Emoji); err != nil {
		return mapDomainError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// DeleteReaction は DELETE /rooms/:id/messages/:messageId/reactions?emoji=
func (h *SocialController) DeleteReaction(c echo.Context) error {
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
	emoji := c.QueryParam("emoji")
	if err := h.Social.RemoveReaction(c.Request().Context(), roomID, userID, messageID, emoji); err != nil {
		return mapDomainError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ListReactions は GET /rooms/:id/messages/:messageId/reactions
func (h *SocialController) ListReactions(c echo.Context) error {
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
	rows, err := h.Social.ListReactions(c.Request().Context(), roomID, userID, messageID)
	if err != nil {
		return mapDomainError(c, err)
	}
	out := make([]reactionRowJSON, 0, len(rows))
	for i := range rows {
		out = append(out, reactionRowJSON{
			UserID:    rows[i].UserID,
			Emoji:     rows[i].Emoji,
			CreatedAt: rows[i].CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"reactions": out})
}

// PostPin は POST /rooms/:id/pin
func (h *SocialController) PostPin(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid room id")
	}
	userID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	var body pinRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	if err := h.Social.SetPin(c.Request().Context(), roomID, userID, body.MessageID); err != nil {
		return mapDomainError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// DeletePin は DELETE /rooms/:id/pin
func (h *SocialController) DeletePin(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid room id")
	}
	userID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	if err := h.Social.ClearPin(c.Request().Context(), roomID, userID); err != nil {
		return mapDomainError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// GetPin は GET /rooms/:id/pin
func (h *SocialController) GetPin(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid room id")
	}
	userID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	p, err := h.Social.GetPin(c.Request().Context(), roomID, userID)
	if err != nil {
		return mapDomainError(c, err)
	}
	if p == nil {
		return c.JSON(http.StatusOK, map[string]any{"pin": nil})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"pin": map[string]any{
			"room_id":    p.RoomID,
			"message_id": p.MessageID,
			"pinned_by":  p.PinnedBy,
			"pinned_at":  p.PinnedAt.UTC().Format(time.RFC3339Nano),
		},
	})
}

// PostBookmark は POST /messages/:id/bookmark
func (h *SocialController) PostBookmark(c echo.Context) error {
	messageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid message id")
	}
	userID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	if err := h.Social.AddBookmark(c.Request().Context(), userID, messageID); err != nil {
		return mapDomainError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// DeleteBookmark は DELETE /messages/:id/bookmark
func (h *SocialController) DeleteBookmark(c echo.Context) error {
	messageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid message id")
	}
	userID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	if err := h.Social.RemoveBookmark(c.Request().Context(), userID, messageID); err != nil {
		return mapDomainError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ListBookmarks は GET /me/bookmarks
func (h *SocialController) ListBookmarks(c echo.Context) error {
	userID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	limit, offset := 0, 0
	if raw := c.QueryParam("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return jsonError(c, http.StatusBadRequest, "invalid limit")
		}
		limit = n
	}
	if raw := c.QueryParam("offset"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return jsonError(c, http.StatusBadRequest, "invalid offset")
		}
		offset = n
	}
	rows, total, err := h.Social.ListBookmarksWithMessages(c.Request().Context(), userID, limit, offset)
	if err != nil {
		return mapDomainError(c, err)
	}
	out := make([]bookmarkRowJSON, 0, len(rows))
	for i := range rows {
		out = append(out, bookmarkRowJSON{
			RoomID:    rows[i].Bookmark.RoomID,
			MessageID: rows[i].Bookmark.MessageID,
			CreatedAt: rows[i].Bookmark.CreatedAt.UTC().Format(time.RFC3339Nano),
			Message:   messageToJSON(&rows[i].Message),
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"bookmarks": out, "total": total})
}
