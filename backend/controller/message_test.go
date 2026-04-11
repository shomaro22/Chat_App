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
