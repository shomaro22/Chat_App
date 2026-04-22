package middleware

import (
	"strings"

	"chat-app/auth"
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

// SessionValidator は JWT 検証後にアカウントが利用可能かを確認する（無効化ユーザー対策）。
// DB 上のロールを ContextUserRoleKey に上書きしてよい（JWT の role と乖離した場合は DB を優先する）。
type SessionValidator func(c echo.Context, userID uuid.UUID, tokenRole entity.UserRole) error

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

// AuthUserID は JWTAuth が ContextUserIDKey に載せたユーザー ID を返す（型が uuid.UUID 以外の場合も string ならパースする）。
func AuthUserID(c echo.Context) (uuid.UUID, bool) {
	v := c.Get(ContextUserIDKey)
	if v == nil {
		return uuid.Nil, false
	}
	switch x := v.(type) {
	case uuid.UUID:
		if x == uuid.Nil {
			return uuid.Nil, false
		}
		return x, true
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return uuid.Nil, false
		}
		u, err := uuid.Parse(s)
		if err != nil || u == uuid.Nil {
			return uuid.Nil, false
		}
		return u, true
	default:
		return uuid.Nil, false
	}
}

// JWTAuth は TokenParser で検証し、コンテキストに user_id / role を載せる。
// validate が非 nil のときはその後に呼び出し、失敗時は 401 を返す
func JWTAuth(parse TokenParser, validate SessionValidator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			raw := BearerOrQueryToken(c)
			uid, role, err := parse(raw)
			if err != nil {
				return echo.NewHTTPError(401, map[string]string{"error": "unauthorized"})
			}
			c.Set(ContextUserIDKey, uid)
			c.Set(ContextUserRoleKey, string(role))
			if validate != nil {
				if err := validate(c, uid, role); err != nil {
					return err
				}
			}
			return next(c)
		}
	}
}

// RequireAdmin は JWTAuth の後段で、ロールが admin のみ通す。
func RequireAdmin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			r, _ := c.Get(ContextUserRoleKey).(string)
			if !auth.IsAdmin(entity.UserRole(r)) {
				return echo.NewHTTPError(403, map[string]string{"error": "admin only"})
			}
			return next(c)
		}
	}
}
