package usecase

import (
	"context"
	"strings"
	"time"

	"chat-app/entity"
	"chat-app/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const jwtTTL = 72 * time.Hour

// AuthUseCase はログイン・登録・JWT 検証を担う。
type AuthUseCase struct {
	Users  repository.UserRepository
	Secret []byte
}

func NewAuthUseCase(users repository.UserRepository, secret []byte) *AuthUseCase {
	return &AuthUseCase{Users: users, Secret: secret}
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
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}
	return a.issueToken(u.ID, u.Role)
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
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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
		PasswordHash: string(hash),
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
	return a.issueToken(u.ID, u.Role)
}

func (a *AuthUseCase) issueToken(userID uuid.UUID, role entity.UserRole) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID.String(),
		"role": string(role),
		"exp":  time.Now().Add(jwtTTL).Unix(),
		"iat":  time.Now().Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(a.Secret)
}

// ParseToken は Authorization / WS 用トークン文字列からユーザー ID とロールを取り出す。
func (a *AuthUseCase) ParseToken(raw string) (uuid.UUID, entity.UserRole, error) {
	if len(a.Secret) == 0 || raw == "" {
		return uuid.Nil, "", ErrUnauthorized
	}
	tok, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, ErrUnauthorized
		}
		return a.Secret, nil
	})
	if err != nil || !tok.Valid {
		return uuid.Nil, "", ErrUnauthorized
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, "", ErrUnauthorized
	}
	sub, _ := claims["sub"].(string)
	uid, err := uuid.Parse(sub)
	if err != nil || uid == uuid.Nil {
		return uuid.Nil, "", ErrUnauthorized
	}
	roleStr, _ := claims["role"].(string)
	return uid, entity.UserRole(roleStr), nil
}
