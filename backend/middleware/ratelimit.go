package middleware

import (
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

// IPRateLimiter は IP ごとにトークンバケットを割り当てる。
type IPRateLimiter struct {
	mu    sync.Mutex
	limit rate.Limit
	burst int
	perIP map[string]*rate.Limiter
}

// NewIPRateLimiter は r は「1秒あたりのイベント数」、burst はバースト許容量。
func NewIPRateLimiter(r rate.Limit, burst int) *IPRateLimiter {
	return &IPRateLimiter{limit: r, burst: burst, perIP: make(map[string]*rate.Limiter)}
}

func (l *IPRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	lim, ok := l.perIP[ip]
	if !ok {
		lim = rate.NewLimiter(l.limit, l.burst)
		l.perIP[ip] = lim
	}
	return lim.Allow()
}

// PublicAuthRateLimit は /auth/login と /auth/register のみに適用する。
func PublicAuthRateLimit(login, register *IPRateLimiter) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path
			ip := c.RealIP()
			switch path {
			case "/auth/login":
				if login != nil && !login.allow(ip) {
					return rateLimitedJSON(c)
				}
			case "/auth/register":
				if register != nil && !register.allow(ip) {
					return rateLimitedJSON(c)
				}
			}
			return next(c)
		}
	}
}

// UserRateLimiter はユーザー ID 文字列ごとのリミッタ。
type UserRateLimiter struct {
	mu    sync.Mutex
	limit rate.Limit
	burst int
	per   map[string]*rate.Limiter
}

func NewUserRateLimiter(r rate.Limit, burst int) *UserRateLimiter {
	return &UserRateLimiter{limit: r, burst: burst, per: make(map[string]*rate.Limiter)}
}

func (l *UserRateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	lim, ok := l.per[key]
	if !ok {
		lim = rate.NewLimiter(l.limit, l.burst)
		l.per[key] = lim
	}
	return lim.Allow()
}

// AuthenticatedUserRateLimit は JWT 後の API グループ用。ユーザー単位で制限する。
func AuthenticatedUserRateLimit(lim *UserRateLimiter) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if lim == nil {
				return next(c)
			}
			raw, ok := c.Get(ContextUserIDKey).(uuid.UUID)
			if !ok || raw == uuid.Nil {
				return next(c)
			}
			if !lim.allow(raw.String()) {
				return rateLimitedJSON(c)
			}
			return next(c)
		}
	}
}

// WSConnectIPRateLimit は GET /ws のみ。アップグレード前に IP で制限。
func WSConnectIPRateLimit(lim *IPRateLimiter) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if lim == nil {
				return next(c)
			}
			if c.Request().URL.Path == "/ws" && !lim.allow(c.RealIP()) {
				return rateLimitedJSON(c)
			}
			return next(c)
		}
	}
}

func rateLimitedJSON(c echo.Context) error {
	return c.JSON(http.StatusTooManyRequests, map[string]string{
		"error": "too many requests",
		"code":  "RATE_LIMITED",
	})
}
