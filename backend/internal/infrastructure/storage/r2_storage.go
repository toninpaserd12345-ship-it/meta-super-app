package storage

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"path/filepath"
	"strings"

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

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Normalize iOS QuickTime video to standard MP4
	if contentType == "video/quicktime" {
		contentType = "video/mp4"
	}

	// Generate a unique filename using UUID
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == ".mov" || ext == ".qt" {
		ext = ".mp4"
		if contentType == "application/octet-stream" {
			contentType = "video/mp4"
		}
	}
	if ext == "" || ext == ".bin" {
		switch contentType {
		case "video/mp4":
			ext = ".mp4"
		case "video/webm":
			ext = ".webm"
		case "video/quicktime":
			ext = ".mp4"
			contentType = "video/mp4"
		case "video/3gpp":
			ext = ".3gp"
		case "audio/mpeg", "audio/mp3":
			ext = ".mp3"
		case "audio/mp4", "audio/x-m4a", "audio/m4a":
			ext = ".m4a"
		case "audio/wav", "audio/x-wav", "audio/wave":
			ext = ".wav"
		case "audio/ogg", "audio/opus":
			ext = ".ogg"
		case "audio/webm":
			ext = ".webm"
		case "audio/aac", "audio/x-aac":
			ext = ".aac"
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/webp":
			ext = ".webp"
		case "image/gif":
			ext = ".gif"
		case "image/heic", "image/heif", "image/heic-sequence", "image/heif-sequence", "image/x-heic", "image/x-heif":
			ext = ".heic"
		default:
			if ext == "" {
				ext = ".bin"
			}
		}
	}

	// Normalize any QuickTime / MOV video format to standard MP4
	isMov := ext == ".mov" || ext == ".qt" || contentType == "video/quicktime"
	if isMov {
		ext = ".mp4"
		contentType = "video/mp4"
	}

	newFilename := uuid.New().String() + ext

	// Construct the object key (path in bucket). We only store the filename to support easy migration.
	objectKey := newFilename

	var uploadBody io.Reader = file
	if isMov {
		header := make([]byte, 1024)
		n, _ := io.ReadFull(file, header)
		if n >= 16 && string(header[4:8]) == "ftyp" {
			boxSize := binary.BigEndian.Uint32(header[0:4])
			if string(header[8:12]) == "qt  " {
				copy(header[8:12], "mp42")
				for i := uint32(16); i+4 <= boxSize && i+4 <= uint32(n); i += 4 {
					if string(header[i:i+4]) == "qt  " {
						copy(header[i:i+4], "mp42")
					}
				}
			}
		}
		uploadBody = io.MultiReader(bytes.NewReader(header[:n]), file)
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(s.bucketName),
		Key:          aws.String(objectKey),
		Body:         uploadBody,
		ContentType:  aws.String(contentType),
		CacheControl: aws.String("no-store"),
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
