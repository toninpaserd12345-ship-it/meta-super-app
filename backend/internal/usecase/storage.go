package usecase

import (
	"context"
	"fmt"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/meta-super-app/backend/internal/domain"
)

type StorageUseCase struct {
	storage domain.StorageService
}

func NewStorageUseCase(storage domain.StorageService) *StorageUseCase {
	return &StorageUseCase{
		storage: storage,
	}
}

// UploadFile validates and uploads a file
func (u *StorageUseCase) UploadFile(ctx context.Context, file *multipart.FileHeader, folder string) (string, error) {
	if u.storage == nil {
		return "", fmt.Errorf("media storage is not configured; use a public HTTPS URL instead")
	}
	// Keep uploads within a practical Messenger attachment size.
	if file.Size > 25*1024*1024 {
		return "", fmt.Errorf("file size exceeds 25MB limit")
	}

	rawContentType := file.Header.Get("Content-Type")
	contentType, _, _ := mime.ParseMediaType(rawContentType)
	contentType = strings.ToLower(strings.TrimSpace(contentType))

	// Fallback to file extension if Content-Type is missing or generic octet-stream
	if contentType == "" || contentType == "application/octet-stream" {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		switch ext {
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".png":
			contentType = "image/png"
		case ".gif":
			contentType = "image/gif"
		case ".webp":
			contentType = "image/webp"
		case ".mp4":
			contentType = "video/mp4"
		case ".webm":
			contentType = "video/webm"
		case ".mov":
			contentType = "video/quicktime"
		case ".3gp":
			contentType = "video/3gpp"
		case ".mp3":
			contentType = "audio/mpeg"
		case ".m4a":
			contentType = "audio/mp4"
		case ".wav":
			contentType = "audio/wav"
		case ".ogg", ".opus", ".oga":
			contentType = "audio/ogg"
		case ".weba":
			contentType = "audio/webm"
		case ".aac":
			contentType = "audio/aac"
		}
	}

	allowedTypes := map[string]bool{
		// Images
		"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true, "image/svg+xml": true, "image/bmp": true,
		// Videos
		"video/mp4": true, "video/webm": true, "video/quicktime": true, "video/3gpp": true, "video/x-matroska": true, "video/mpeg": true,
		// Voice & Audio ("void" / audio clips)
		"audio/mpeg": true, "audio/mp3": true, "audio/mp4": true, "audio/x-m4a": true, "audio/m4a": true,
		"audio/wav": true, "audio/x-wav": true, "audio/wave": true,
		"audio/ogg": true, "audio/opus": true, "audio/webm": true, "audio/aac": true, "audio/x-aac": true,
		"audio/3gpp": true, "audio/amr": true, "audio/flac": true,
	}

	if !allowedTypes[contentType] {
		return "", fmt.Errorf("unsupported media type %q; upload an image, MP4/WebM video, or common audio file", rawContentType)
	}

	// Update the file header with the clean, canonical content type
	file.Header.Set("Content-Type", contentType)

	return u.storage.UploadFile(ctx, file, folder)
}

// DeleteFile deletes a file from storage
func (u *StorageUseCase) DeleteFile(ctx context.Context, fileURL string) error {
	if u.storage == nil {
		return fmt.Errorf("media storage is not configured")
	}
	return u.storage.DeleteFile(ctx, fileURL)
}

// GetPublicURL returns the full URL for the frontend
func (u *StorageUseCase) GetPublicURL(path string) string {
	if u.storage == nil {
		return path
	}
	return u.storage.GetPublicURL(path)
}

// StripPublicURL removes the full URL before saving to DB
func (u *StorageUseCase) StripPublicURL(fullURL string) string {
	if u.storage == nil {
		return fullURL
	}
	return u.storage.StripPublicURL(fullURL)
}
