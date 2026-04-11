package controller

import (
	"errors"
	"net/http"

	"chat-app/usecase"

	"github.com/labstack/echo/v4"
)

// MediaController は画像アップロード。
type MediaController struct {
	Media MediaUseCase
}

// NewMediaController は MediaController を生成する。
func NewMediaController(media MediaUseCase) *MediaController {
	return &MediaController{Media: media}
}

// UploadImage は POST /media/images（multipart field: file）。
func (h *MediaController) UploadImage(c echo.Context) error {
	if _, err := CurrentUserID(c); err != nil {
		return err
	}
	file, err := c.FormFile("file")
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "file is required")
	}
	src, err := file.Open()
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "cannot open upload")
	}
	defer src.Close()

	ct := file.Header.Get("Content-Type")
	urlPath, err := h.Media.SaveUploadedImage(ct, src)
	if err != nil {
		return mapMediaError(c, err)
	}
	return c.JSON(http.StatusCreated, map[string]string{"url": urlPath})
}

func mapMediaError(c echo.Context, err error) error {
	status, msg := mediaUsecaseHTTP(err)
	return c.JSON(status, map[string]string{"error": msg})
}

func mediaUsecaseHTTP(err error) (int, string) {
	switch {
	case errors.Is(err, usecase.ErrUnsupportedMedia):
		return http.StatusUnsupportedMediaType, err.Error()
	case errors.Is(err, usecase.ErrUploadTooLarge):
		return http.StatusRequestEntityTooLarge, err.Error()
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
