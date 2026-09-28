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
	// Validate file size (e.g., max 10MB)
	if file.Size > 10*1024*1024 {
		return "", fmt.Errorf("file size exceeds 10MB limit")
	}

	// Validate content type
	contentType := file.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/gif" && contentType != "image/webp" {
		return "", fmt.Errorf("invalid file type: only JPEG, PNG, GIF, and WEBP are allowed")
	}

	return u.storage.UploadFile(ctx, file, folder)
}

// DeleteFile deletes a file from storage
func (u *StorageUseCase) DeleteFile(ctx context.Context, fileURL string) error {
	return u.storage.DeleteFile(ctx, fileURL)
}
