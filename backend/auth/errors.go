package auth

import "errors"

var (
	// ErrInvalidToken は JWT が空・不正・期限切れ等のときに返す。
	ErrInvalidToken = errors.New("invalid token")
	// ErrNotActive はユーザーが存在しないか無効化されているときに返す。
	ErrNotActive = errors.New("account inactive or not found")
)
