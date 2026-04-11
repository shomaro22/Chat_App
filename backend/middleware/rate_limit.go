package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// RedisFixedWindowAllow は固定窓レート制限の判定（テスト用モック可能）。
type RedisFixedWindowAllow interface {
	Allow(ctx context.Context, key string, max int64, window time.Duration) (bool, error)
}

// RateLimitedJSON は 429 を既存 API と同形で返す。
func RateLimitedJSON(c echo.Context) error {
	return c.JSON(http.StatusTooManyRequests, map[string]string{
		"error": "too many requests",
		"code":  "RATE_LIMITED",
	})
}

// RedisLoginRateLimit は POST /auth/login 用。JSON の email があれば rl:login:email:{email}、なければ rl:login:ip:{ip}。
func RedisLoginRateLimit(lim RedisFixedWindowAllow, max int64, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if lim == nil {
				return next(c)
			}
			body, err := io.ReadAll(c.Request().Body)
			if err != nil {
				return next(c)
			}
			c.Request().Body = io.NopCloser(bytes.NewReader(body))
			var partial struct {
				Email string `json:"email"`
			}
			_ = json.Unmarshal(body, &partial)
			email := strings.ToLower(strings.TrimSpace(partial.Email))
			var key string
			if email != "" {
				key = "rl:login:email:" + email
			} else {
				key = "rl:login:ip:" + c.RealIP()
			}
			ok, _ := lim.Allow(c.Request().Context(), key, max, window)
			if !ok {
				return RateLimitedJSON(c)
			}
			return next(c)
		}
	}
}

// RedisRegisterRateLimit は POST /auth/register 用。rl:register:ip:{ip}
func RedisRegisterRateLimit(lim RedisFixedWindowAllow, max int64, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if lim == nil {
				return next(c)
			}
			key := "rl:register:ip:" + c.RealIP()
			ok, _ := lim.Allow(c.Request().Context(), key, max, window)
			if !ok {
				return RateLimitedJSON(c)
			}
			return next(c)
		}
	}
}

// RedisMessageSendRateLimit は POST /rooms/:id/messages 用。rl:message:{user_id}:{room_id}
func RedisMessageSendRateLimit(lim RedisFixedWindowAllow, max int64, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if lim == nil {
				return next(c)
			}
			uid, ok := c.Get(ContextUserIDKey).(uuid.UUID)
			if !ok || uid == uuid.Nil {
				return next(c)
			}
			rid := strings.TrimSpace(c.Param("id"))
			if rid == "" {
				return next(c)
			}
			key := "rl:message:" + uid.String() + ":" + rid
			ok2, _ := lim.Allow(c.Request().Context(), key, max, window)
			if !ok2 {
				return RateLimitedJSON(c)
			}
			return next(c)
		}
	}
}

// RedisMediaUploadRateLimit は POST /media/images 用。rl:upload:{user_id}
func RedisMediaUploadRateLimit(lim RedisFixedWindowAllow, max int64, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if lim == nil {
				return next(c)
			}
			uid, ok := c.Get(ContextUserIDKey).(uuid.UUID)
			if !ok || uid == uuid.Nil {
				return next(c)
			}
			key := "rl:upload:" + uid.String()
			ok2, _ := lim.Allow(c.Request().Context(), key, max, window)
			if !ok2 {
				return RateLimitedJSON(c)
			}
			return next(c)
		}
	}
}

// RedisWSConnectRateLimit は GET /ws 用（JWT 後）。rl:ws:{user_id}
func RedisWSConnectRateLimit(lim RedisFixedWindowAllow, max int64, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if lim == nil {
				return next(c)
			}
			uid, ok := c.Get(ContextUserIDKey).(uuid.UUID)
			if !ok || uid == uuid.Nil {
				return next(c)
			}
			key := "rl:ws:" + uid.String()
			ok2, _ := lim.Allow(c.Request().Context(), key, max, window)
			if !ok2 {
				return RateLimitedJSON(c)
			}
			return next(c)
		}
	}
}
