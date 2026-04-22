package controller

import (
	"errors"
	"net/http"

	"chat-app/usecase"

	"github.com/labstack/echo/v4"
)

// AuthController はログイン・登録・プロフィール・パスワード周り。
type AuthController struct {
	Auth                AuthUseCase
	Profile             ProfileUseCase
	BootstrapAdminEmail string
}

// NewAuthController は AuthController を生成する。
func NewAuthController(auth AuthUseCase, profile ProfileUseCase, bootstrapAdminEmail string) *AuthController {
	return &AuthController{Auth: auth, Profile: profile, BootstrapAdminEmail: bootstrapAdminEmail}
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
	res, err := h.Auth.Register(c.Request().Context(), body.Name, body.Email, body.Password, h.BootstrapAdminEmail)
	if err != nil {
		return mapAuthError(c, err)
	}
	if res.NeedsEmailVerification {
		return c.JSON(http.StatusCreated, registerResponse{
			User:                   userToPublicJSON(res.User),
			VerificationRequired:   true,
		})
	}
	token, err := h.Auth.CreateTokenForUser(c.Request().Context(), res.User.ID)
	if err != nil {
		return mapAuthError(c, err)
	}
	return c.JSON(http.StatusCreated, registerResponse{
		User:         userToPublicJSON(res.User),
		AccessToken:  token,
	})
}

// Me は GET /me。
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

// PatchMe は PATCH /me。
func (h *AuthController) PatchMe(c echo.Context) error {
	if h.Profile == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "profile not configured")
	}
	uid, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	var body patchMeRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	u, err := h.Profile.UpdateProfile(c.Request().Context(), uid, usecase.UpdateProfileInput{
		Name: body.Name,
	})
	if err != nil {
		return mapDomainError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]userPublicJSON{"user": userToPublicJSON(u)})
}

// ForgotPassword は POST /auth/forgot-password。
func (h *AuthController) ForgotPassword(c echo.Context) error {
	var body forgotPasswordRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	_ = h.Auth.ForgotPassword(c.Request().Context(), body.Email)
	return c.NoContent(http.StatusOK)
}

// ResetPassword は POST /auth/reset-password。
func (h *AuthController) ResetPassword(c echo.Context) error {
	var body resetPasswordRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	if err := h.Auth.ResetPassword(c.Request().Context(), body.Token, body.NewPassword); err != nil {
		return mapAuthError(c, err)
	}
	return c.NoContent(http.StatusOK)
}

// VerifyEmail は POST /auth/verify-email。
func (h *AuthController) VerifyEmail(c echo.Context) error {
	var body verifyEmailRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	if err := h.Auth.VerifyEmail(c.Request().Context(), body.Token); err != nil {
		return mapAuthError(c, err)
	}
	return c.NoContent(http.StatusOK)
}

// ChangePassword は POST /auth/change-password（JWT 必須）。
func (h *AuthController) ChangePassword(c echo.Context) error {
	uid, err := CurrentUserID(c)
	if err != nil {
		return err
	}
	var body changePasswordRequest
	if err := c.Bind(&body); err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid json body")
	}
	if err := h.Auth.ChangePassword(c.Request().Context(), uid, body.CurrentPassword, body.NewPassword); err != nil {
		return mapAuthError(c, err)
	}
	return c.NoContent(http.StatusOK)
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
	case errors.Is(err, usecase.ErrEmailNotVerified):
		return http.StatusForbidden, err.Error()
	case errors.Is(err, usecase.ErrInvalidToken):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, usecase.ErrWrongPassword):
		return http.StatusUnauthorized, err.Error()
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
