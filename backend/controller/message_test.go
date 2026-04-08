package controller

import (
	"errors"
	"net/http"
	"testing"

	"chat-app/usecase"

	"github.com/stretchr/testify/require"
)

func TestMessageUsecaseHTTP(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{usecase.ErrInvalidRoomID, http.StatusBadRequest},
		{usecase.ErrInvalidMessageID, http.StatusBadRequest},
		{usecase.ErrInvalidSenderID, http.StatusBadRequest},
		{usecase.ErrEmptyMessage, http.StatusBadRequest},
		{usecase.ErrInvalidUserID, http.StatusBadRequest},
		{usecase.ErrNotRoomMember, http.StatusForbidden},
		{usecase.ErrForbiddenMessageEdit, http.StatusForbidden},
		{usecase.ErrMessageNotFound, http.StatusNotFound},
		{errors.New("db exploded"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		code, msg := messageUsecaseHTTP(tt.err)
		require.Equal(t, tt.want, code)
		if tt.want == http.StatusInternalServerError {
			require.Equal(t, "internal server error", msg)
		} else {
			require.Equal(t, tt.err.Error(), msg)
		}
	}
}
