package auth

import (
	"context"

	"chat-app/entity"

	"github.com/google/uuid"
)

// ActiveUserFinder はセッション検証に必要なユーザー取得の抽象（リポジトリを直接参照しない）。
type ActiveUserFinder interface {
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
}

// RequireActiveUser はユーザーが存在し active なとき *entity.User を返す（JWT 後の DB 整合用）。
func RequireActiveUser(ctx context.Context, users ActiveUserFinder, userID uuid.UUID) (*entity.User, error) {
	u, err := users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil || !u.Active {
		return nil, ErrNotActive
	}
	return u, nil
}
