package usecase

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// 許可する画像 Content-Type → 拡張子
var imageExtByMIME = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

const maxImageBytes = 10 << 20 // 10 MiB

// MediaUseCase はアップロード画像を公開ディレクトリへ保存する。
type MediaUseCase struct {
	PublicRoot string // 例: data/public（この下に uploads/）
}

func NewMediaUseCase(publicRoot string) *MediaUseCase {
	return &MediaUseCase{PublicRoot: publicRoot}
}

// SaveUploadedImage は検証のうえファイルを保存し、URL パス（/static/uploads/...）を返す。
func (m *MediaUseCase) SaveUploadedImage(contentType string, r io.Reader) (urlPath string, err error) {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	ext, ok := imageExtByMIME[contentType]
	if !ok {
		return "", ErrUnsupportedMedia
	}
	dir := filepath.Join(m.PublicRoot, "uploads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := uuid.New().String() + ext
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	n, err := io.Copy(f, io.LimitReader(r, maxImageBytes+1))
	if err != nil {
		_ = os.Remove(path)
		return "", err
	}
	if n > maxImageBytes {
		_ = os.Remove(path)
		return "", ErrUploadTooLarge
	}
	return "/static/uploads/" + name, nil
}

// DeleteByPublicURL は当サービスが発行した /static/uploads/ 配下のファイルだけを削除する（他パスは無視）。
func (m *MediaUseCase) DeleteByPublicURL(urlPath string) {
	if m == nil || urlPath == "" {
		return
	}
	urlPath = strings.TrimSpace(urlPath)
	const prefix = "/static/uploads/"
	if !strings.HasPrefix(urlPath, prefix) {
		return
	}
	name := strings.TrimPrefix(urlPath, prefix)
	if name == "" || strings.Contains(name, "..") {
		return
	}
	full := filepath.Join(m.PublicRoot, "uploads", filepath.Clean(name))
	root := filepath.Clean(m.PublicRoot)
	if !strings.HasPrefix(filepath.Clean(full), root) {
		return
	}
	_ = os.Remove(full)
}
