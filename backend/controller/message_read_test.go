package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"chat-app/usecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMessageReadUsecaseHTTP(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{usecase.ErrInvalidUserID, http.StatusBadRequest, APICodeInvalidUserID},
		{usecase.ErrInvalidRoomID, http.StatusBadRequest, APICodeInvalidRoomID},
		{usecase.ErrInvalidReadMessageID, http.StatusBadRequest, APICodeInvalidReadMessageID},
		{usecase.ErrReadMessageRoomMismatch, http.StatusBadRequest, APICodeReadMessageRoomMismatch},
		{usecase.ErrNotRoomMember, http.StatusForbidden, APICodeNotRoomMember},
		{usecase.ErrRoomFrozen, http.StatusForbidden, APICodeRoomFrozen},
		{usecase.ErrReadMessageNotFound, http.StatusNotFound, APICodeReadMessageNotFound},
		{errors.New("db"), http.StatusInternalServerError, APICodeInternal},
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

func TestReadUpdatedPayloadJSON(t *testing.T) {
	rid := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	uid := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	mid := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	b, err := json.Marshal(readUpdatedPayload{
		Type:              WSEventReadUpdated,
		SchemaVersion:     WSEventSchemaVersion,
		RoomID:            rid,
		UserID:            uid,
		LastReadMessageID: mid,
	})
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))
	require.Equal(t, WSEventReadUpdated, out["type"])
	require.EqualValues(t, WSEventSchemaVersion, out["schema_version"])
	require.Equal(t, rid.String(), out["room_id"])
	require.Equal(t, uid.String(), out["user_id"])
	require.Equal(t, mid.String(), out["last_read_message_id"])
}
