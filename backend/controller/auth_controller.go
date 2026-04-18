package controller

import (
	"errors"
	"net/http"

	"chat-app/usecase"

	"github.com/labstack/echo/v4"
)

// AuthController はログイン・登録。
type AuthController struct {
	Auth                AuthUseCase
	BootstrapAdminEmail string
}

// NewAuthController は AuthController を生成する。
func NewAuthController(auth AuthUseCase, bootstrapAdminEmail string) *AuthController {
	return &AuthController{Auth: auth, BootstrapAdminEmail: bootstrapAdminEmail}
}

// Login は POST /auth/login。
func (h *AuthController) Login(c echo.Context) error {
	var body loginRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	token, err := h.Auth.Login(c.Request().Context(), body.Email, body.Password)
	if err != nil {
		return mapAuthError(c, err)
	}
	return c.JSON(http.StatusOK, loginResponse{AccessToken: token})
}

// Register は POST /auth/register。
func (h *AuthController) Register(c echo.Context) error {
	var body registerRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	u, err := h.Auth.Register(c.Request().Context(), body.Name, body.Email, body.Password, h.BootstrapAdminEmail)
	if err != nil {
		return mapAuthError(c, err)
	}
	token, err := h.Auth.CreateTokenForUser(c.Request().Context(), u.ID)
	if err != nil {
		return mapAuthError(c, err)
	}
	return c.JSON(http.StatusCreated, registerResponse{
		User:        userToPublicJSON(u),
		AccessToken: token,
	})
}

// Me は GET /me。ログイン中ユーザーのメール・ID 等（公開フィールドのみ）。
func (h *AuthController) Me(c echo.Context) error {
	uid, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	u, err := h.Auth.CurrentUser(c.Request().Context(), uid)
	if err != nil {
		return mapAuthError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]userPublicJSON{"user": userToPublicJSON(u)})
}

func mapAuthError(c echo.Context, err error) error {
	status, msg := authUsecaseHTTP(err)
	return c.JSON(status, map[string]string{"error": msg})
}

func authUsecaseHTTP(err error) (int, string) {
	switch {
	case errors.Is(err, usecase.ErrInvalidCredentials):
		return http.StatusUnauthorized, err.Error()
	case errors.Is(err, usecase.ErrInvalidDisplayName),
		errors.Is(err, usecase.ErrWeakPassword):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, usecase.ErrEmailAlreadyInUse):
		return http.StatusConflict, err.Error()
	case errors.Is(err, usecase.ErrUserNotFound):
		return http.StatusNotFound, err.Error()
	case errors.Is(err, usecase.ErrInvalidUserID):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, usecase.ErrInactiveUser):
		return http.StatusForbidden, err.Error()
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
