package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"chat-app/auth"
	"chat-app/entity"
	"chat-app/repository"

	"github.com/google/uuid"
)

// AuthUseCase はログイン・登録・トークン発行。JWT / パスワードは auth パッケージに委譲する。
type AuthUseCase struct {
	Users  repository.UserRepository
	Tokens *auth.TokenService
}

func NewAuthUseCase(users repository.UserRepository, tokens *auth.TokenService) *AuthUseCase {
	return &AuthUseCase{Users: users, Tokens: tokens}
}

// Login はメールとパスワードで JWT を発行する。
func (a *AuthUseCase) Login(ctx context.Context, email, password string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return "", ErrInvalidCredentials
	}
	u, err := a.Users.FindByEmail(ctx, email)
	if err != nil {
		return "", err
	}
	if u == nil || !u.Active {
		return "", ErrInvalidCredentials
	}
	if err := auth.ComparePassword(u.PasswordHash, password); err != nil {
		return "", ErrInvalidCredentials
	}
	token, err := a.Tokens.Issue(u.ID, u.Role)
	if err != nil {
		return "", err
	}
	return token, nil
}

// Register はユーザーを作成する（最初の管理者は環境変数 BOOTSTRAP_ADMIN_EMAIL と一致する場合のみ admin）。
func (a *AuthUseCase) Register(ctx context.Context, name, email, password, bootstrapAdminEmail string) (*entity.User, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(strings.ToLower(email))
	if name == "" {
		return nil, ErrInvalidDisplayName
	}
	if email == "" {
		return nil, ErrInvalidCredentials
	}
	if len(password) < 8 {
		return nil, ErrWeakPassword
	}
	existing, err := a.Users.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailAlreadyInUse
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	role := entity.UserRoleUser
	if bootstrapAdminEmail != "" && email == strings.TrimSpace(strings.ToLower(bootstrapAdminEmail)) {
		role = entity.UserRoleAdmin
	}
	now := time.Now().UTC()
	u := &entity.User{
		ID:           uuid.New(),
		Name:         name,
		Email:        email,
		PasswordHash: hash,
		Role:         role,
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := a.Users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// CreateTokenForUser は登録直後などに JWT を発行する。
func (a *AuthUseCase) CreateTokenForUser(ctx context.Context, userID uuid.UUID) (string, error) {
	u, err := a.Users.FindByID(ctx, userID)
	if err != nil {
		return "", err
	}
	if u == nil || !u.Active {
		return "", ErrUserNotFound
	}
	return a.Tokens.Issue(u.ID, u.Role)
}

// ParseToken は Authorization / WS 用トークン文字列からユーザー ID とロールを取り出す。
func (a *AuthUseCase) ParseToken(raw string) (uuid.UUID, entity.UserRole, error) {
	uid, role, err := a.Tokens.Parse(raw)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) {
			return uuid.Nil, "", ErrUnauthorized
		}
		return uuid.Nil, "", err
	}
	return uid, role, nil
}
