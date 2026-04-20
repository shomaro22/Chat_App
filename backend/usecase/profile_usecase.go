package usecase

import (
	"context"
	"strings"
	"time"

	"chat-app/entity"
	"chat-app/repository"

	"github.com/google/uuid"
)

type ProfileUseCase struct {
	Users repository.UserRepository
}

func NewProfileUseCase(users repository.UserRepository) *ProfileUseCase {
	return &ProfileUseCase{Users: users}
}

type UpdateProfileInput struct {
	Name *string
}

func (p *ProfileUseCase) UpdateProfile(ctx context.Context, userID uuid.UUID, in UpdateProfileInput) (*entity.User, error) {
	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	u, err := p.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil || !u.Active {
		return nil, ErrUserNotFound
	}
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if n == "" {
			return nil, ErrInvalidDisplayName
		}
		u.Name = n
	}
	u.UpdatedAt = time.Now().UTC()
	if err := p.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}
