package controller

import (
	"time"

	"chat-app/entity"

	"github.com/google/uuid"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userPublicJSON struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

type registerResponse struct {
	User        userPublicJSON `json:"user"`
	AccessToken string         `json:"access_token"`
}

func userToPublicJSON(u *entity.User) userPublicJSON {
	return userPublicJSON{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Role:      string(u.Role),
		Active:    u.Active,
		CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: u.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}
