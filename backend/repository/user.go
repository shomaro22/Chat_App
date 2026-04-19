package repository

import (
	"context"
	"errors"
	"strings"

	"chat-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, u *entity.User) error
	Update(ctx context.Context, u *entity.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]entity.User, error)
	SetActive(ctx context.Context, id uuid.UUID, active bool) error
	// ListForAdmin は管理者向けユーザー一覧（メール・名前・UUID の部分一致、作成日降順）。
	ListForAdmin(ctx context.Context, search string, limit, offset int) ([]entity.User, int64, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (users *userRepository) Create(ctx context.Context, u *entity.User) error {
	return users.db.WithContext(ctx).Create(u).Error
}

func (users *userRepository) Update(ctx context.Context, u *entity.User) error {
	return users.db.WithContext(ctx).Save(u).Error
}

func (users *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	var m entity.User
	if err := users.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (users *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var m entity.User
	if err := users.db.WithContext(ctx).Where("email = ?", email).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (users *userRepository) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	return users.db.WithContext(ctx).Model(&entity.User{}).Where("id = ?", id).Update("active", active).Error
}

func (users *userRepository) ListForAdmin(ctx context.Context, search string, limit, offset int) ([]entity.User, int64, error) {
	q := users.db.WithContext(ctx).Model(&entity.User{})
	if s := strings.TrimSpace(search); s != "" {
		pattern := "%" + s + "%"
		q = q.Where("CAST(id AS TEXT) ILIKE ? OR email ILIKE ? OR name ILIKE ?", pattern, pattern, pattern)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	var rows []entity.User
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (users *userRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]entity.User, error) {
	uniq := uniqueUserIDs(ids)
	if len(uniq) == 0 {
		return nil, nil
	}
	var usersRows []entity.User
	if err := users.db.WithContext(ctx).Where("id IN ?", uniq).Find(&usersRows).Error; err != nil {
		return nil, err
	}
	return usersRows, nil
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
