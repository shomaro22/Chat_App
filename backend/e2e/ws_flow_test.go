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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"chat-app/controller"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestWS_ChatFlow_SubscribeSendReadLeave(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	e := newTestEcho(t, gormDB, publicRoot, "")

	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Alice", "email": "wsalice@e2e.test", "password": pass,
	}, http.StatusCreated)
	recB := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Bob", "email": "wsbob@e2e.test", "password": pass,
	}, http.StatusCreated)
	var bobReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &bobReg))

	tokenAlice := login(t, e, "wsalice@e2e.test", pass)
	tokenBob := login(t, e, "wsbob@e2e.test", pass)

	recG := doRequest(t, e, http.MethodPost, "/rooms/groups", tokenAlice, map[string]any{
		"name":       "WSRoom",
		"member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var room roomIDBody
	require.NoError(t, json.Unmarshal(recG.Body.Bytes(), &room))
	roomID := room.ID.String()

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	u.Scheme = "ws"
	u.Path = "/ws"
	q := u.Query()
	q.Set("access_token", tokenBob)
	u.RawQuery = q.Encode()

	bobConn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = bobConn.Close() })

	_, data, err := bobConn.ReadMessage()
	require.NoError(t, err)
	var ready map[string]any
	require.NoError(t, json.Unmarshal(data, &ready))
	require.Equal(t, controller.WSEventSessionReady, ready["type"])

	sub := map[string]any{"type": controller.WSClientSubscribePrimary, "room_id": roomID, "schema_version": controller.WSClientSchemaVersion}
	require.NoError(t, bobConn.WriteJSON(sub))
	_, data, err = bobConn.ReadMessage()
	require.NoError(t, err)
	var ack map[string]any
	require.NoError(t, json.Unmarshal(data, &ack))
	require.Equal(t, controller.WSEventSubscribed, ack["type"])
	require.Equal(t, roomID, ack["room_id"])

	txt := "ws hello"
	recMsg := doRequest(t, e, http.MethodPost, "/rooms/"+roomID+"/messages", tokenAlice, map[string]any{"content": txt}, http.StatusCreated)
	var created createMessageAPIResponse
	require.NoError(t, json.Unmarshal(recMsg.Body.Bytes(), &created))
	msgID := created.Message.ID.String()

	_ = bobConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, data, err = bobConn.ReadMessage()
	require.NoError(t, err)
	var evt map[string]any
	require.NoError(t, json.Unmarshal(data, &evt))
	require.Equal(t, controller.WSEventMessageCreated, evt["type"])
	require.Equal(t, roomID, evt["room_id"])
	msgObj, ok := evt["message"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, txt, msgObj["content"])

	// Alice も同ルームを購読して read_updated を受け取る
	u2, uerr := url.Parse(srv.URL)
	require.NoError(t, uerr)
	u2.Scheme = "ws"
	u2.Path = "/ws"
	q2 := u2.Query()
	q2.Set("access_token", tokenAlice)
	u2.RawQuery = q2.Encode()
	aliceConn, _, err := websocket.DefaultDialer.Dial(u2.String(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = aliceConn.Close() })
	_, _, _ = aliceConn.ReadMessage() // session_ready
	require.NoError(t, aliceConn.WriteJSON(map[string]any{"type": controller.WSClientSubscribePrimary, "room_id": roomID}))
	_, _, _ = aliceConn.ReadMessage() // subscribed

	doRequest(t, e, http.MethodPost, "/rooms/"+roomID+"/read", tokenBob, map[string]string{"message_id": msgID}, http.StatusNoContent)

	_ = aliceConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, data, err = aliceConn.ReadMessage()
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &evt))
	require.Equal(t, controller.WSEventReadUpdated, evt["type"])
	require.Equal(t, bobReg.User.ID.String(), evt["user_id"])

	// Bob が自ら退室 → ルーム詳細 403
	doRequest(t, e, http.MethodDelete, "/rooms/"+roomID+"/members/"+bobReg.User.ID.String(), tokenBob, nil, http.StatusNoContent)
	rec403 := httptest.NewRecorder()
	req403 := httptest.NewRequest(http.MethodGet, "/rooms/"+roomID, nil)
	req403.Header.Set("Authorization", "Bearer "+tokenBob)
	e.ServeHTTP(rec403, req403)
	require.Equal(t, http.StatusForbidden, rec403.Code)
	var eb map[string]any
	require.NoError(t, json.Unmarshal(rec403.Body.Bytes(), &eb))
	require.Equal(t, "NOT_ROOM_MEMBER", eb["code"])
}

func TestWS_JSONPingPong(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	e := newTestEcho(t, gormDB, publicRoot, "")

	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Pinger", "email": "pinger@e2e.test", "password": pass,
	}, http.StatusCreated)
	tok := login(t, e, "pinger@e2e.test", pass)

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	u.Scheme = "ws"
	u.Path = "/ws"
	q := u.Query()
	q.Set("access_token", tok)
	u.RawQuery = q.Encode()
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	_, _, err = conn.ReadMessage()
	require.NoError(t, err)

	require.NoError(t, conn.WriteJSON(map[string]any{"type": controller.WSClientPing, "schema_version": controller.WSClientSchemaVersion}))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)
	var pong map[string]any
	require.NoError(t, json.Unmarshal(data, &pong))
	require.Equal(t, controller.WSEventPong, pong["type"])
}

func TestWS_SendForbiddenAfterLeave(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	e := newTestEcho(t, gormDB, publicRoot, "")

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Alice", "email": "leavealice@e2e.test", "password": pass,
	}, http.StatusCreated)
	recB := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Bob", "email": "leavebob@e2e.test", "password": pass,
	}, http.StatusCreated)
	var bobReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &bobReg))

	tokenAlice := login(t, e, "leavealice@e2e.test", pass)
	tokenBob := login(t, e, "leavebob@e2e.test", pass)

	recG := doRequest(t, e, http.MethodPost, "/rooms/groups", tokenAlice, map[string]any{
		"name":       "LeaveRoom",
		"member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var room roomIDBody
	require.NoError(t, json.Unmarshal(recG.Body.Bytes(), &room))
	roomID := room.ID.String()

	doRequest(t, e, http.MethodDelete, "/rooms/"+roomID+"/members/"+bobReg.User.ID.String(), tokenBob, nil, http.StatusNoContent)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID+"/messages", strings.NewReader(`{"content":"nope"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+tokenBob)
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
	var eb apiErrBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &eb))
	require.Equal(t, "NOT_ROOM_MEMBER", eb.Code)
}

func TestWS_FrozenRoomSubscribeError(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(publicRoot, "uploads"), 0o755))
	e := newTestEcho(t, gormDB, publicRoot, "freezeadmin@e2e.test")

	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Admin", "email": "freezeadmin@e2e.test", "password": pass,
	}, http.StatusCreated)
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Alice", "email": "freezealice@e2e.test", "password": pass,
	}, http.StatusCreated)
	recB := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Bob", "email": "freezebob@e2e.test", "password": pass,
	}, http.StatusCreated)
	var bobReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &bobReg))

	tokenAdmin := login(t, e, "freezeadmin@e2e.test", pass)
	tokenAlice := login(t, e, "freezealice@e2e.test", pass)
	tokenBob := login(t, e, "freezebob@e2e.test", pass)

	recG := doRequest(t, e, http.MethodPost, "/rooms/groups", tokenAlice, map[string]any{
		"name":       "FrozenWS",
		"member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var room roomIDBody
	require.NoError(t, json.Unmarshal(recG.Body.Bytes(), &room))
	roomID := room.ID.String()

	doRequest(t, e, http.MethodPatch, "/admin/rooms/"+roomID+"/frozen", tokenAdmin, map[string]bool{"frozen": true}, http.StatusNoContent)

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	u.Scheme = "ws"
	u.Path = "/ws"
	q := u.Query()
	q.Set("access_token", tokenBob)
	u.RawQuery = q.Encode()
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	_, _, _ = conn.ReadMessage() // session_ready
	require.NoError(t, conn.WriteJSON(map[string]any{"type": "subscribe", "room_id": roomID}))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)
	var wse map[string]any
	require.NoError(t, json.Unmarshal(data, &wse))
	require.Equal(t, controller.WSEventError, wse["type"])
	require.Equal(t, controller.WSCodeRoomFrozen, wse["code"])

	// HTTP 送信も拒否（凍結）
	recPost := httptest.NewRecorder()
	reqPost := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID+"/messages", strings.NewReader(`{"content":"blocked"}`))
	reqPost.Header.Set("Content-Type", echo.MIMEApplicationJSON)
	reqPost.Header.Set("Authorization", "Bearer "+tokenAlice)
	e.ServeHTTP(recPost, reqPost)
	require.Equal(t, http.StatusForbidden, recPost.Code)
	var eb apiErrBody
	require.NoError(t, json.Unmarshal(recPost.Body.Bytes(), &eb))
	require.Equal(t, "ROOM_FROZEN", eb.Code)
}

func TestWS_AdminDeactivateClosesConnection(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	e := newTestEcho(t, gormDB, publicRoot, "adminws@e2e.test")

	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Admin", "email": "adminws@e2e.test", "password": pass,
	}, http.StatusCreated)
	recV := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "User", "email": "victim@e2e.test", "password": pass,
	}, http.StatusCreated)
	var victimReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recV.Body.Bytes(), &victimReg))
	victimID := victimReg.User.ID.String()

	adminTok := login(t, e, "adminws@e2e.test", pass)
	victimTok := login(t, e, "victim@e2e.test", pass)

	u, uerr := url.Parse(srv.URL)
	require.NoError(t, uerr)
	u.Scheme = "ws"
	u.Path = "/ws"
	q := u.Query()
	q.Set("access_token", victimTok)
	u.RawQuery = q.Encode()
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)

	doRequest(t, e, http.MethodPatch, "/admin/users/"+victimID+"/active", adminTok, map[string]bool{"active": false}, http.StatusNoContent)

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, _, err = conn.ReadMessage()
	require.Error(t, err)
	require.True(t, websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) ||
		strings.Contains(err.Error(), "use of closed network connection") ||
		strings.Contains(err.Error(), "EOF"))
}

func TestWS_MessageDeletedBroadcast(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	e := newTestEcho(t, gormDB, publicRoot, "")

	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Alice", "email": "delalice@e2e.test", "password": pass,
	}, http.StatusCreated)
	recB := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Bob", "email": "delbob@e2e.test", "password": pass,
	}, http.StatusCreated)
	var bobReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &bobReg))
	tokenAlice := login(t, e, "delalice@e2e.test", pass)
	tokenBob := login(t, e, "delbob@e2e.test", pass)

	recG := doRequest(t, e, http.MethodPost, "/rooms/groups", tokenAlice, map[string]any{
		"name":       "DelRoom",
		"member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var room roomIDBody
	require.NoError(t, json.Unmarshal(recG.Body.Bytes(), &room))
	roomID := room.ID.String()

	recM := doRequest(t, e, http.MethodPost, "/rooms/"+roomID+"/messages", tokenAlice, map[string]any{"content": "to delete"}, http.StatusCreated)
	var cm createMessageAPIResponse
	require.NoError(t, json.Unmarshal(recM.Body.Bytes(), &cm))
	msgID := cm.Message.ID.String()

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	u.Scheme = "ws"
	u.Path = "/ws"
	q := u.Query()
	q.Set("access_token", tokenBob)
	u.RawQuery = q.Encode()
	bobConn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = bobConn.Close() })
	_, _, _ = bobConn.ReadMessage()
	require.NoError(t, bobConn.WriteJSON(map[string]any{"type": "subscribe", "room_id": roomID}))
	_, _, _ = bobConn.ReadMessage()

	doRequest(t, e, http.MethodDelete, "/messages/"+msgID, tokenAlice, nil, http.StatusNoContent)

	_ = bobConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, data, err := bobConn.ReadMessage()
	require.NoError(t, err)
	var del map[string]any
	require.NoError(t, json.Unmarshal(data, &del))
	require.Equal(t, controller.WSEventMessageDeleted, del["type"])
	require.Equal(t, roomID, del["room_id"])
	require.Equal(t, msgID, del["message_id"])
}

func TestWS_MessageCreatedWithImageMultipart(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(publicRoot, "uploads"), 0o755))
	e := newTestEcho(t, gormDB, publicRoot, "")

	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Alice", "email": "imgalice@e2e.test", "password": pass,
	}, http.StatusCreated)
	recB := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Bob", "email": "imgbob@e2e.test", "password": pass,
	}, http.StatusCreated)
	var bobReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &bobReg))
	tokenAlice := login(t, e, "imgalice@e2e.test", pass)
	tokenBob := login(t, e, "imgbob@e2e.test", pass)

	recG := doRequest(t, e, http.MethodPost, "/rooms/groups", tokenAlice, map[string]any{
		"name":       "ImgRoom",
		"member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var room roomIDBody
	require.NoError(t, json.Unmarshal(recG.Body.Bytes(), &room))
	roomID := room.ID.String()

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	u.Scheme = "ws"
	u.Path = "/ws"
	q := u.Query()
	q.Set("access_token", tokenBob)
	u.RawQuery = q.Encode()
	bobConn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = bobConn.Close() })
	_, _, _ = bobConn.ReadMessage()
	require.NoError(t, bobConn.WriteJSON(map[string]any{"type": "subscribe", "room_id": roomID}))
	_, _, _ = bobConn.ReadMessage()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="px.png"`)
	h.Set("Content-Type", "image/png")
	part, err := w.CreatePart(h)
	require.NoError(t, err)
	_, err = part.Write([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a})
	require.NoError(t, err)
	_ = w.WriteField("content", "photo caption")
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID+"/messages", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	_ = bobConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, data, err := bobConn.ReadMessage()
	require.NoError(t, err)
	var evt map[string]any
	require.NoError(t, json.Unmarshal(data, &evt))
	require.Equal(t, controller.WSEventMessageCreated, evt["type"])
	msgObj, ok := evt["message"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "photo caption", msgObj["content"])
	require.NotEmpty(t, msgObj["image_url"])
}

func TestWS_PresenceUpdatedWhenPeerConnectsAndAdminDeactivates(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	e := newTestEcho(t, gormDB, publicRoot, "pradm@e2e.test")

	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Admin", "email": "pradm@e2e.test", "password": pass,
	}, http.StatusCreated)
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Alice", "email": "pralice@e2e.test", "password": pass,
	}, http.StatusCreated)
	recB := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Bob", "email": "prbob@e2e.test", "password": pass,
	}, http.StatusCreated)
	var bobReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &bobReg))

	tokenAdmin := login(t, e, "pradm@e2e.test", pass)
	tokenAlice := login(t, e, "pralice@e2e.test", pass)
	tokenBob := login(t, e, "prbob@e2e.test", pass)

	recG := doRequest(t, e, http.MethodPost, "/rooms/groups", tokenAlice, map[string]any{
		"name":       "PrRoom",
		"member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var room roomIDBody
	require.NoError(t, json.Unmarshal(recG.Body.Bytes(), &room))
	roomID := room.ID.String()

	parseWS := func(tok string) *websocket.Conn {
		u, err := url.Parse(srv.URL)
		require.NoError(t, err)
		u.Scheme = "ws"
		u.Path = "/ws"
		q := u.Query()
		q.Set("access_token", tok)
		u.RawQuery = q.Encode()
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		require.NoError(t, err)
		return c
	}

	aliceConn := parseWS(tokenAlice)
	t.Cleanup(func() { _ = aliceConn.Close() })
	_, _, _ = aliceConn.ReadMessage()
	require.NoError(t, aliceConn.WriteJSON(map[string]any{"type": "subscribe", "room_id": roomID}))
	_, _, _ = aliceConn.ReadMessage()

	bobConn := parseWS(tokenBob)
	t.Cleanup(func() { _ = bobConn.Close() })
	_, _, _ = bobConn.ReadMessage()

	_ = aliceConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, data, err := aliceConn.ReadMessage()
	require.NoError(t, err)
	var prOn map[string]any
	require.NoError(t, json.Unmarshal(data, &prOn))
	require.Equal(t, controller.WSEventPresenceUpdated, prOn["type"])
	require.Equal(t, bobReg.User.ID.String(), prOn["user_id"])
	require.Equal(t, true, prOn["is_online"])

	doRequest(t, e, http.MethodPatch, "/admin/users/"+bobReg.User.ID.String()+"/active", tokenAdmin, map[string]bool{"active": false}, http.StatusNoContent)

	_ = aliceConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, data, err = aliceConn.ReadMessage()
	require.NoError(t, err)
	var prOff map[string]any
	require.NoError(t, json.Unmarshal(data, &prOff))
	require.Equal(t, controller.WSEventPresenceUpdated, prOff["type"])
	require.Equal(t, bobReg.User.ID.String(), prOff["user_id"])
	require.Equal(t, false, prOff["is_online"])
}

func TestWS_AdminForceRemoveThenResubscribeForbidden(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(publicRoot, "uploads"), 0o755))
	e := newTestEcho(t, gormDB, publicRoot, "kickws@e2e.test")

	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Admin", "email": "kickws@e2e.test", "password": pass,
	}, http.StatusCreated)
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Alice", "email": "kickalice@e2e.test", "password": pass,
	}, http.StatusCreated)
	recB := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Bob", "email": "kickbob@e2e.test", "password": pass,
	}, http.StatusCreated)
	var bobReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &bobReg))

	tokenAdmin := login(t, e, "kickws@e2e.test", pass)
	tokenAlice := login(t, e, "kickalice@e2e.test", pass)
	tokenBob := login(t, e, "kickbob@e2e.test", pass)

	recG := doRequest(t, e, http.MethodPost, "/rooms/groups", tokenAlice, map[string]any{
		"name":       "KickWS",
		"member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var room roomIDBody
	require.NoError(t, json.Unmarshal(recG.Body.Bytes(), &room))
	roomID := room.ID.String()

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	u.Scheme = "ws"
	u.Path = "/ws"
	q := u.Query()
	q.Set("access_token", tokenBob)
	u.RawQuery = q.Encode()
	bobConn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	_, _, _ = bobConn.ReadMessage()
	require.NoError(t, bobConn.WriteJSON(map[string]any{"type": "subscribe", "room_id": roomID}))
	_, _, _ = bobConn.ReadMessage()

	doRequest(t, e, http.MethodDelete, "/admin/rooms/"+roomID+"/members/"+bobReg.User.ID.String(), tokenAdmin, nil, http.StatusNoContent)

	_ = bobConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, _, err = bobConn.ReadMessage()
	require.Error(t, err)
	_ = bobConn.Close()

	u2, err := url.Parse(srv.URL)
	require.NoError(t, err)
	u2.Scheme = "ws"
	u2.Path = "/ws"
	q2 := u2.Query()
	q2.Set("access_token", tokenBob)
	u2.RawQuery = q2.Encode()
	bob2, _, err := websocket.DefaultDialer.Dial(u2.String(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = bob2.Close() })
	_, _, _ = bob2.ReadMessage()
	require.NoError(t, bob2.WriteJSON(map[string]any{"type": "subscribe", "room_id": roomID}))
	_, data, err := bob2.ReadMessage()
	require.NoError(t, err)
	var wse map[string]any
	require.NoError(t, json.Unmarshal(data, &wse))
	require.Equal(t, controller.WSEventError, wse["type"])
	require.Equal(t, controller.WSCodeForbiddenNotMember, wse["code"])
}
