package controller

import (
	"chat-app/entity"
	"chat-app/middleware"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// CurrentUserID はミドルウェアが設定した認証ユーザー ID を返す。
func CurrentUserID(c echo.Context) (uuid.UUID, error) {
	uid, ok := middleware.AuthUserID(c)
	if !ok {
		return uuid.Nil, echo.NewHTTPError(401, map[string]string{"error": "unauthorized"})
	}
	return uid, nil
}

// CurrentUserRole は認証ユーザーのロールを返す（未設定時は user）。
func CurrentUserRole(c echo.Context) entity.UserRole {
	r, _ := c.Get(middleware.ContextUserRoleKey).(string)
	if r == "" {
		return entity.UserRoleUser
	}
	return entity.UserRole(r)
}
