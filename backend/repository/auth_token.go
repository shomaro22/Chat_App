package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	AuthTokenKindPasswordReset = "password_reset"
	AuthTokenKindEmailVerify   = "email_verify"
)

type AuthToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Kind      string     `gorm:"not null"`
	TokenHash string     `gorm:"column:token_hash;not null"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null"`
	CreatedAt time.Time  `gorm:"not null"`
	ExpiresAt time.Time  `gorm:"not null"`
	UsedAt    *time.Time `gorm:"column:used_at"`
}

func (AuthToken) TableName() string { return "auth_tokens" }

type AuthTokenRepository interface {
	Create(ctx context.Context, t *AuthToken) error
	FindValidByHash(ctx context.Context, kind, tokenHash string) (*AuthToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID, at time.Time) error
	DeleteExpired(ctx context.Context, before time.Time) error
}

type authTokenRepository struct{ db *gorm.DB }

func NewAuthTokenRepository(db *gorm.DB) AuthTokenRepository {
	return &authTokenRepository{db: db}
}

func (r *authTokenRepository) Create(ctx context.Context, t *AuthToken) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *authTokenRepository) FindValidByHash(ctx context.Context, kind, tokenHash string) (*AuthToken, error) {
	var row AuthToken
	err := r.db.WithContext(ctx).
		Where("kind = ? AND token_hash = ? AND used_at IS NULL AND expires_at > ?", kind, tokenHash, time.Now().UTC()).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *authTokenRepository) MarkUsed(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).Model(&AuthToken{}).Where("id = ?", id).Update("used_at", at).Error
}

func (r *authTokenRepository) DeleteExpired(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).Where("expires_at < ?", before).Delete(&AuthToken{}).Error
}
