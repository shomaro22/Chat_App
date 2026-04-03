package repository

import (
	"context"
	"errors"
	"time"

	"chat-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, u *entity.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]entity.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (users *userRepository) Create(ctx context.Context, u *entity.User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	now := time.Now().UTC()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}
	m := userToModel(u)
	return users.db.WithContext(ctx).Create(m).Error
}

func (users *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	var m UserModel
	if err := users.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return userToEntity(&m), nil
}

func (users *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var m UserModel
	if err := users.db.WithContext(ctx).Where("email = ?", email).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return userToEntity(&m), nil
}

func (users *userRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]entity.User, error) {
	uniq := uniqueUserIDs(ids)
	if len(uniq) == 0 {
		return nil, nil
	}
	var models []UserModel
	if err := users.db.WithContext(ctx).Where("id IN ?", uniq).Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]entity.User, 0, len(models))
	for i := range models {
		if e := userToEntity(&models[i]); e != nil {
			out = append(out, *e)
		}
	}
	return out, nil
}

func uniqueUserIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{})
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
