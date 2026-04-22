package usecase

import "github.com/google/uuid"

// UserOnlineLookup は WS 接続状況などからユーザーがオンラインかを返す（nil なら常に false 扱い）。
type UserOnlineLookup interface {
	IsUserOnline(userID uuid.UUID) bool
}
