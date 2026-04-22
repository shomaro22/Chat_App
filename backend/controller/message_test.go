package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"net/http"
	"testing"
	"time"

	"chat-app/entity"
	"chat-app/middleware"
	"chat-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestMessageUsecaseHTTP(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{usecase.ErrInvalidRoomID, http.StatusBadRequest, APICodeInvalidRoomID},
		{usecase.ErrInvalidMessageID, http.StatusBadRequest, APICodeInvalidMessageID},
		{usecase.ErrInvalidSenderID, http.StatusBadRequest, APICodeInvalidSenderID},
		{usecase.ErrEmptyMessage, http.StatusBadRequest, APICodeEmptyMessage},
		{usecase.ErrInvalidUserID, http.StatusBadRequest, APICodeInvalidUserID},
		{usecase.ErrNotRoomMember, http.StatusForbidden, APICodeNotRoomMember},
		{usecase.ErrForbiddenMessageEdit, http.StatusForbidden, APICodeForbiddenMessageEdit},
		{usecase.ErrRoomFrozen, http.StatusForbidden, APICodeRoomFrozen},
		{usecase.ErrMessageNotFound, http.StatusNotFound, APICodeMessageNotFound},
		{errors.New("db exploded"), http.StatusInternalServerError, APICodeInternal},
	}
	for _, tt := range tests {
		status, code, msg := domainErrHTTP(tt.err)
		require.Equal(t, tt.wantStatus, status)
		require.Equal(t, tt.wantCode, code)
		if tt.wantStatus == http.StatusInternalServerError {
			require.Equal(t, "internal server error", msg)
		} else {
			require.Equal(t, tt.err.Error(), msg)
		}
	}
}

type messageUseCaseStub struct {
	sendMessageFn     func(ctx context.Context, roomID, senderID uuid.UUID, content, imageURL *string) (*entity.Message, error)
	sendRoomMessageFn func(ctx context.Context, in usecase.MessageSendInput) (*entity.Message, error)
	editMessageFn     func(ctx context.Context, messageID, editorID uuid.UUID, content *string) (*entity.Message, error)
	searchJoinedFn    func(ctx context.Context, userID uuid.UUID, query string, limit int) ([]ListedMessage, error)
	listMessagesFn    func(ctx context.Context, roomID, userID uuid.UUID, beforeID, afterID *uuid.UUID, limit int, orderDesc, includeReadCount bool) ([]ListedMessage, error)
	deleteMessageFn   func(ctx context.Context, messageID, userID uuid.UUID) (uuid.UUID, error)
}

func (s *messageUseCaseStub) SendMessage(ctx context.Context, roomID, senderID uuid.UUID, content, imageURL *string) (*entity.Message, error) {
	if s.sendMessageFn == nil {
		return nil, errors.New("unexpected SendMessage call")
	}
	return s.sendMessageFn(ctx, roomID, senderID, content, imageURL)
}

func (s *messageUseCaseStub) SendRoomMessage(ctx context.Context, in usecase.MessageSendInput) (*entity.Message, error) {
	if s.sendRoomMessageFn == nil {
		return nil, errors.New("unexpected SendRoomMessage call")
	}
	return s.sendRoomMessageFn(ctx, in)
}

func (s *messageUseCaseStub) EditMessage(ctx context.Context, messageID, editorID uuid.UUID, content *string) (*entity.Message, error) {
	if s.editMessageFn == nil {
		return nil, errors.New("unexpected EditMessage call")
	}
	return s.editMessageFn(ctx, messageID, editorID, content)
}

func (s *messageUseCaseStub) SearchJoined(ctx context.Context, userID uuid.UUID, query string, limit int) ([]ListedMessage, error) {
	if s.searchJoinedFn == nil {
		return nil, errors.New("unexpected SearchJoined call")
	}
	return s.searchJoinedFn(ctx, userID, query, limit)
}

func (s *messageUseCaseStub) ListMessages(ctx context.Context, roomID, userID uuid.UUID, beforeID, afterID *uuid.UUID, limit int, orderDesc, includeReadCount bool) ([]ListedMessage, error) {
	if s.listMessagesFn == nil {
		return nil, errors.New("unexpected ListMessages call")
	}
	return s.listMessagesFn(ctx, roomID, userID, beforeID, afterID, limit, orderDesc, includeReadCount)
}

func (s *messageUseCaseStub) DeleteMessage(ctx context.Context, messageID, userID uuid.UUID) (uuid.UUID, error) {
	if s.deleteMessageFn == nil {
		return uuid.Nil, errors.New("unexpected DeleteMessage call")
	}
	return s.deleteMessageFn(ctx, messageID, userID)
}

func TestMessageController_CreateMessage_JSON_BindsAndResponds(t *testing.T) {
	e := echo.New()
	roomID := uuid.New()
	userID := uuid.New()
	parentID := uuid.New()
	now := time.Now().UTC()
	body := `{"content":"hello","image_url":"/static/uploads/a.png","file_url":"/static/uploads/f.txt","file_name":"f.txt","file_content_type":"text/plain","parent_message_id":"` + parentID.String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID.String()+"/messages", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/rooms/:id/messages")
	c.SetParamNames("id")
	c.SetParamValues(roomID.String())
	c.Set(middleware.ContextUserIDKey, userID.String())

	var gotIn usecase.MessageSendInput
	h := NewMessageController(&messageUseCaseStub{
		sendRoomMessageFn: func(_ context.Context, in usecase.MessageSendInput) (*entity.Message, error) {
			gotIn = in
			return &entity.Message{
				ID:              uuid.New(),
				RoomID:          in.RoomID,
				SenderID:        in.SenderID,
				ParentMessageID: in.ParentMessageID,
				Content:         in.Content,
				ImageURL:        in.ImageURL,
				FileURL:         in.FileURL,
				FileName:        in.FileName,
				FileContentType: in.FileContentType,
				CreatedAt:       now,
				UpdatedAt:       now,
			}, nil
		},
	}, nil)

	err := h.CreateMessage(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, roomID, gotIn.RoomID)
	require.Equal(t, userID, gotIn.SenderID)
	require.NotNil(t, gotIn.ParentMessageID)
	require.Equal(t, parentID, *gotIn.ParentMessageID)
	require.NotNil(t, gotIn.Content)
	require.Equal(t, "hello", *gotIn.Content)

	var out map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	msg := out["message"].(map[string]any)
	require.Equal(t, roomID.String(), msg["room_id"])
	require.Equal(t, userID.String(), msg["sender_id"])
	require.Equal(t, "hello", msg["content"])
}

func TestMessageController_CreateMessage_MultipartWithoutMedia_Returns503(t *testing.T) {
	e := echo.New()
	roomID := uuid.New()
	userID := uuid.New()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("file", "a.png")
	require.NoError(t, err)
	_, err = fw.Write([]byte("png"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID.String()+"/messages", &b)
	req.Header.Set(echo.HeaderContentType, w.FormDataContentType())
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/rooms/:id/messages")
	c.SetParamNames("id")
	c.SetParamValues(roomID.String())
	c.Set(middleware.ContextUserIDKey, userID.String())

	h := NewMessageController(&messageUseCaseStub{}, nil)
	err = h.CreateMessage(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Contains(t, rec.Body.String(), "image upload is not configured")
}

func TestMessageController_PatchMessage_RoomMismatch_Returns400(t *testing.T) {
	e := echo.New()
	roomID := uuid.New()
	otherRoomID := uuid.New()
	messageID := uuid.New()
	userID := uuid.New()
	req := httptest.NewRequest(http.MethodPatch, "/rooms/"+roomID.String()+"/messages/"+messageID.String(), strings.NewReader(`{"content":"  updated  "}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/rooms/:id/messages/:messageId")
	c.SetParamNames("id", "messageId")
	c.SetParamValues(roomID.String(), messageID.String())
	c.Set(middleware.ContextUserIDKey, userID.String())

	var gotContent *string
	h := NewMessageController(&messageUseCaseStub{
		editMessageFn: func(_ context.Context, gotMsgID, gotUserID uuid.UUID, content *string) (*entity.Message, error) {
			require.Equal(t, messageID, gotMsgID)
			require.Equal(t, userID, gotUserID)
			gotContent = content
			return &entity.Message{ID: messageID, RoomID: otherRoomID, SenderID: userID, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}, nil
		},
	}, nil)

	err := h.PatchMessage(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.NotNil(t, gotContent)
	require.Equal(t, "updated", *gotContent)
	require.Contains(t, rec.Body.String(), "message does not belong to room")
}

func TestMessageController_ListMessages_ParsesQueryAndReturnsNormalizedLimit(t *testing.T) {
	e := echo.New()
	roomID := uuid.New()
	userID := uuid.New()
	beforeID := uuid.New()
	afterID := uuid.New()
	readCount := int64(3)
	msgID := uuid.New()
	now := time.Now().UTC()
	req := httptest.NewRequest(http.MethodGet, "/rooms/"+roomID.String()+"/messages?limit=999&before_id="+beforeID.String()+"&after_id="+afterID.String()+"&order=asc&include_read_count=true", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/rooms/:id/messages")
	c.SetParamNames("id")
	c.SetParamValues(roomID.String())
	c.Set(middleware.ContextUserIDKey, userID.String())

	h := NewMessageController(&messageUseCaseStub{
		listMessagesFn: func(_ context.Context, gotRoomID, gotUserID uuid.UUID, gotBefore, gotAfter *uuid.UUID, gotLimit int, gotOrderDesc, gotIncludeRead bool) ([]ListedMessage, error) {
			require.Equal(t, roomID, gotRoomID)
			require.Equal(t, userID, gotUserID)
			require.NotNil(t, gotBefore)
			require.Equal(t, beforeID, *gotBefore)
			require.NotNil(t, gotAfter)
			require.Equal(t, afterID, *gotAfter)
			require.Equal(t, 999, gotLimit)
			require.False(t, gotOrderDesc)
			require.True(t, gotIncludeRead)
			return []ListedMessage{{
				Message: entity.Message{
					ID:        msgID,
					RoomID:    roomID,
					SenderID:  userID,
					Content:   strPtr("hi"),
					CreatedAt: now,
					UpdatedAt: now,
				},
				SenderName: "alice",
				ReadCount:  &readCount,
			}}, nil
		},
	}, nil)

	err := h.ListMessages(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	var out map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Equal(t, "asc", out["order"])
	require.EqualValues(t, MaxMessageListLimit, out["limit"])
	require.EqualValues(t, 1, out["count"])
}

func TestMessageController_SearchMessages_TrimQueryAndValidateLimit(t *testing.T) {
	t.Run("bad limit", func(t *testing.T) {
		e := echo.New()
		userID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/search/messages?q=abc&limit=-1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(middleware.ContextUserIDKey, userID.String())
		h := NewMessageController(&messageUseCaseStub{}, nil)
		err := h.SearchMessages(c)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("trimmed query", func(t *testing.T) {
		e := echo.New()
		userID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/search/messages?q=%20%20hello%20%20&limit=2", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(middleware.ContextUserIDKey, userID.String())
		h := NewMessageController(&messageUseCaseStub{
			searchJoinedFn: func(_ context.Context, gotUserID uuid.UUID, query string, limit int) ([]ListedMessage, error) {
				require.Equal(t, userID, gotUserID)
				require.Equal(t, "hello", query)
				require.Equal(t, 2, limit)
				return nil, nil
			},
		}, nil)
		err := h.SearchMessages(c)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rec.Code)
	})
}

func strPtr(v string) *string { return &v }
