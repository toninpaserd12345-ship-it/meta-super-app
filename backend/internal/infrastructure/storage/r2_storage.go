package storage

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/domain"
)

type R2StorageService struct {
	client     *s3.Client
	bucketName string
	publicURL  string
}

// NewR2StorageService initializes a new Cloudflare R2 Storage Service
func NewR2StorageService(ctx context.Context, accountID, accessKey, secretKey, bucketName, publicURL string) (domain.StorageService, error) {
	if accountID == "" || accessKey == "" || secretKey == "" || bucketName == "" {
		return nil, fmt.Errorf("missing required Cloudflare R2 credentials")
	}

	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load R2 config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	// Ensure publicURL has no trailing slash
	publicURL = strings.TrimSuffix(publicURL, "/")

	return &R2StorageService{
		client:     client,
		bucketName: bucketName,
		publicURL:  publicURL,
	}, nil
}

func (s *R2StorageService) UploadFile(ctx context.Context, fileHeader *multipart.FileHeader, folder string) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Generate a unique filename using UUID
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == "" {
		ext = ".bin"
	}
	
	newFilename := uuid.New().String() + ext
	
	// Construct the object key (path in bucket)
	var objectKey string
	if folder != "" {
		folder = strings.Trim(folder, "/")
		objectKey = fmt.Sprintf("%s/%s/%s", folder, time.Now().Format("2006/01"), newFilename)
	} else {
		objectKey = fmt.Sprintf("uploads/%s/%s", time.Now().Format("2006/01"), newFilename)
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(objectKey),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to R2: %w", err)
	}

	return objectKey, nil
}

func (s *R2StorageService) GetPublicURL(path string) string {
	if s.publicURL == "" || path == "" || strings.HasPrefix(path, "http") {
		return path
	}
	return fmt.Sprintf("%s/%s", s.publicURL, path)
}

func (s *R2StorageService) StripPublicURL(fullURL string) string {
	if s.publicURL != "" && strings.HasPrefix(fullURL, s.publicURL) {
		path := strings.TrimPrefix(fullURL, s.publicURL)
		return strings.TrimPrefix(path, "/")
	}
	return fullURL
}

func (s *R2StorageService) DeleteFile(ctx context.Context, fileURL string) error {
	// Extract the object key from the public URL
	var objectKey string
	
	if strings.HasPrefix(fileURL, "http") {
		u, err := url.Parse(fileURL)
		if err != nil {
			return fmt.Errorf("invalid file URL: %w", err)
		}
		objectKey = strings.TrimPrefix(u.Path, "/")
	} else {
		objectKey = fileURL
	}

	if objectKey == "" {
		return fmt.Errorf("empty object key")
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(objectKey),
	})
	
	if err != nil {
		return fmt.Errorf("failed to delete file from R2: %w", err)
	}

	return nil
}
