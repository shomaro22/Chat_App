package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"chat-app/repository"
	"chat-app/usecase"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestAfterAuthRateLimits_WSRetryAfterWithRepository(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	p := repository.DefaultRateLimitPolicy()
	p.WSConnectLimit = 1
	p.WSConnectWindow = time.Minute
	repo := repository.NewRateLimitRepository(rdb, p)
	uc := usecase.NewRateLimitUseCase(repo)

	e := echo.New()
	uid := uuid.New()
	g := e.Group("")
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(ContextUserIDKey, uid)
			return next(c)
		}
	})
	g.Use(AfterAuthRateLimits(uc))
	g.GET("/ws", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req1 := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusTooManyRequests, rec2.Code)
	require.NotEmpty(t, rec2.Header().Get("Retry-After"))
}

func TestRegisterRateLimit_FailOpenOnRedisError(t *testing.T) {
	s := miniredis.RunT(t)
	addr := s.Addr()
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	s.Close()

	p := repository.DefaultRateLimitPolicy()
	p.RegisterLimit = 1
	p.RegisterWindow = time.Minute
	repo := repository.NewRateLimitRepository(rdb, p)
	uc := usecase.NewRateLimitUseCase(repo)

	e := echo.New()
	e.POST("/auth/register", func(c echo.Context) error {
		return c.NoContent(http.StatusCreated)
	}, RegisterRateLimit(uc))

	req1 := httptest.NewRequest(http.MethodPost, "/auth/register", nil)
	req1 = req1.WithContext(context.Background())
	req1.RemoteAddr = "203.0.113.20:1000"
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusCreated, rec1.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/auth/register", nil)
	req2.RemoteAddr = "203.0.113.20:1001"
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusCreated, rec2.Code)
}
