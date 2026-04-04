package controller

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"chat-app/entity"
	"chat-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type RoomController struct {
	Rooms *usecase.RoomUseCase
}

func NewRoomController(rooms *usecase.RoomUseCase) *RoomController {
	return &RoomController{Rooms: rooms}
}

// --- GET /rooms?user_id= ---

type listRoomsResponse struct {
	Rooms []roomJSON `json:"rooms"`
}

// --- POST /rooms/direct ---

type createDirectRoomRequest struct {
	UserID       string `json:"user_id"`
	TargetUserID string `json:"target_user_id"`
}

// --- POST /rooms/groups ---

type createGroupRoomRequest struct {
	CreatedBy string   `json:"created_by"`
	Name      string   `json:"name"`
	MemberIDs []string `json:"member_ids"`
}

type roomJSON struct {
	ID            uuid.UUID  `json:"id"`
	Type          string     `json:"type"`
	Name          *string    `json:"name,omitempty"`
	DirectUserAID *uuid.UUID `json:"direct_user_a_id,omitempty"`
	DirectUserBID *uuid.UUID `json:"direct_user_b_id,omitempty"`
	CreatedBy     uuid.UUID  `json:"created_by"`
	CreatedAt     string     `json:"created_at"`
	UpdatedAt     string     `json:"updated_at"`
}

func roomToJSON(r *entity.Room) roomJSON {
	return roomJSON{
		ID:            r.ID,
		Type:          string(r.Type),
		Name:          r.Name,
		DirectUserAID: r.DirectUserAID,
		DirectUserBID: r.DirectUserBID,
		CreatedBy:     r.CreatedBy,
		CreatedAt:     r.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:     r.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (h *RoomController) ListJoinedRooms(c echo.Context) error {
	raw := c.QueryParam("user_id")
	if raw == "" {
		return jsonError(c, http.StatusBadRequest, "user_id is required")
	}
	uid, err := uuid.Parse(raw)
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid user_id")
	}
	rooms, err := h.Rooms.GetJoinedRooms(c.Request().Context(), uid)
	if err != nil {
		return mapUsecaseError(c, err)
	}
	out := make([]roomJSON, 0, len(rooms))
	for i := range rooms {
		out = append(out, roomToJSON(&rooms[i]))
	}
	return c.JSON(http.StatusOK, listRoomsResponse{Rooms: out})
}

func (h *RoomController) CreateDirectRoom(c echo.Context) error {
	var body createDirectRoomRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	userID, err := uuid.Parse(body.UserID)
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid user_id")
	}
	targetID, err := uuid.Parse(body.TargetUserID)
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid target_user_id")
	}
	room, err := h.Rooms.CreateOrGetDirectRoom(c.Request().Context(), userID, targetID)
	if err != nil {
		return mapUsecaseError(c, err)
	}
	return c.JSON(http.StatusOK, roomToJSON(room))
}

func (h *RoomController) CreateGroupRoom(c echo.Context) error {
	var body createGroupRoomRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	createdBy, err := uuid.Parse(body.CreatedBy)
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid created_by")
	}
	memberIDs := make([]uuid.UUID, 0, len(body.MemberIDs))
	for i, s := range body.MemberIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			return jsonError(c, http.StatusBadRequest, "invalid member_ids["+strconv.Itoa(i)+"]")
		}
		memberIDs = append(memberIDs, id)
	}
	room, err := h.Rooms.CreateGroupRoom(c.Request().Context(), createdBy, body.Name, memberIDs)
	if err != nil {
		return mapUsecaseError(c, err)
	}
	return c.JSON(http.StatusCreated, roomToJSON(room))
}

func jsonError(c echo.Context, code int, msg string) error {
	return c.JSON(code, map[string]string{"error": msg})
}

func mapUsecaseError(c echo.Context, err error) error {
	status, msg := usecaseErrorHTTP(err)
	return c.JSON(status, map[string]string{"error": msg})
}

func usecaseErrorHTTP(err error) (int, string) {
	switch {
	case errors.Is(err, usecase.ErrInvalidUserID),
		errors.Is(err, usecase.ErrInvalidCreatorID),
		errors.Is(err, usecase.ErrSelfDirect),
		errors.Is(err, usecase.ErrGroupNameRequired),
		errors.Is(err, usecase.ErrGroupTooFewMembers),
		errors.Is(err, usecase.ErrInvalidGroupMembers):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, usecase.ErrUserNotFound),
		errors.Is(err, usecase.ErrTargetUserNotFound):
		return http.StatusNotFound, err.Error()
	case errors.Is(err, usecase.ErrInactiveUser):
		return http.StatusForbidden, err.Error()
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
