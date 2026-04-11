package controller

import (
	"errors"
	"net/http"

	"chat-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// AdminController は管理者 API。
type AdminController struct {
	Admin AdminUseCase
}

// NewAdminController は AdminController を生成する。
func NewAdminController(admin AdminUseCase) *AdminController {
	return &AdminController{Admin: admin}
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
	return c.NoContent(http.StatusNoContent)
}

func mapAdminUsecaseError(c echo.Context, err error) error {
	status, msg := adminUsecaseHTTP(err)
	return c.JSON(status, map[string]string{"error": msg})
}

func adminUsecaseHTTP(err error) (int, string) {
	switch {
	case errors.Is(err, usecase.ErrForbiddenAdmin):
		return http.StatusForbidden, err.Error()
	case errors.Is(err, usecase.ErrInvalidUserID),
		errors.Is(err, usecase.ErrInvalidRoomID):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, usecase.ErrRoomNotFound),
		errors.Is(err, usecase.ErrMemberNotFound):
		return http.StatusNotFound, err.Error()
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
