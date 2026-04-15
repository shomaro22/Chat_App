//go:build integration

package e2e_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"strings"
	"testing"
	"time"

	"chat-app/controller"
	"chat-app/repository"

	"github.com/alicebob/miniredis/v2"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRateLimitFlow_HTTPPolicies_LoginRegisterMessageMedia(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	p := repository.DefaultRateLimitPolicy()
	p.LoginIPLimit = 2
	p.LoginEmailLimit = 1
	p.RegisterLimit = 1
	p.MessageLimit = 1
	p.MediaLimit = 1
	p.WSConnectLimit = 10

	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	e := newTestEchoWithRateLimiter(t, gormDB, publicRoot, "", repository.NewRateLimitRepository(rdb, p))

	const pass = "password12"
	doRequestWithIP(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Alice", "email": "rla@e2e.test", "password": pass,
	}, "203.0.113.1", http.StatusCreated)
	recB := doRequestWithIP(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Bob", "email": "rlb@e2e.test", "password": pass,
	}, "203.0.113.2", http.StatusCreated)
	doRequestWithIP(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Carol", "email": "rlc@e2e.test", "password": pass,
	}, "203.0.113.3", http.StatusCreated)
	var bobReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &bobReg))

	doRequestWithIP(t, e, http.MethodPost, "/auth/login", "", map[string]string{
		"email": "rla@e2e.test", "password": pass,
	}, "198.51.100.10", http.StatusOK)
	loginEmail429 := doRequestWithIP(t, e, http.MethodPost, "/auth/login", "", map[string]string{
		"email": "rla@e2e.test", "password": pass,
	}, "198.51.100.10", http.StatusTooManyRequests)
	require.NotEmpty(t, loginEmail429.Header().Get("Retry-After"))
	loginIPConsumed := doRequestWithIP(t, e, http.MethodPost, "/auth/login", "", map[string]string{
		"email": "rlb@e2e.test", "password": pass,
	}, "198.51.100.10", http.StatusTooManyRequests)
	require.NotEmpty(t, loginIPConsumed.Header().Get("Retry-After"))
	doRequestWithIP(t, e, http.MethodPost, "/auth/login", "", map[string]string{
		"email": "rla@e2e.test", "password": pass,
	}, "198.51.100.11", http.StatusTooManyRequests)
	loginIP429 := doRequestWithIP(t, e, http.MethodPost, "/auth/login", "", map[string]string{
		"email": "rlc@e2e.test", "password": pass,
	}, "198.51.100.10", http.StatusTooManyRequests)
	require.NotEmpty(t, loginIP429.Header().Get("Retry-After"))

	doRequestWithIP(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Reg1", "email": "reg1@e2e.test", "password": pass,
	}, "192.0.2.88", http.StatusCreated)
	reg429 := doRequestWithIP(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Reg2", "email": "reg2@e2e.test", "password": pass,
	}, "192.0.2.88", http.StatusTooManyRequests)
	require.NotEmpty(t, reg429.Header().Get("Retry-After"))

	tokenAlice := login(t, e, "rla@e2e.test", pass)
	recG1 := doRequest(t, e, http.MethodPost, "/rooms/groups", tokenAlice, map[string]any{
		"name":       "RL-1",
		"member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	recG2 := doRequest(t, e, http.MethodPost, "/rooms/groups", tokenAlice, map[string]any{
		"name":       "RL-2",
		"member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var room1 roomIDBody
	var room2 roomIDBody
	require.NoError(t, json.Unmarshal(recG1.Body.Bytes(), &room1))
	require.NoError(t, json.Unmarshal(recG2.Body.Bytes(), &room2))

	doRequest(t, e, http.MethodPost, "/rooms/"+room1.ID.String()+"/messages", tokenAlice, map[string]any{
		"content": "first",
	}, http.StatusCreated)
	msg429 := doRequest(t, e, http.MethodPost, "/rooms/"+room1.ID.String()+"/messages", tokenAlice, map[string]any{
		"content": "second",
	}, http.StatusTooManyRequests)
	require.NotEmpty(t, msg429.Header().Get("Retry-After"))
	doRequest(t, e, http.MethodPost, "/rooms/"+room2.ID.String()+"/messages", tokenAlice, map[string]any{
		"content": "other-room",
	}, http.StatusCreated)

	doUploadImage(t, e, tokenAlice, "a.png", []byte{0x89, 0x50, 0x4e, 0x47}, http.StatusCreated)
	media429 := doUploadImage(t, e, tokenAlice, "b.png", []byte{0x89, 0x50, 0x4e, 0x47}, http.StatusTooManyRequests)
	require.NotEmpty(t, media429.Header().Get("Retry-After"))
}

func TestRateLimitFlow_WSConnectReconnectAndNoConflict(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	p := repository.DefaultRateLimitPolicy()
	p.WSConnectLimit = 1
	p.WSConnectWindow = time.Minute
	p.RegisterLimit = 100
	p.LoginIPLimit = 100
	p.LoginEmailLimit = 100
	p.MessageLimit = 100
	p.MediaLimit = 100

	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	e := newTestEchoWithRateLimiter(t, gormDB, publicRoot, "", repository.NewRateLimitRepository(rdb, p))
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Alice", "email": "wsrlalice@e2e.test", "password": pass,
	}, http.StatusCreated)
	recB := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Bob", "email": "wsrlbob@e2e.test", "password": pass,
	}, http.StatusCreated)
	var bobReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &bobReg))
	tokenAlice := login(t, e, "wsrlalice@e2e.test", pass)
	tokenBob := login(t, e, "wsrlbob@e2e.test", pass)

	recG := doRequest(t, e, http.MethodPost, "/rooms/groups", tokenAlice, map[string]any{
		"name":       "WS-RL",
		"member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var room roomIDBody
	require.NoError(t, json.Unmarshal(recG.Body.Bytes(), &room))
	roomID := room.ID.String()

	conn := mustDialWS(t, srv.URL, tokenAlice)
	t.Cleanup(func() { _ = conn.Close() })
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)
	var ready map[string]any
	require.NoError(t, json.Unmarshal(data, &ready))
	require.Equal(t, controller.WSEventSessionReady, ready["type"])

	require.NoError(t, conn.WriteJSON(map[string]any{
		"type":           controller.WSClientSubscribePrimary,
		"room_id":        roomID,
		"schema_version": controller.WSClientSchemaVersion,
	}))
	_, data, err = conn.ReadMessage()
	require.NoError(t, err)
	var sub map[string]any
	require.NoError(t, json.Unmarshal(data, &sub))
	require.Equal(t, controller.WSEventSubscribed, sub["type"])
	require.NoError(t, conn.WriteJSON(map[string]any{
		"type":           controller.WSClientUnsubscribePrimary,
		"room_id":        roomID,
		"schema_version": controller.WSClientSchemaVersion,
	}))
	_, data, err = conn.ReadMessage()
	require.NoError(t, err)
	var unsub map[string]any
	require.NoError(t, json.Unmarshal(data, &unsub))
	require.Equal(t, controller.WSEventUnsubscribed, unsub["type"])

	_, resp, err := dialWSRaw(srv.URL, tokenAlice)
	require.Error(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	require.NotEmpty(t, resp.Header.Get("Retry-After"))

	require.NoError(t, conn.WriteJSON(map[string]any{
		"type":           controller.WSClientPing,
		"schema_version": controller.WSClientSchemaVersion,
	}))
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, data, err = conn.ReadMessage()
	require.NoError(t, err)
	var pong map[string]any
	require.NoError(t, json.Unmarshal(data, &pong))
	require.Equal(t, controller.WSEventPong, pong["type"])

	for i := 0; i < 2; i++ {
		_, loopResp, loopErr := dialWSRaw(srv.URL, tokenBob)
		require.Error(t, loopErr)
		require.NotNil(t, loopResp)
		require.Equal(t, http.StatusTooManyRequests, loopResp.StatusCode)
	}
}

func TestRateLimitFlow_RedisFailOpen(t *testing.T) {
	s := miniredis.RunT(t)
	addr := s.Addr()
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	p := repository.DefaultRateLimitPolicy()
	p.RegisterLimit = 1
	p.RegisterWindow = time.Minute
	s.Close()

	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	e := newTestEchoWithRateLimiter(t, gormDB, publicRoot, "", repository.NewRateLimitRepository(rdb, p))

	const pass = "password12"
	doRequestWithIP(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "FailOpen1", "email": "fo1@e2e.test", "password": pass,
	}, "198.51.100.44", http.StatusCreated)
	doRequestWithIP(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "FailOpen2", "email": "fo2@e2e.test", "password": pass,
	}, "198.51.100.44", http.StatusCreated)
}

func doRequestWithIP(t *testing.T, e *echo.Echo, method, path, bearer string, body any, ip string, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != nil {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	if bearer != "" {
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+bearer)
	}
	req.RemoteAddr = ip + ":12345"
	out := httptest.NewRecorder()
	e.ServeHTTP(out, req)
	require.Equal(t, wantStatus, out.Code, "body=%s", out.Body.String())
	return out
}

func doUploadImage(t *testing.T, e *echo.Echo, bearer, filename string, file []byte, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", "image/png")
	part, err := w.CreatePart(h)
	require.NoError(t, err)
	_, err = part.Write(file)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, "/media/images", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+bearer)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, wantStatus, rec.Code, "body=%s", rec.Body.String())
	return rec
}

func mustDialWS(t *testing.T, serverURL, token string) *websocket.Conn {
	t.Helper()
	conn, _, err := dialWSRaw(serverURL, token)
	require.NoError(t, err)
	return conn
}

func dialWSRaw(serverURL, token string) (*websocket.Conn, *http.Response, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, nil, err
	}
	u.Scheme = "ws"
	u.Path = "/ws"
	q := u.Query()
	q.Set("access_token", strings.TrimSpace(token))
	u.RawQuery = q.Encode()
	return websocket.DefaultDialer.Dial(u.String(), nil)
}
