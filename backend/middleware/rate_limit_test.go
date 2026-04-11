package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"chat-app/repository"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRedisRegisterRateLimit_429Body(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	lim := repository.NewFixedWindowRateLimiter(rdb)

	e := echo.New()
	e.POST("/auth/register", func(c echo.Context) error {
		return c.NoContent(http.StatusCreated)
	}, RedisRegisterRateLimit(lim, 3, time.Minute))

	const ip = "203.0.113.50"
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte(`{}`)))
		req.RemoteAddr = ip + ":1234"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		require.Equal(t, http.StatusCreated, rec.Code, "i=%d", i)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte(`{}`)))
	req.RemoteAddr = ip + ":1234"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "RATE_LIMITED", body["code"])
	require.Contains(t, body["error"], "too many")
}

func TestRedisLoginRateLimit_EmailKeySeparateFromIP(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	lim := repository.NewFixedWindowRateLimiter(rdb)

	e := echo.New()
	e.POST("/auth/login", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}, RedisLoginRateLimit(lim, 1, time.Minute))

	req1 := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte(`{"email":"a@x.test","password":"x"}`)))
	req1.RemoteAddr = "198.51.100.1:1"
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte(`{"email":"a@x.test","password":"y"}`)))
	req2.RemoteAddr = "198.51.100.1:1"
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusTooManyRequests, rec2.Code)

	// 別メールは別キー
	req3 := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte(`{"email":"b@x.test","password":"z"}`)))
	req3.RemoteAddr = "198.51.100.1:1"
	rec3 := httptest.NewRecorder()
	e.ServeHTTP(rec3, req3)
	require.Equal(t, http.StatusOK, rec3.Code)
}

func TestRedisMessageSendRateLimit_PerRoom(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	lim := repository.NewFixedWindowRateLimiter(rdb)

	e := echo.New()
	uid := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	rid1 := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	rid2 := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")

	h := func(c echo.Context) error { return c.NoContent(http.StatusCreated) }
	g := e.Group("")
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(ContextUserIDKey, uid)
			return next(c)
		}
	})
	g.POST("/rooms/:id/messages", h, RedisMessageSendRateLimit(lim, 1, time.Minute))

	req := httptest.NewRequest(http.MethodPost, "/rooms/"+rid1.String()+"/messages", bytes.NewReader([]byte(`{"content":"x"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/rooms/"+rid1.String()+"/messages", bytes.NewReader([]byte(`{"content":"y"}`)))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusTooManyRequests, rec2.Code)

	req3 := httptest.NewRequest(http.MethodPost, "/rooms/"+rid2.String()+"/messages", bytes.NewReader([]byte(`{"content":"z"}`)))
	req3.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec3 := httptest.NewRecorder()
	e.ServeHTTP(rec3, req3)
	require.Equal(t, http.StatusCreated, rec3.Code)
}

func TestRedisWSConnectRateLimit_PerUser(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	lim := repository.NewFixedWindowRateLimiter(rdb)

	e := echo.New()
	uid := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	e.GET("/ws", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}, func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(ContextUserIDKey, uid)
			return next(c)
		}
	}, RedisWSConnectRateLimit(lim, 1, time.Minute))

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusTooManyRequests, rec2.Code)
}
