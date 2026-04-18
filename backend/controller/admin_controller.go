package controller

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"chat-app/audit"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const adminPresenceNotifyTimeout = 5 * time.Second

// AdminController は管理者 API。
type AdminController struct {
	Admin AdminUseCase
	// Sockets はユーザー単位の WS 即時切断（ハブからの登録解除を含む）。nil なら切断しない。
	Sockets UserConnectionCloser
	// PresenceHub / PresenceRooms は管理者切断後の presence_updated 配信に使う（いずれか nil ならスキップ）。
	PresenceHub   PresenceBroadcastHub
	PresenceRooms PresenceJoinedRooms
}

// NewAdminController は AdminController を生成する。sockets / presence は省略可（nil）。
func NewAdminController(admin AdminUseCase, sockets UserConnectionCloser, presenceHub PresenceBroadcastHub, presenceRooms PresenceJoinedRooms) *AdminController {
	return &AdminController{Admin: admin, Sockets: sockets, PresenceHub: presenceHub, PresenceRooms: presenceRooms}
}

func (h *AdminController) notifyPresenceAfterForcedDisconnect(ctx context.Context, ch HubPresenceChange) {
	if !ch.Changed || h.PresenceHub == nil || h.PresenceRooms == nil {
		return
	}
	pctx, cancel := context.WithTimeout(ctx, adminPresenceNotifyTimeout)
	defer cancel()
	BroadcastPresenceUpdated(h.PresenceHub, h.PresenceRooms, pctx, ch.UserID, ch.IsOnline)
}

func parseAdminListParams(c echo.Context) (limit, offset int) {
	limit = 50
	offset = 0
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 200 {
		limit = 200
	}
	if v := c.QueryParam("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}

// ListUsers は GET /admin/users（検索 q、limit、offset）。
func (h *AdminController) ListUsers(c echo.Context) error {
	adminID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	limit, offset := parseAdminListParams(c)
	q := strings.TrimSpace(c.QueryParam("q"))
	rows, total, err := h.Admin.ListUsers(c.Request().Context(), adminID, q, limit, offset)
	if err != nil {
		return mapAdminUsecaseError(c, err)
	}
	out := make([]userPublicJSON, 0, len(rows))
	for i := range rows {
		out = append(out, userToPublicJSON(&rows[i]))
	}
	audit.Event("admin.list_users", slog.String("admin_id", adminID.String()), slog.Int("limit", limit), slog.Int("offset", offset))
	return c.JSON(http.StatusOK, adminUserListResponse{Users: out, Total: total, Limit: limit, Offset: offset})
}

// ListRooms は GET /admin/rooms（検索 q、limit、offset）。
func (h *AdminController) ListRooms(c echo.Context) error {
	adminID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	limit, offset := parseAdminListParams(c)
	q := strings.TrimSpace(c.QueryParam("q"))
	rows, total, err := h.Admin.ListRooms(c.Request().Context(), adminID, q, limit, offset)
	if err != nil {
		return mapAdminUsecaseError(c, err)
	}
	out := make([]adminRoomListItemJSON, 0, len(rows))
	for i := range rows {
		out = append(out, roomToAdminListItemJSON(&rows[i]))
	}
	audit.Event("admin.list_rooms", slog.String("admin_id", adminID.String()), slog.Int("limit", limit), slog.Int("offset", offset))
	return c.JSON(http.StatusOK, adminRoomListResponse{Rooms: out, Total: total, Limit: limit, Offset: offset})
}

// SetUserActive は PATCH /admin/users/:id/active。
func (h *AdminController) SetUserActive(c echo.Context) error {
	adminID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid user id")
	}
	var body setUserActiveRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	if err := h.Admin.SetUserActive(c.Request().Context(), adminID, targetID, body.Active); err != nil {
		return mapAdminUsecaseError(c, err)
	}
	audit.Event("admin.set_user_active",
		slog.String("admin_id", adminID.String()),
		slog.String("target_user_id", targetID.String()),
		slog.Bool("active", body.Active),
	)
	if !body.Active && h.Sockets != nil {
		ch := h.Sockets.CloseUserConnections(targetID)
		h.notifyPresenceAfterForcedDisconnect(c.Request().Context(), ch)
	}
	return c.NoContent(http.StatusNoContent)
}

// SetRoomFrozen は PATCH /admin/rooms/:id/frozen。
func (h *AdminController) SetRoomFrozen(c echo.Context) error {
	adminID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid room id")
	}
	var body setRoomFrozenRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	if err := h.Admin.SetRoomFrozen(c.Request().Context(), adminID, roomID, body.Frozen); err != nil {
		return mapAdminUsecaseError(c, err)
	}
	audit.Event("admin.set_room_frozen",
		slog.String("admin_id", adminID.String()),
		slog.String("room_id", roomID.String()),
		slog.Bool("frozen", body.Frozen),
	)
	return c.NoContent(http.StatusNoContent)
}

// ForceRemoveMember は DELETE /admin/rooms/:id/members/:userId。
func (h *AdminController) ForceRemoveMember(c echo.Context) error {
	adminID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid room id")
	}
	targetID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid user id")
	}
	if err := h.Admin.ForceRemoveMember(c.Request().Context(), adminID, roomID, targetID); err != nil {
		return mapAdminUsecaseError(c, err)
	}
	audit.Event("admin.force_remove_member",
		slog.String("admin_id", adminID.String()),
		slog.String("room_id", roomID.String()),
		slog.String("target_user_id", targetID.String()),
	)
	if h.Sockets != nil {
		ch := h.Sockets.CloseUserConnections(targetID)
		h.notifyPresenceAfterForcedDisconnect(c.Request().Context(), ch)
	}
	return c.NoContent(http.StatusNoContent)
}

func mapAdminUsecaseError(c echo.Context, err error) error {
	status, code, msg := domainErrHTTP(err)
	body := map[string]string{"error": msg}
	if code != "" {
		body["code"] = code
	}
	return c.JSON(status, body)
}
