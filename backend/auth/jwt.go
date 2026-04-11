package auth

import (
	"time"

	"chat-app/entity"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// DefaultTokenTTL はアクセストークンの既定有効期限。
const DefaultTokenTTL = 72 * time.Hour

// TokenService は HS256 JWT の発行・検証を担う。
type TokenService struct {
	secret []byte
	ttl    time.Duration
}

// NewTokenService は TokenService を生成する。ttl が 0 以下なら DefaultTokenTTL を使う。
func NewTokenService(secret []byte, ttl time.Duration) *TokenService {
	if ttl <= 0 {
		ttl = DefaultTokenTTL
	}
	return &TokenService{secret: secret, ttl: ttl}
}

// Issue は sub / role / exp を含む JWT を返す。
func (s *TokenService) Issue(userID uuid.UUID, role entity.UserRole) (string, error) {
	if len(s.secret) == 0 {
		return "", ErrInvalidToken
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  userID.String(),
		"role": string(role),
		"exp":  now.Add(s.ttl).Unix(),
		"iat":  now.Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.secret)
}

// Parse は生トークン文字列からユーザー ID とロールを取り出す。
func (s *TokenService) Parse(raw string) (uuid.UUID, entity.UserRole, error) {
	if len(s.secret) == 0 || raw == "" {
		return uuid.Nil, "", ErrInvalidToken
	}
	tok, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	})
	if err != nil || !tok.Valid {
		return uuid.Nil, "", ErrInvalidToken
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, "", ErrInvalidToken
	}
	sub, _ := claims["sub"].(string)
	uid, err := uuid.Parse(sub)
	if err != nil || uid == uuid.Nil {
		return uuid.Nil, "", ErrInvalidToken
	}
	roleStr, _ := claims["role"].(string)
	return uid, entity.UserRole(roleStr), nil
}
