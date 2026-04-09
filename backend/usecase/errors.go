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
	ErrForbiddenMessageEdit = errors.New("forbidden message operation")

	ErrInvalidMessageReadUserID = errors.New("invalid user id")
	ErrInvalidMessageReadRoomID = errors.New("invalid room id")
	ErrInvalidReadMessageID     = errors.New("invalid read message id")
	ErrReadNotRoomMember        = errors.New("not active room member")
	ErrReadMessageNotFound      = errors.New("read target message not found")
	ErrReadMessageRoomMismatch  = errors.New("message does not belong to room")
)
