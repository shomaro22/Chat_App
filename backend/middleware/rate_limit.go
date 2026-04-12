package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"chat-app/usecase"

	"github.com/labstack/echo/v4"
)

// RateLimitedJSON は 429 を既存 API と同形で返し、Retry-After と retry_after_seconds を付与する。
func RateLimitedJSON(c echo.Context, retryAfterSec int64) error {
	if retryAfterSec > 0 {
		c.Response().Header().Set("Retry-After", strconv.FormatInt(retryAfterSec, 10))
	}
	body := map[string]any{
		"error": "too many requests",
		"code":  "RATE_LIMITED",
	}
	if retryAfterSec > 0 {
		body["retry_after_seconds"] = retryAfterSec
	}
	return c.JSON(http.StatusTooManyRequests, body)
}

func applyRateLimit(c echo.Context, res usecase.RateLimitResult, next echo.HandlerFunc) error {
	if res.Err != nil {
		log.Printf("rate limit: redis error (fail-open): %v", res.Err)
	}
	if !res.Allowed {
		return RateLimitedJSON(c, res.RetryAfterSeconds)
	}
	return next(c)
}

// LoginRateLimit は POST /auth/login 用。ボディから email を読み取り RateLimitUseCase に委譲する。
func LoginRateLimit(uc *usecase.RateLimitUseCase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if uc == nil {
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
			res := uc.CheckLogin(c.Request().Context(), partial.Email, c.RealIP())
			return applyRateLimit(c, res, next)
		}
	}
}

// RegisterRateLimit は POST /auth/register 用。
func RegisterRateLimit(uc *usecase.RateLimitUseCase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if uc == nil {
				return next(c)
			}
			res := uc.CheckRegister(c.Request().Context(), c.RealIP())
			return applyRateLimit(c, res, next)
		}
	}
}

// AfterAuthRateLimits は JWT 通過後の API について、登録パス（c.Path()）に応じてレート制限する。
// ルート個別に付けたミドルウェアは Echo のマッチング順の影響を受けやすいため、api.Use(jwt); api.Use(AfterAuthRateLimits(...)) で適用する。
func AfterAuthRateLimits(uc *usecase.RateLimitUseCase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if uc == nil {
				return next(c)
			}
			p := c.Path()
			m := c.Request().Method
			uid, ok := AuthUserID(c)
			if !ok {
				return next(c)
			}
			var res usecase.RateLimitResult
			switch {
			case m == http.MethodPost && p == "/rooms/:id/messages":
				res = uc.CheckMessageSend(c.Request().Context(), uid, strings.TrimSpace(c.Param("id")))
			case m == http.MethodPost && p == "/media/images":
				res = uc.CheckMediaUpload(c.Request().Context(), uid)
			case m == http.MethodGet && p == "/ws":
				res = uc.CheckWSConnect(c.Request().Context(), uid)
			default:
				return next(c)
			}
			return applyRateLimit(c, res, next)
		}
	}
}
