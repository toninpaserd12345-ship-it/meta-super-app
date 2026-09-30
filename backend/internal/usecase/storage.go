package usecase

import (
	"context"
	"fmt"
	"mime/multipart"

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

	contentType := file.Header.Get("Content-Type")
	allowedTypes := map[string]bool{
		"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true,
		"video/mp4": true, "video/webm": true, "video/quicktime": true,
		"audio/mpeg": true, "audio/mp4": true, "audio/wav": true, "audio/x-wav": true,
		"audio/ogg": true, "audio/webm": true, "audio/aac": true,
	}
	if !allowedTypes[contentType] {
		return "", fmt.Errorf("unsupported media type %q; upload an image, MP4/WebM video, or common audio file", contentType)
	}

	return u.storage.UploadFile(ctx, file, folder)
}

// DeleteFile deletes a file from storage
func (u *StorageUseCase) DeleteFile(ctx context.Context, fileURL string) error {
	if u.storage == nil {
		return fmt.Errorf("media storage is not configured")
	}
	return u.storage.DeleteFile(ctx, fileURL)
}
