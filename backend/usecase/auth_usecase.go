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

// AuthUseCase はログイン・登録・トークン・メール検証・パスワードリセット。
type AuthUseCase struct {
	Users              repository.UserRepository
	Tokens             *auth.TokenService
	TokRepo            repository.AuthTokenRepository
	Mailer             Mailer
	RequireEmailVerify bool
	AppPublicURL       string // メール内リンク用（末尾スラッシュなし）
}

func NewAuthUseCase(users repository.UserRepository, tokens *auth.TokenService) *AuthUseCase {
	return &AuthUseCase{Users: users, Tokens: tokens}
}

// WithEmailFlow はメール検証・パスワードリセット用の依存を付与する（nil 可）。
func (a *AuthUseCase) WithEmailFlow(tok repository.AuthTokenRepository, m Mailer, requireVerify bool, appPublicURL string) *AuthUseCase {
	a.TokRepo = tok
	a.Mailer = m
	a.RequireEmailVerify = requireVerify
	a.AppPublicURL = strings.TrimRight(strings.TrimSpace(appPublicURL), "/")
	return a
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
	if u == nil {
		return "", ErrInvalidCredentials
	}
	if !u.Active {
		return "", ErrInactiveUser
	}
	if err := auth.ComparePassword(u.PasswordHash, password); err != nil {
		return "", ErrInvalidCredentials
	}
	if a.RequireEmailVerify && u.EmailVerifiedAt == nil {
		return "", ErrEmailNotVerified
	}
	token, err := a.Tokens.Issue(u.ID, u.Role)
	if err != nil {
		return "", err
	}
	return token, nil
}

// RegisterResult は登録直後の状態。
type RegisterResult struct {
	User                 *entity.User
	NeedsEmailVerification bool
}

// Register はユーザーを作成する（最初の管理者は環境変数 BOOTSTRAP_ADMIN_EMAIL と一致する場合のみ admin）。
func (a *AuthUseCase) Register(ctx context.Context, name, email, password, bootstrapAdminEmail string) (*RegisterResult, error) {
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
	var verifiedAt *time.Time
	needs := false
	if a.RequireEmailVerify && a.TokRepo != nil && a.Mailer != nil {
		needs = true
		verifiedAt = nil
	} else {
		verifiedAt = &now
	}
	u := &entity.User{
		ID:              uuid.New(),
		Name:            name,
		Email:           email,
		PasswordHash:    hash,
		Role:            role,
		Active:          true,
		EmailVerifiedAt: verifiedAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := a.Users.Create(ctx, u); err != nil {
		return nil, err
	}
	if needs {
		if err := a.sendEmailVerifyToken(ctx, u); err != nil {
			return nil, err
		}
	}
	return &RegisterResult{User: u, NeedsEmailVerification: needs}, nil
}

func (a *AuthUseCase) sendEmailVerifyToken(ctx context.Context, u *entity.User) error {
	plain, err := auth.OpaqueTokenHex()
	if err != nil {
		return err
	}
	hash := auth.HashOpaqueToken(plain)
	exp := time.Now().UTC().Add(48 * time.Hour)
	row := &repository.AuthToken{
		ID:        uuid.New(),
		Kind:      repository.AuthTokenKindEmailVerify,
		TokenHash: hash,
		UserID:    u.ID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: exp,
	}
	if err := a.TokRepo.Create(ctx, row); err != nil {
		return err
	}
	link := a.AppPublicURL + "/verify-email?token=" + plain
	body := "次のリンクでメールアドレスを確認してください:\n" + link
	return a.Mailer.Send(ctx, u.Email, "メールアドレスの確認", body)
}

// VerifyEmail はワンタイムトークンで検証済みにする。
func (a *AuthUseCase) VerifyEmail(ctx context.Context, plainToken string) error {
	if a.TokRepo == nil {
		return ErrInvalidToken
	}
	plainToken = strings.TrimSpace(plainToken)
	if plainToken == "" {
		return ErrInvalidToken
	}
	hash := auth.HashOpaqueToken(plainToken)
	row, err := a.TokRepo.FindValidByHash(ctx, repository.AuthTokenKindEmailVerify, hash)
	if err != nil || row == nil {
		return ErrInvalidToken
	}
	now := time.Now().UTC()
	if err := a.TokRepo.MarkUsed(ctx, row.ID, now); err != nil {
		return err
	}
	u, err := a.Users.FindByID(ctx, row.UserID)
	if err != nil || u == nil {
		return ErrInvalidToken
	}
	u.EmailVerifiedAt = &now
	u.UpdatedAt = now
	return a.Users.Update(ctx, u)
}

// ForgotPassword は常に成功を装い、存在するユーザーにリセットメールを送る。
func (a *AuthUseCase) ForgotPassword(ctx context.Context, email string) error {
	if a.TokRepo == nil || a.Mailer == nil {
		return nil
	}
	email = strings.TrimSpace(strings.ToLower(email))
	u, err := a.Users.FindByEmail(ctx, email)
	if err != nil {
		return err
	}
	if u == nil || !u.Active {
		return nil
	}
	plain, err := auth.OpaqueTokenHex()
	if err != nil {
		return err
	}
	hash := auth.HashOpaqueToken(plain)
	row := &repository.AuthToken{
		ID:        uuid.New(),
		Kind:      repository.AuthTokenKindPasswordReset,
		TokenHash: hash,
		UserID:    u.ID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	if err := a.TokRepo.Create(ctx, row); err != nil {
		return err
	}
	link := a.AppPublicURL + "/reset-password?token=" + plain
	body := "パスワード再設定:\n" + link
	return a.Mailer.Send(ctx, u.Email, "パスワード再設定", body)
}

// ResetPassword はワンタイムトークンでパスワードを更新する。
func (a *AuthUseCase) ResetPassword(ctx context.Context, plainToken, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrWeakPassword
	}
	if a.TokRepo == nil {
		return ErrInvalidToken
	}
	hash := auth.HashOpaqueToken(strings.TrimSpace(plainToken))
	row, err := a.TokRepo.FindValidByHash(ctx, repository.AuthTokenKindPasswordReset, hash)
	if err != nil || row == nil {
		return ErrInvalidToken
	}
	now := time.Now().UTC()
	if err := a.TokRepo.MarkUsed(ctx, row.ID, now); err != nil {
		return err
	}
	h, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	u, err := a.Users.FindByID(ctx, row.UserID)
	if err != nil || u == nil {
		return ErrInvalidToken
	}
	u.PasswordHash = h
	u.UpdatedAt = now
	return a.Users.Update(ctx, u)
}

// ChangePassword はログイン済みユーザーが現在パスワード確認のうえ変更する。
func (a *AuthUseCase) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrWeakPassword
	}
	u, err := a.Users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil || !u.Active {
		return ErrUserNotFound
	}
	if err := auth.ComparePassword(u.PasswordHash, currentPassword); err != nil {
		return ErrWrongPassword
	}
	h, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	u.PasswordHash = h
	u.UpdatedAt = now
	return a.Users.Update(ctx, u)
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
	if a.RequireEmailVerify && u.EmailVerifiedAt == nil {
		return "", ErrEmailNotVerified
	}
	return a.Tokens.Issue(u.ID, u.Role)
}

// CurrentUser は JWT 相当のユーザー ID から公開表示用のユーザーを返す。
func (a *AuthUseCase) CurrentUser(ctx context.Context, userID uuid.UUID) (*entity.User, error) {
	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	u, err := a.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}
	if !u.Active {
		return nil, ErrInactiveUser
	}
	return u, nil
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
