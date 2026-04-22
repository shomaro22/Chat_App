//go:build integration

package e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"chat-app/entity"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSecurity_NoJWTOnProtectedRoutes(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	e := newTestEcho(t, gormDB, publicRoot, "")

	doRequest(t, e, http.MethodGet, "/rooms", "", nil, http.StatusUnauthorized)
	doRequest(t, e, http.MethodGet, "/me", "", nil, http.StatusUnauthorized)
}

func TestSecurity_TamperedJWTRejected(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	e := newTestEcho(t, gormDB, publicRoot, "")

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "T", "email": "tamper@e2e.test", "password": pass,
	}, http.StatusCreated)
	tok := login(t, e, "tamper@e2e.test", pass)
	require.Greater(t, len(tok), 20)
	// 署名部分を破壊（末尾を書き換え）
	bad := tok[:len(tok)-4] + "xxxx"
	doRequest(t, e, http.MethodGet, "/me", bad, nil, http.StatusUnauthorized)
}

func TestSecurity_ExpiredJWTRejected(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	e := newTestEcho(t, gormDB, publicRoot, "")

	uid := uuid.New()
	claims := jwt.MapClaims{
		"sub":  uid.String(),
		"role": string(entity.UserRoleUser),
		"exp":  time.Now().Add(-2 * time.Hour).Unix(),
		"iat":  time.Now().Add(-3 * time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(jwtSecretE2E))
	require.NoError(t, err)

	doRequest(t, e, http.MethodGet, "/me", signed, nil, http.StatusUnauthorized)
}

func TestSecurity_UserCannotAccessAdminUsers(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	e := newTestEcho(t, gormDB, publicRoot, "admsec@e2e.test")

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Admin", "email": "admsec@e2e.test", "password": pass,
	}, http.StatusCreated)
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "User", "email": "normsec@e2e.test", "password": pass,
	}, http.StatusCreated)

	userTok := login(t, e, "normsec@e2e.test", pass)
	doRequest(t, e, http.MethodGet, "/admin/users", userTok, nil, http.StatusForbidden)
}

func TestSecurity_BobCannotDeleteAliceMessage(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	e := newTestEcho(t, gormDB, publicRoot, "")

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
		"name":       "DelSec",
		"member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var room roomIDBody
	require.NoError(t, json.Unmarshal(recG.Body.Bytes(), &room))

	recM := doRequest(t, e, http.MethodPost, "/rooms/"+room.ID.String()+"/messages", tokenAlice, map[string]any{"content": "owned by alice"}, http.StatusCreated)
	var cm createMessageAPIResponse
	require.NoError(t, json.Unmarshal(recM.Body.Bytes(), &cm))

	doRequest(t, e, http.MethodDelete, "/messages/"+cm.Message.ID.String(), tokenBob, nil, http.StatusForbidden)
}

func TestSecurity_EmptyMessageRejected(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	e := newTestEcho(t, gormDB, publicRoot, "")

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "A", "email": "emptymsg@e2e.test", "password": pass,
	}, http.StatusCreated)
	recB := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "B", "email": "emptymsgbob@e2e.test", "password": pass,
	}, http.StatusCreated)
	var bobReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &bobReg))
	tok := login(t, e, "emptymsg@e2e.test", pass)

	recG := doRequest(t, e, http.MethodPost, "/rooms/groups", tok, map[string]any{
		"name": "E", "member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var room roomIDBody
	require.NoError(t, json.Unmarshal(recG.Body.Bytes(), &room))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+room.ID.String()+"/messages", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSecurity_NonMemberPostMessageForbidden(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	e := newTestEcho(t, gormDB, publicRoot, "")

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Alice", "email": "postalice@e2e.test", "password": pass,
	}, http.StatusCreated)
	recB := doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Bob", "email": "postbob@e2e.test", "password": pass,
	}, http.StatusCreated)
	var bobReg registerAPIResponse
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &bobReg))
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "Carol", "email": "postcarol@e2e.test", "password": pass,
	}, http.StatusCreated)

	tokenAlice := login(t, e, "postalice@e2e.test", pass)
	tokenCarol := login(t, e, "postcarol@e2e.test", pass)

	recG := doRequest(t, e, http.MethodPost, "/rooms/groups", tokenAlice, map[string]any{
		"name":       "PostG",
		"member_ids": []string{bobReg.User.ID.String()},
	}, http.StatusCreated)
	var room roomIDBody
	require.NoError(t, json.Unmarshal(recG.Body.Bytes(), &room))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+room.ID.String()+"/messages", strings.NewReader(`{"content":"intruder"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenCarol)
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestSecurity_InvalidRoomUUIDInPath(t *testing.T) {
	gormDB := openTruncatedDB(t)
	publicRoot := t.TempDir()
	e := newTestEcho(t, gormDB, publicRoot, "")

	const pass = "password12"
	doRequest(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"name": "A", "email": "badpath@e2e.test", "password": pass,
	}, http.StatusCreated)
	tok := login(t, e, "badpath@e2e.test", pass)

	doRequest(t, e, http.MethodGet, "/rooms/not-a-uuid/messages", tok, nil, http.StatusBadRequest)
}
