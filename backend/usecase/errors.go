package usecase

import "errors"

var (
	ErrInvalidUserID     = errors.New("invalid user id")
	ErrInvalidCreatorID  = errors.New("invalid creator id")
	ErrSelfDirect        = errors.New("cannot open direct room with yourself")
	ErrGroupNameRequired = errors.New("group name is required")

	ErrUserNotFound        = errors.New("user not found")
	ErrTargetUserNotFound  = errors.New("target user not found")
	ErrInactiveUser        = errors.New("user is inactive")
	ErrInvalidGroupMembers = errors.New("one or more members do not exist")
	ErrGroupTooFewMembers  = errors.New("group must have at least two members including the creator")

	ErrInvalidRoomID        = errors.New("invalid room id")
	ErrInvalidMessageID     = errors.New("invalid message id")
	ErrInvalidSenderID      = errors.New("invalid sender id")
	ErrMessageNotFound      = errors.New("message not found")
	ErrNotRoomMember        = errors.New("not active room member")
	ErrEmptyMessage         = errors.New("message content or image is required")
	ErrInvalidImageURL      = errors.New("image_url must be a path from this server (/static/uploads/...)")
	ErrForbiddenMessageEdit = errors.New("forbidden message operation")

	ErrInvalidReadMessageID    = errors.New("invalid read message id")
	ErrReadMessageNotFound     = errors.New("read target message not found")
	ErrReadMessageRoomMismatch = errors.New("message does not belong to room")

	ErrRoomNotFound   = errors.New("room not found")
	ErrNotGroupRoom   = errors.New("not a group room")
	ErrAlreadyMember  = errors.New("user is already an active member")
	ErrForbiddenKick  = errors.New("forbidden to remove this member")
	ErrMemberNotFound = errors.New("member not found in room")

	ErrUnauthorized        = errors.New("unauthorized")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrRoomFrozen          = errors.New("room is frozen")
	ErrForbiddenAdmin      = errors.New("admin only")
	ErrInvalidDisplayName  = errors.New("name is required")
	ErrWeakPassword        = errors.New("password must be at least 8 characters")
	ErrEmailAlreadyInUse   = errors.New("email already in use")
	ErrForbiddenRoomUpdate = errors.New("forbidden to update this room")

	ErrUnsupportedMedia = errors.New("unsupported image type")
	ErrUploadTooLarge   = errors.New("upload too large")
)
