//go:build integration

package e2e_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"chat-app/auth"
	"chat-app/controller"
	"chat-app/db"
	"chat-app/entity"
	"chat-app/middleware"
	"chat-app/repository"
	"chat-app/router"
	"chat-app/usecase"
	"chat-app/wshub"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const jwtSecretE2E = "e2e-test-jwt-secret-32-bytes-min!!"

func testDatabaseURL(t *testing.T) string {
	t.Helper()
	u := os.Getenv("DATABASE_URL")
	if u == "" {
		u = "postgres://postgres:postgres@localhost:5432/chatapp?sslmode=disable"
	}
	return u
}

func openTruncatedDB(t *testing.T) *gorm.DB {
	t.Helper()
	databaseURL := testDatabaseURL(t)
	require.NoError(t, db.Up(databaseURL))
	gormDB, err := db.Open(databaseURL)
	require.NoError(t, err)
	sqlDB, err := gormDB.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec(`TRUNCATE message_reads, messages, room_members, rooms, users RESTART IDENTITY CASCADE`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	return gormDB
}

// wsHubAdapter と hubOnlineLookup は main と同じ（テスト用に最小複製）。
type wsHubAdapter struct{ *wshub.ConnectionManager }

func (a wsHubAdapter) Register(userID uuid.UUID, conn *websocket.Conn) (uuid.UUID, controller.HubPresenceChange) {
	id, ch := a.ConnectionManager.Register(userID, conn)
	return id, controller.HubPresenceChange{Changed: ch.Changed, UserID: ch.UserID, IsOnline: ch.IsOnline}
}

func (a wsHubAdapter) RemoveConn(connID uuid.UUID) controller.HubPresenceChange {
	ch := a.ConnectionManager.RemoveConn(connID)
	return controller.HubPresenceChange{Changed: ch.Changed, UserID: ch.UserID, IsOnline: ch.IsOnline}
}

func (a wsHubAdapter) CloseUserConnections(userID uuid.UUID) controller.HubPresenceChange {
	ch := a.ConnectionManager.CloseUserConnections(userID)
	return controller.HubPresenceChange{Changed: ch.Changed, UserID: ch.UserID, IsOnline: ch.IsOnline}
}

type hubOnlineLookup struct{ *wshub.ConnectionManager }

func (h hubOnlineLookup) IsUserOnline(userID uuid.UUID) bool {
	return h.ConnectionManager.IsUserOnline(userID)
}

func newTestEcho(t *testing.T, gormDB *gorm.DB, publicRoot, bootstrapAdminEmail string) *echo.Echo {
	t.Helper()
	hub := wshub.NewConnectionManager()
	users := repository.NewUserRepository(gormDB)
	rooms := repository.NewRoomRepository(gormDB)
	members := repository.NewRoomMemberRepository(gormDB)
	roomTx := repository.NewRoomUnitOfWork(gormDB)
	msgRepo := repository.NewMessageRepository(gormDB)
	readRepo := repository.NewMessageReadRepository(gormDB)

	tokenSvc := auth.NewTokenService([]byte(jwtSecretE2E), auth.DefaultTokenTTL)
	authUC := usecase.NewAuthUseCase(users, tokenSvc)
	roomUC := usecase.NewRoomUseCase(roomTx, users, rooms, members, msgRepo, readRepo, hubOnlineLookup{hub})
	mediaUC := usecase.NewMediaUseCase(publicRoot)
	msgUC := usecase.NewMessageUseCase(msgRepo, users, rooms, mediaUC)
	readUC := usecase.NewMessageReadUseCase(msgRepo, readRepo, rooms, users)
	presenceUC := usecase.NewPresenceUseCase(rooms)
	adminUC := usecase.NewAdminUseCase(users, rooms, members)

	validateSession := func(c echo.Context, uid uuid.UUID, _ entity.UserRole) error {
		u, err := auth.RequireActiveUser(c.Request().Context(), users, uid)
		if err != nil {
			if errors.Is(err, auth.ErrNotActive) {
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		c.Set(middleware.ContextUserRoleKey, string(u.Role))
		return nil
	}

	return router.New(router.Config{
		Auth:            controller.NewAuthController(authUC, bootstrapAdminEmail),
		Room:            controller.NewRoomController(controller.NewRoomUseCaseAdapter(roomUC)),
		Messages:        controller.NewMessageController(controller.NewMessageUseCaseAdapter(msgUC), hub, mediaUC),
		Reads:           controller.NewMessageReadController(controller.NewMessageReadUseCaseAdapter(readUC), hub),
		WS:              controller.NewWSController(wsHubAdapter{hub}, controller.NewRoomUseCaseAdapter(roomUC), presenceUC),
		Media:           controller.NewMediaController(mediaUC),
		Admin:           controller.NewAdminController(adminUC, wsHubAdapter{hub}, hub, presenceUC),
		ParseJWT:        authUC.ParseToken,
		ValidateSession: validateSession,
		StaticDir:       publicRoot,
	})
}

func doRequest(t *testing.T, e *echo.Echo, method, path, bearer string, body any, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != nil {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, wantStatus, rec.Code, "body=%s", rec.Body.String())
	return rec
}

type apiErrBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

type regUserBody struct {
	ID uuid.UUID `json:"id"`
}

type registerAPIResponse struct {
	User        regUserBody `json:"user"`
	AccessToken string      `json:"access_token"`
}

type loginAPIResponse struct {
	AccessToken string `json:"access_token"`
}

type roomIDBody struct {
	ID uuid.UUID `json:"id"`
}

type createMessageAPIResponse struct {
	Message struct {
		ID       uuid.UUID `json:"id"`
		RoomID   uuid.UUID `json:"room_id"`
		Content  *string   `json:"content"`
		ImageURL *string   `json:"image_url"`
	} `json:"message"`
}

type listMessagesAPIResponse struct {
	RoomID   uuid.UUID `json:"room_id"`
	Messages []struct {
		ID        uuid.UUID `json:"id"`
		Content   *string   `json:"content"`
		IsDeleted bool      `json:"is_deleted"`
		ImageURL  *string   `json:"image_url"`
	} `json:"messages"`
}

type roomReadStateAPIResponse struct {
	RoomID              uuid.UUID  `json:"room_id"`
	MyLastReadMessageID *uuid.UUID `json:"my_last_read_message_id"`
	UnreadCount         int64      `json:"unread_count"`
}

func TestHTTP_LoginRoomsDetailMessagesReadDelete(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(publicRoot, "uploads"), 0o755))
	e := newTestEcho(t, gormDB, publicRoot, "")

	const pass = "password12"
	recA := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Alice", "email": "alice@e2e.test", "password": pass,
	}, http.StatusCreated)
	var aliceReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recA.Body.Bytes(), &aliceReg))

	recB := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Bob", "email": "bob@e2e.test", "password": pass,
	}, http.StatusCreated)
	var bobReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &bobReg))

	tokenAlice := login(t, e, "alice@e2e.test", pass)
	tokenBob := login(t, e, "bob@e2e.test", pass)

	recG := doRequest(t, e, http.MethodPost, "/rooms/groups", tokenAlice, map[string]any{
		"name":       "E2E Group",
		"member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var createdRoom roomIDBody
	require.NoError(t, json.Unmarshal(recG.Body.Bytes(), &createdRoom))
	roomID := createdRoom.ID

	recList := doRequest(t, e, http.MethodGet, "/rooms", tokenAlice, nil, http.StatusOK)
	require.Contains(t, recList.Body.String(), roomID.String())

	recDet := doRequest(t, e, http.MethodGet, "/rooms/"+roomID.String(), tokenAlice, nil, http.StatusOK)
	require.Contains(t, recDet.Body.String(), `"members"`)

	recMsg0 := doRequest(t, e, http.MethodGet, "/rooms/"+roomID.String()+"/messages", tokenAlice, nil, http.StatusOK)
	var list0 listMessagesAPIResponse
	require.NoError(t, json.Unmarshal(recMsg0.Body.Bytes(), &list0))
	require.Empty(t, list0.Messages)

	txt := "hello e2e"
	recPost := doRequest(t, e, http.MethodPost, "/rooms/"+roomID.String()+"/messages", tokenAlice, map[string]any{
		"content": txt,
	}, http.StatusCreated)
	var createdMsg createMessageAPIResponse
	require.NoError(t, json.Unmarshal(recPost.Body.Bytes(), &createdMsg))
	require.Equal(t, txt, *createdMsg.Message.Content)
	msgID := createdMsg.Message.ID

	recMsg1 := doRequest(t, e, http.MethodGet, "/rooms/"+roomID.String()+"/messages", tokenAlice, nil, http.StatusOK)
	var list1 listMessagesAPIResponse
	require.NoError(t, json.Unmarshal(recMsg1.Body.Bytes(), &list1))
	require.Len(t, list1.Messages, 1)
	require.Equal(t, msgID, list1.Messages[0].ID)

	recRS := doRequest(t, e, http.MethodGet, "/rooms/"+roomID.String()+"/read-state", tokenBob, nil, http.StatusOK)
	var rs0 roomReadStateAPIResponse
	require.NoError(t, json.Unmarshal(recRS.Body.Bytes(), &rs0))
	require.Equal(t, roomID, rs0.RoomID)

	doRequest(t, e, http.MethodPost, "/rooms/"+roomID.String()+"/read", tokenBob, map[string]string{
		"message_id": msgID.String(),
	}, http.StatusNoContent)

	recRS2 := doRequest(t, e, http.MethodGet, "/rooms/"+roomID.String()+"/read-state", tokenBob, nil, http.StatusOK)
	var rs1 roomReadStateAPIResponse
	require.NoError(t, json.Unmarshal(recRS2.Body.Bytes(), &rs1))
	require.NotNil(t, rs1.MyLastReadMessageID)
	require.Equal(t, msgID, *rs1.MyLastReadMessageID)

	doRequest(t, e, http.MethodDelete, "/messages/"+msgID.String(), tokenAlice, nil, http.StatusNoContent)

	recMsg2 := doRequest(t, e, http.MethodGet, "/rooms/"+roomID.String()+"/messages", tokenAlice, nil, http.StatusOK)
	var list2 listMessagesAPIResponse
	require.NoError(t, json.Unmarshal(recMsg2.Body.Bytes(), &list2))
	require.Len(t, list2.Messages, 1)
	require.True(t, list2.Messages[0].IsDeleted)
}

func login(t *testing.T, e *echo.Echo, email, password string) string {
	t.Helper()
	rec := doRequest(t, e, http.MethodPost, "/auth/login", "", map[string]string{
		"email": email, "password": password,
	}, http.StatusOK)
	var lr loginAPIResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &lr))
	require.NotEmpty(t, lr.AccessToken)
	return lr.AccessToken
}

func TestHTTP_NonMemberRoomForbidden(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(publicRoot, "uploads"), 0o755))
	e := newTestEcho(t, gormDB, publicRoot, "")

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Alice", "email": "a@e2e.test", "password": pass,
	}, http.StatusCreated)
	recB := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Bob", "email": "b@e2e.test", "password": pass,
	}, http.StatusCreated)
	var bobReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &bobReg))
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Carol", "email": "c@e2e.test", "password": pass,
	}, http.StatusCreated)

	tokenA := login(t, e, "a@e2e.test", pass)
	tokenCarol := login(t, e, "c@e2e.test", pass)

	recG := doRequest(t, e, http.MethodPost, "/rooms/groups", tokenA, map[string]any{
		"name":       "G",
		"member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var room roomIDBody
	require.NoError(t, json.Unmarshal(recG.Body.Bytes(), &room))

	for _, p := range []string{
		"/rooms/" + room.ID.String(),
		"/rooms/" + room.ID.String() + "/messages",
		"/rooms/" + room.ID.String() + "/read-state",
	} {
		rec := doRequest(t, e, http.MethodGet, p, tokenCarol, nil, http.StatusForbidden)
		var eb apiErrBody
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &eb))
		require.Equal(t, "NOT_ROOM_MEMBER", eb.Code)
	}

	recPost := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+room.ID.String()+"/read", bytes.NewReader([]byte(`{"message_id":"`+uuid.New().String()+`"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+tokenCarol)
	e.ServeHTTP(recPost, req)
	require.Equal(t, http.StatusForbidden, recPost.Code)
	var eb apiErrBody
	require.NoError(t, json.Unmarshal(recPost.Body.Bytes(), &eb))
	require.Equal(t, "NOT_ROOM_MEMBER", eb.Code)
}

func TestHTTP_AdminFrozenBlocksSend(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(publicRoot, "uploads"), 0o755))
	e := newTestEcho(t, gormDB, publicRoot, "admin@e2e.test")

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Admin", "email": "admin@e2e.test", "password": pass,
	}, http.StatusCreated)
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Alice", "email": "alice@e2e.test", "password": pass,
	}, http.StatusCreated)
	recB := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Bob", "email": "bob@e2e.test", "password": pass,
	}, http.StatusCreated)
	var bobReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &bobReg))

	tokenAdmin := login(t, e, "admin@e2e.test", pass)
	tokenAlice := login(t, e, "alice@e2e.test", pass)

	recG := doRequest(t, e, http.MethodPost, "/rooms/groups", tokenAlice, map[string]any{
		"name":       "FrozenRoom",
		"member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var room roomIDBody
	require.NoError(t, json.Unmarshal(recG.Body.Bytes(), &room))

	doRequest(t, e, http.MethodPatch, "/admin/rooms/"+room.ID.String()+"/frozen", tokenAdmin, map[string]bool{
		"frozen": true,
	}, http.StatusNoContent)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+room.ID.String()+"/messages", bytes.NewReader([]byte(`{"content":"nope"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
	var eb apiErrBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &eb))
	require.Equal(t, "ROOM_FROZEN", eb.Code)
}

func TestHTTP_FrozenRoomListMessagesStillOK(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(publicRoot, "uploads"), 0o755))
	e := newTestEcho(t, gormDB, publicRoot, "frlist@e2e.test")

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Admin", "email": "frlist@e2e.test", "password": pass,
	}, http.StatusCreated)
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Alice", "email": "frlistalice@e2e.test", "password": pass,
	}, http.StatusCreated)
	recB := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Bob", "email": "frlistbob@e2e.test", "password": pass,
	}, http.StatusCreated)
	var bobReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &bobReg))

	tokenAdmin := login(t, e, "frlist@e2e.test", pass)
	tokenAlice := login(t, e, "frlistalice@e2e.test", pass)

	recG := doRequest(t, e, http.MethodPost, "/rooms/groups", tokenAlice, map[string]any{
		"name":       "FrList",
		"member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var room roomIDBody
	require.NoError(t, json.Unmarshal(recG.Body.Bytes(), &room))
	doRequest(t, e, http.MethodPost, "/rooms/"+room.ID.String()+"/messages", tokenAlice, map[string]any{"content": "before freeze"}, http.StatusCreated)

	doRequest(t, e, http.MethodPatch, "/admin/rooms/"+room.ID.String()+"/frozen", tokenAdmin, map[string]bool{"frozen": true}, http.StatusNoContent)

	doRequest(t, e, http.MethodGet, "/rooms/"+room.ID.String()+"/messages", tokenAlice, nil, http.StatusOK)
}

func TestHTTP_FrozenRoomMarkReadForbidden(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(publicRoot, "uploads"), 0o755))
	e := newTestEcho(t, gormDB, publicRoot, "frread@e2e.test")

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Admin", "email": "frread@e2e.test", "password": pass,
	}, http.StatusCreated)
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Alice", "email": "frreadalice@e2e.test", "password": pass,
	}, http.StatusCreated)
	recB := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Bob", "email": "frreadbob@e2e.test", "password": pass,
	}, http.StatusCreated)
	var bobReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &bobReg))

	tokenAdmin := login(t, e, "frread@e2e.test", pass)
	tokenAlice := login(t, e, "frreadalice@e2e.test", pass)

	recG := doRequest(t, e, http.MethodPost, "/rooms/groups", tokenAlice, map[string]any{
		"name":       "FrRead",
		"member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var room roomIDBody
	require.NoError(t, json.Unmarshal(recG.Body.Bytes(), &room))
	recM := doRequest(t, e, http.MethodPost, "/rooms/"+room.ID.String()+"/messages", tokenAlice, map[string]any{"content": "x"}, http.StatusCreated)
	var cm createMessageAPIResponse
	require.NoError(t, json.Unmarshal(recM.Body.Bytes(), &cm))

	doRequest(t, e, http.MethodPatch, "/admin/rooms/"+room.ID.String()+"/frozen", tokenAdmin, map[string]bool{"frozen": true}, http.StatusNoContent)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+room.ID.String()+"/read", strings.NewReader(`{"message_id":"`+cm.Message.ID.String()+`"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
	var eb apiErrBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &eb))
	require.Equal(t, "ROOM_FROZEN", eb.Code)
}

func TestHTTP_AdminForceRemoveBlocksAccess(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(publicRoot, "uploads"), 0o755))
	e := newTestEcho(t, gormDB, publicRoot, "admin@e2e.test")

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Admin", "email": "admin@e2e.test", "password": pass,
	}, http.StatusCreated)
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Alice", "email": "alice@e2e.test", "password": pass,
	}, http.StatusCreated)
	recB := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Bob", "email": "bob@e2e.test", "password": pass,
	}, http.StatusCreated)
	var bobReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &bobReg))

	tokenAdmin := login(t, e, "admin@e2e.test", pass)
	tokenAlice := login(t, e, "alice@e2e.test", pass)
	tokenBob := login(t, e, "bob@e2e.test", pass)

	recG := doRequest(t, e, http.MethodPost, "/rooms/groups", tokenAlice, map[string]any{
		"name":       "KickRoom",
		"member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var room roomIDBody
	require.NoError(t, json.Unmarshal(recG.Body.Bytes(), &room))

	doRequest(t, e, http.MethodDelete, "/admin/rooms/"+room.ID.String()+"/members/"+bobReg.User.ID.String(), tokenAdmin, nil, http.StatusNoContent)

	rec := doRequest(t, e, http.MethodGet, "/rooms/"+room.ID.String(), tokenBob, nil, http.StatusForbidden)
	var eb apiErrBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &eb))
	require.Equal(t, "NOT_ROOM_MEMBER", eb.Code)
}

func TestHTTP_MediaUploadThenMessageWithImage(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(publicRoot, "uploads"), 0o755))
	e := newTestEcho(t, gormDB, publicRoot, "")

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Alice", "email": "alice@e2e.test", "password": pass,
	}, http.StatusCreated)
	recB := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Bob", "email": "bob@e2e.test", "password": pass,
	}, http.StatusCreated)
	var bobReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &bobReg))

	tokenAlice := login(t, e, "alice@e2e.test", pass)

	recG := doRequest(t, e, http.MethodPost, "/rooms/groups", tokenAlice, map[string]any{
		"name":       "Media",
		"member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var room roomIDBody
	require.NoError(t, json.Unmarshal(recG.Body.Bytes(), &room))

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="pixel.png"`)
	h.Set("Content-Type", "image/png")
	part, err := w.CreatePart(h)
	require.NoError(t, err)
	_, err = part.Write([]byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	})
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, "/media/images", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tokenAlice)
	recUp := httptest.NewRecorder()
	e.ServeHTTP(recUp, req)
	require.Equal(t, http.StatusCreated, recUp.Code, recUp.Body.String())
	var up struct {
		URL string `json:"url"`
	}
	require.NoError(t, json.Unmarshal(recUp.Body.Bytes(), &up))
	require.NotEmpty(t, up.URL)

	recPost := doRequest(t, e, http.MethodPost, "/rooms/"+room.ID.String()+"/messages", tokenAlice, map[string]any{
		"image_url": up.URL,
	}, http.StatusCreated)
	var cm createMessageAPIResponse
	require.NoError(t, json.Unmarshal(recPost.Body.Bytes(), &cm))
	require.NotNil(t, cm.Message.ImageURL)
	require.Equal(t, up.URL, *cm.Message.ImageURL)

	recList := doRequest(t, e, http.MethodGet, "/rooms/"+room.ID.String()+"/messages", tokenAlice, nil, http.StatusOK)
	var list listMessagesAPIResponse
	require.NoError(t, json.Unmarshal(recList.Body.Bytes(), &list))
	require.Len(t, list.Messages, 1)
	require.NotNil(t, list.Messages[0].ImageURL)

	// キャプション付き画像（content + image_url 併用）
	recCap := doRequest(t, e, http.MethodPost, "/rooms/"+room.ID.String()+"/messages", tokenAlice, map[string]any{
		"content":   "caption",
		"image_url": up.URL,
	}, http.StatusCreated)
	var capMsg createMessageAPIResponse
	require.NoError(t, json.Unmarshal(recCap.Body.Bytes(), &capMsg))
	require.NotNil(t, capMsg.Message.Content)
	require.Equal(t, "caption", *capMsg.Message.Content)
	require.NotNil(t, capMsg.Message.ImageURL)
	require.Equal(t, up.URL, *capMsg.Message.ImageURL)
}

func TestHTTP_InvalidImageURLRejected(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	e := newTestEcho(t, gormDB, publicRoot, "")

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Alice", "email": "badimg@e2e.test", "password": pass,
	}, http.StatusCreated)
	recB := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Bob", "email": "badimgbob@e2e.test", "password": pass,
	}, http.StatusCreated)
	var bobReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &bobReg))
	token := login(t, e, "badimg@e2e.test", pass)

	recG := doRequest(t, e, http.MethodPost, "/rooms/groups", token, map[string]any{
		"name": "R", "member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var room roomIDBody
	require.NoError(t, json.Unmarshal(recG.Body.Bytes(), &room))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+room.ID.String()+"/messages", strings.NewReader(`{"image_url":"https://evil.example/x.png"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	var eb apiErrBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &eb))
	require.Equal(t, "INVALID_IMAGE_URL", eb.Code)
}
