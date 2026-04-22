package controller

import (
	"net/http"

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
// 201 の JSON は {"url":"/static/uploads/<uuid>.<ext>"}。メッセージ作成時 image_url にその値を渡す。
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

// UploadFile は POST /media/files（multipart field: file）。画像以外の許可 MIME のみ。
func (h *MediaController) UploadFile(c echo.Context) error {
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
	urlPath, err := h.Media.SaveUploadedFile(ct, file.Filename, src)
	if err != nil {
		return mapMediaError(c, err)
	}
	return c.JSON(http.StatusCreated, map[string]string{
		"url":           urlPath,
		"filename":      file.Filename,
		"content_type":  ct,
	})
}

func mapMediaError(c echo.Context, err error) error {
	return mapDomainError(c, err)
}
