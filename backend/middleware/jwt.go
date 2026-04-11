package middleware

import (
	"strings"

	"chat-app/entity"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Echo コンテキストに認証情報を格納するキー。
const (
	ContextUserIDKey   = "auth_user_id"
	ContextUserRoleKey = "auth_user_role"
)

// TokenParser は JWT 等からユーザー ID とロールを解決する。
type TokenParser func(raw string) (uuid.UUID, entity.UserRole, error)

// BearerOrQueryToken は Authorization: Bearer または ?access_token= を読む。
func BearerOrQueryToken(c echo.Context) string {
	if t := c.QueryParam("access_token"); t != "" {
		return t
	}
	h := c.Request().Header.Get("Authorization")
	const p = "Bearer "
	if len(h) > len(p) && strings.EqualFold(h[:len(p)], p) {
		return strings.TrimSpace(h[len(p):])
	}
	return ""
}

// JWTAuth は TokenParser で検証し、コンテキストに user_id / role を載せる。
func JWTAuth(parse TokenParser) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			raw := BearerOrQueryToken(c)
			uid, role, err := parse(raw)
			if err != nil {
				return echo.NewHTTPError(401, map[string]string{"error": "unauthorized"})
			}
			c.Set(ContextUserIDKey, uid)
			c.Set(ContextUserRoleKey, string(role))
			return next(c)
		}
	}
}

// RequireAdmin は JWTAuth の後段で、ロールが admin のみ通す。
func RequireAdmin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			r, _ := c.Get(ContextUserRoleKey).(string)
			if entity.UserRole(r) != entity.UserRoleAdmin {
				return echo.NewHTTPError(403, map[string]string{"error": "admin only"})
			}
			return next(c)
		}
	}
}
