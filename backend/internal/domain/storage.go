package domain

import (
	"context"
	"mime/multipart"
)

// StorageService defines the interface for interacting with file storage systems (like Cloudflare R2 / AWS S3)
type StorageService interface {
	// UploadFile uploads a file and returns its public URL
	UploadFile(ctx context.Context, file *multipart.FileHeader, folder string) (string, error)
	// DeleteFile deletes a file by its URL or key
	DeleteFile(ctx context.Context, fileURL string) error
}
