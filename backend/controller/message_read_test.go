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
		err  error
		want int
	}{
		{usecase.ErrInvalidMessageReadUserID, http.StatusBadRequest},
		{usecase.ErrInvalidMessageReadRoomID, http.StatusBadRequest},
		{usecase.ErrInvalidReadMessageID, http.StatusBadRequest},
		{usecase.ErrReadMessageRoomMismatch, http.StatusBadRequest},
		{usecase.ErrReadNotRoomMember, http.StatusForbidden},
		{usecase.ErrRoomFrozen, http.StatusForbidden},
		{usecase.ErrReadMessageNotFound, http.StatusNotFound},
		{errors.New("db"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		code, msg := messageReadUsecaseHTTP(tt.err)
		require.Equal(t, tt.want, code)
		if tt.want == http.StatusInternalServerError {
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
		Type:              "read_updated",
		RoomID:            rid,
		UserID:            uid,
		LastReadMessageID: mid,
	})
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))
	require.Equal(t, "read_updated", out["type"])
	require.Equal(t, rid.String(), out["room_id"])
	require.Equal(t, uid.String(), out["user_id"])
	require.Equal(t, mid.String(), out["last_read_message_id"])
}
