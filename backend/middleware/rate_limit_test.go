package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"chat-app/entity"
	"chat-app/repository"
	"chat-app/usecase"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRegisterRateLimit_429BodyAndRetryAfter(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	p := usecase.DefaultRateLimitPolicy()
	uc := usecase.NewRateLimitUseCase(repository.NewFixedWindowRateLimiter(rdb), usecase.RateLimitPolicy{
		RegisterLimit:    3,
		RegisterWindow:   p.RegisterWindow,
		LoginLimit:       p.LoginLimit,
		LoginWindow:      p.LoginWindow,
		MessageLimit:     p.MessageLimit,
		MessageWindow:    p.MessageWindow,
		MediaLimit:       p.MediaLimit,
		MediaWindow:      p.MediaWindow,
		WSConnectLimit:   p.WSConnectLimit,
		WSConnectWindow:  p.WSConnectWindow,
	})

	e := echo.New()
	e.POST("/auth/register", func(c echo.Context) error {
		return c.NoContent(http.StatusCreated)
	}, RegisterRateLimit(uc))

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
	ra := rec.Header().Get("Retry-After")
	require.NotEmpty(t, ra)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "RATE_LIMITED", body["code"])
	require.Contains(t, body["error"], "too many")
	require.NotNil(t, body["retry_after_seconds"])
}

func TestLoginRateLimit_EmailKeySeparateFromIP(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	repo := repository.NewFixedWindowRateLimiter(rdb)
	p := usecase.DefaultRateLimitPolicy()
	p.LoginLimit = 1
	uc := usecase.NewRateLimitUseCase(repo, p)

	e := echo.New()
	e.POST("/auth/login", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}, LoginRateLimit(uc))

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

	req3 := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte(`{"email":"b@x.test","password":"z"}`)))
	req3.RemoteAddr = "198.51.100.1:1"
	rec3 := httptest.NewRecorder()
	e.ServeHTTP(rec3, req3)
	require.Equal(t, http.StatusOK, rec3.Code)
}

func TestMessageSendRateLimit_PerRoom(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	repo := repository.NewFixedWindowRateLimiter(rdb)
	p := usecase.DefaultRateLimitPolicy()
	p.MessageLimit = 1
	uc := usecase.NewRateLimitUseCase(repo, p)

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
	g.Use(AfterAuthRateLimits(uc))
	g.POST("/rooms/:id/messages", h)

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

// Echo の api.Use(jwt); api.Use(AfterAuthRateLimits) と同じ順で 429 になること。
func TestMessageSendRateLimit_AfterJWTAuthLikeRouter(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	repo := repository.NewFixedWindowRateLimiter(rdb)
	p := usecase.DefaultRateLimitPolicy()
	p.MessageLimit = 2
	uc := usecase.NewRateLimitUseCase(repo, p)

	uid := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	rid := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	parse := func(raw string) (uuid.UUID, entity.UserRole, error) {
		if raw == "" {
			return uuid.Nil, "", errors.New("no token")
		}
		return uid, entity.UserRoleUser, nil
	}

	e := echo.New()
	g := e.Group("")
	g.Use(JWTAuth(parse, nil))
	g.Use(AfterAuthRateLimits(uc))
	g.POST("/rooms/:id/messages", func(c echo.Context) error {
		return c.NoContent(http.StatusCreated)
	})

	path := "/rooms/" + rid.String() + "/messages"
	do := func() int {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"content":"x"}`))
		req.Header.Set(echo.HeaderAuthorization, "Bearer any")
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec.Code
	}
	require.Equal(t, http.StatusCreated, do())
	require.Equal(t, http.StatusCreated, do())
	require.Equal(t, http.StatusTooManyRequests, do())
}

func TestWSConnectRateLimit_PerUser(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	repo := repository.NewFixedWindowRateLimiter(rdb)
	p := usecase.DefaultRateLimitPolicy()
	p.WSConnectLimit = 1
	uc := usecase.NewRateLimitUseCase(repo, p)

	e := echo.New()
	uid := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	g := e.Group("")
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(ContextUserIDKey, uid)
			return next(c)
		}
	})
	g.Use(AfterAuthRateLimits(uc))
	g.GET("/ws", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusTooManyRequests, rec2.Code)
}
