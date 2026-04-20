package controller

import (
	"errors"
	"net/http"

	"chat-app/usecase"

	"github.com/labstack/echo/v4"
)

// Stable API error codes for clients (REST). Pair with HTTP status.
const (
	APICodeInvalidUserID           = "INVALID_USER_ID"
	APICodeInvalidCreatorID        = "INVALID_CREATOR_ID"
	APICodeInvalidRoomID           = "INVALID_ROOM_ID"
	APICodeInvalidMessageID        = "INVALID_MESSAGE_ID"
	APICodeInvalidSenderID         = "INVALID_SENDER_ID"
	APICodeInvalidReadMessageID    = "INVALID_READ_MESSAGE_ID"
	APICodeSelfDirect              = "SELF_DIRECT_NOT_ALLOWED"
	APICodeGroupNameRequired       = "GROUP_NAME_REQUIRED"
	APICodeGroupTooFewMembers      = "GROUP_TOO_FEW_MEMBERS"
	APICodeInvalidGroupMembers     = "INVALID_GROUP_MEMBERS"
	APICodeNotGroupRoom            = "NOT_GROUP_ROOM"
	APICodeEmptyMessage            = "EMPTY_MESSAGE"
	APICodeInvalidImageURL         = "INVALID_IMAGE_URL"
	APICodeReadMessageRoomMismatch = "MESSAGE_ROOM_MISMATCH"
	APICodeUserNotFound            = "USER_NOT_FOUND"
	APICodeTargetUserNotFound      = "TARGET_USER_NOT_FOUND"
	APICodeRoomNotFound            = "ROOM_NOT_FOUND"
	APICodeMemberNotFound          = "MEMBER_NOT_FOUND"
	APICodeMessageNotFound         = "MESSAGE_NOT_FOUND"
	APICodeReadMessageNotFound     = "READ_MESSAGE_NOT_FOUND"
	APICodeUserInactive            = "USER_INACTIVE"
	APICodeNotRoomMember           = "NOT_ROOM_MEMBER"
	APICodeRoomFrozen              = "ROOM_FROZEN"
	APICodeForbiddenMessageEdit    = "FORBIDDEN_MESSAGE_EDIT"
	APICodeForbiddenKick           = "FORBIDDEN_KICK"
	APICodeForbiddenRoomUpdate     = "FORBIDDEN_ROOM_UPDATE"
	APICodeAlreadyMember           = "ALREADY_MEMBER"
	APICodeCannotBanAdmin          = "CANNOT_BAN_ADMIN"
	APICodeUnsupportedMedia        = "UNSUPPORTED_MEDIA"
	APICodeUploadTooLarge          = "UPLOAD_TOO_LARGE"
	APICodeInternal                = "INTERNAL_ERROR"
	APICodeForbiddenAdmin          = "FORBIDDEN_ADMIN"
	APICodeRateLimited             = "RATE_LIMITED"
)

// mapDomainError maps chat domain errors to JSON: error message + optional stable code.
func mapDomainError(c echo.Context, err error) error {
	status, code, msg := domainErrHTTP(err)
	body := map[string]string{"error": msg}
	if code != "" {
		body["code"] = code
	}
	return c.JSON(status, body)
}

func domainErrHTTP(err error) (status int, apiCode string, msg string) {
	switch {
	case errors.Is(err, usecase.ErrInvalidUserID):
		return http.StatusBadRequest, APICodeInvalidUserID, err.Error()
	case errors.Is(err, usecase.ErrInvalidCreatorID):
		return http.StatusBadRequest, APICodeInvalidCreatorID, err.Error()
	case errors.Is(err, usecase.ErrInvalidRoomID):
		return http.StatusBadRequest, APICodeInvalidRoomID, err.Error()
	case errors.Is(err, usecase.ErrInvalidMessageID):
		return http.StatusBadRequest, APICodeInvalidMessageID, err.Error()
	case errors.Is(err, usecase.ErrInvalidSenderID):
		return http.StatusBadRequest, APICodeInvalidSenderID, err.Error()
	case errors.Is(err, usecase.ErrInvalidReadMessageID):
		return http.StatusBadRequest, APICodeInvalidReadMessageID, err.Error()
	case errors.Is(err, usecase.ErrSelfDirect):
		return http.StatusBadRequest, APICodeSelfDirect, err.Error()
	case errors.Is(err, usecase.ErrGroupNameRequired):
		return http.StatusBadRequest, APICodeGroupNameRequired, err.Error()
	case errors.Is(err, usecase.ErrGroupTooFewMembers):
		return http.StatusBadRequest, APICodeGroupTooFewMembers, err.Error()
	case errors.Is(err, usecase.ErrInvalidGroupMembers):
		return http.StatusBadRequest, APICodeInvalidGroupMembers, err.Error()
	case errors.Is(err, usecase.ErrNotGroupRoom):
		return http.StatusBadRequest, APICodeNotGroupRoom, err.Error()
	case errors.Is(err, usecase.ErrEmptyMessage):
		return http.StatusBadRequest, APICodeEmptyMessage, err.Error()
	case errors.Is(err, usecase.ErrInvalidImageURL):
		return http.StatusBadRequest, APICodeInvalidImageURL, err.Error()
	case errors.Is(err, usecase.ErrReadMessageRoomMismatch):
		return http.StatusBadRequest, APICodeReadMessageRoomMismatch, err.Error()

	case errors.Is(err, usecase.ErrUserNotFound):
		return http.StatusNotFound, APICodeUserNotFound, err.Error()
	case errors.Is(err, usecase.ErrTargetUserNotFound):
		return http.StatusNotFound, APICodeTargetUserNotFound, err.Error()
	case errors.Is(err, usecase.ErrRoomNotFound):
		return http.StatusNotFound, APICodeRoomNotFound, err.Error()
	case errors.Is(err, usecase.ErrMemberNotFound):
		return http.StatusNotFound, APICodeMemberNotFound, err.Error()
	case errors.Is(err, usecase.ErrMessageNotFound):
		return http.StatusNotFound, APICodeMessageNotFound, err.Error()
	case errors.Is(err, usecase.ErrReadMessageNotFound):
		return http.StatusNotFound, APICodeReadMessageNotFound, err.Error()

	case errors.Is(err, usecase.ErrForbiddenAdmin):
		return http.StatusForbidden, APICodeForbiddenAdmin, err.Error()

	case errors.Is(err, usecase.ErrInactiveUser):
		return http.StatusForbidden, APICodeUserInactive, err.Error()
	case errors.Is(err, usecase.ErrNotRoomMember):
		return http.StatusForbidden, APICodeNotRoomMember, err.Error()
	case errors.Is(err, usecase.ErrRoomFrozen):
		return http.StatusForbidden, APICodeRoomFrozen, err.Error()
	case errors.Is(err, usecase.ErrForbiddenMessageEdit):
		return http.StatusForbidden, APICodeForbiddenMessageEdit, err.Error()
	case errors.Is(err, usecase.ErrForbiddenKick):
		return http.StatusForbidden, APICodeForbiddenKick, err.Error()
	case errors.Is(err, usecase.ErrForbiddenRoomUpdate):
		return http.StatusForbidden, APICodeForbiddenRoomUpdate, err.Error()
	case errors.Is(err, usecase.ErrCannotBanAdmin):
		return http.StatusBadRequest, APICodeCannotBanAdmin, err.Error()

	case errors.Is(err, usecase.ErrAlreadyMember):
		return http.StatusConflict, APICodeAlreadyMember, err.Error()

	case errors.Is(err, usecase.ErrUnsupportedMedia):
		return http.StatusUnsupportedMediaType, APICodeUnsupportedMedia, err.Error()
	case errors.Is(err, usecase.ErrUploadTooLarge):
		return http.StatusRequestEntityTooLarge, APICodeUploadTooLarge, err.Error()

	case errors.Is(err, usecase.ErrInvalidSearchQuery):
		return http.StatusBadRequest, "INVALID_SEARCH_QUERY", err.Error()
	case errors.Is(err, usecase.ErrInvalidEmoji):
		return http.StatusBadRequest, "INVALID_EMOJI", err.Error()
	case errors.Is(err, usecase.ErrInvalidToken):
		return http.StatusBadRequest, "INVALID_TOKEN", err.Error()
	case errors.Is(err, usecase.ErrWrongPassword):
		return http.StatusUnauthorized, "WRONG_PASSWORD", err.Error()
	case errors.Is(err, usecase.ErrEmailNotVerified):
		return http.StatusForbidden, "EMAIL_NOT_VERIFIED", err.Error()

	default:
		return http.StatusInternalServerError, APICodeInternal, "internal server error"
	}
}
