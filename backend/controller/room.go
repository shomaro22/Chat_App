package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"chat-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// RoomController はルーム関連の HTTP ハンドラ。
type RoomController struct {
	Rooms RoomUseCase
}

// NewRoomController は RoomController を生成する。
func NewRoomController(rooms RoomUseCase) *RoomController {
	return &RoomController{Rooms: rooms}
}

// ListJoinedRooms は GET /rooms?limit=&offset=&sort= で参加ルーム一覧を返す。
func (h *RoomController) ListJoinedRooms(c echo.Context) error {
	uid, err := CurrentUserID(c)
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
	offset := 0
	if raw := c.QueryParam("offset"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return jsonError(c, http.StatusBadRequest, "invalid offset")
		}
		offset = n
	}
	updatedDesc := true
	sortLabel := "updated_desc"
	switch strings.ToLower(strings.TrimSpace(c.QueryParam("sort"))) {
	case "", "updated_desc":
		updatedDesc = true
		sortLabel = "updated_desc"
	case "updated_asc":
		updatedDesc = false
		sortLabel = "updated_asc"
	default:
		return jsonError(c, http.StatusBadRequest, "invalid sort (use updated_desc or updated_asc)")
	}
	summaries, err := h.Rooms.GetJoinedRoomSummaries(c.Request().Context(), uid, limit, offset, updatedDesc)
	if err != nil {
		return mapDomainError(c, err)
	}
	out := make([]roomListItemJSON, 0, len(summaries))
	for i := range summaries {
		out = append(out, joinedRoomSummaryToJSON(summaries[i]))
	}
	effectiveLimit := limit
	if effectiveLimit <= 0 {
		effectiveLimit = DefaultJoinedRoomsLimit
	}
	return c.JSON(http.StatusOK, listRoomsEnrichedResponse{Rooms: out, Sort: sortLabel, Limit: effectiveLimit, Offset: offset})
}

// GetRoomDetail は GET /rooms/:id でルーム詳細を返す。
func (h *RoomController) GetRoomDetail(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid room id")
	}
	uid, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	detail, err := h.Rooms.GetRoomDetail(c.Request().Context(), roomID, uid)
	if err != nil {
		return mapDomainError(c, err)
	}
	return c.JSON(http.StatusOK, roomDetailToJSON(detail))
}

// AddRoomMember は POST /rooms/:id/members でグループにメンバーを追加する。
func (h *RoomController) AddRoomMember(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid room id")
	}
	actorID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	var body addRoomMemberRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	memberRef := strings.TrimSpace(body.MemberID)
	if memberRef == "" {
		return jsonError(c, http.StatusBadRequest, "member_id is required")
	}
	memberID, err := h.Rooms.ResolveMemberRef(c.Request().Context(), memberRef)
	if err != nil {
		return mapDomainError(c, err)
	}
	if err := h.Rooms.AddGroupMember(c.Request().Context(), roomID, actorID, memberID); err != nil {
		return mapDomainError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// RemoveRoomMember は DELETE /rooms/:id/members/:userId で退出または作成者によるキックを行う。
func (h *RoomController) RemoveRoomMember(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid room id")
	}
	targetID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid user id")
	}
	actorID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	if err := h.Rooms.RemoveRoomMember(c.Request().Context(), roomID, actorID, targetID); err != nil {
		return mapDomainError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// UpdateGroupRoom は PATCH /rooms/:id でグループ名を更新する（作成者のみ）。
func (h *RoomController) UpdateGroupRoom(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid room id")
	}
	actorID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	var body patchGroupRoomRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	if err := h.Rooms.UpdateGroupName(c.Request().Context(), roomID, actorID, body.Name); err != nil {
		return mapDomainError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// CloseRoom は DELETE /rooms/:id でルームを論理削除する。
func (h *RoomController) CloseRoom(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid room id")
	}
	actorID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	if err := h.Rooms.CloseRoom(c.Request().Context(), roomID, actorID); err != nil {
		return mapDomainError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// CreateDirectRoom は POST /rooms/direct で DM を作成または取得する。
func (h *RoomController) CreateDirectRoom(c echo.Context) error {
	userID, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	var body createDirectRoomRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	ref := strings.TrimSpace(body.TargetUserID)
	if ref == "" {
		return jsonError(c, http.StatusBadRequest, "target_user_id is required")
	}
	targetID, err := h.Rooms.ResolveMemberRef(c.Request().Context(), ref)
	if err != nil {
		return mapDomainError(c, err)
	}
	room, err := h.Rooms.CreateOrGetDirectRoom(c.Request().Context(), userID, targetID)
	if err != nil {
		return mapDomainError(c, err)
	}
	return c.JSON(http.StatusOK, roomToJSON(room))
}

// CreateGroupRoom は POST /rooms/groups でグループルームを作成する。
func (h *RoomController) CreateGroupRoom(c echo.Context) error {
	createdBy, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	var body createGroupRoomRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	memberIDs := make([]uuid.UUID, 0, len(body.MemberIDs))
	for i, s := range body.MemberIDs {
		id, err := h.Rooms.ResolveMemberRef(c.Request().Context(), s)
		if err != nil {
			if errors.Is(err, usecase.ErrInvalidUserID) {
				return jsonError(c, http.StatusBadRequest, "invalid member_ids["+strconv.Itoa(i)+"]")
			}
			return mapDomainError(c, err)
		}
		memberIDs = append(memberIDs, id)
	}
	room, err := h.Rooms.CreateGroupRoom(c.Request().Context(), createdBy, body.Name, memberIDs)
	if err != nil {
		return mapDomainError(c, err)
	}
	return c.JSON(http.StatusCreated, roomToJSON(room))
}

// jsonError は JSON 形式のエラーレスポンスを返す。
func jsonError(c echo.Context, code int, msg string) error {
	return c.JSON(code, map[string]string{"error": msg})
}
