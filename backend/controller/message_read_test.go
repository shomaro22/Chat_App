package controller

import (
	"errors"
	"net/http"
	"testing"

	"chat-app/usecase"

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
