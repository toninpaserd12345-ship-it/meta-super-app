package storage

import (
	"context"
	"fmt"
	"os"
	"strings"
	"log"

)

func TestR2() {
	r2AccountID := os.Getenv("R2_ACCOUNT_ID")
	r2AccessKeyID := os.Getenv("R2_ACCESS_KEY_ID")
	r2SecretAccessKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	r2BucketName := os.Getenv("R2_BUCKET_NAME")
	r2PublicURL := os.Getenv("R2_PUBLIC_URL")

	ctx := context.Background()
	storageSvc, err := NewR2StorageService(ctx, r2AccountID, r2AccessKeyID, r2SecretAccessKey, r2BucketName, r2PublicURL)
	if err != nil {
		log.Fatalf("Failed to initialize R2: %v", err)
	}

	reader := strings.NewReader("Hello R2 upload test!")
	url, err := storageSvc.Upload(ctx, "test-file.txt", "text/plain", reader, int64(reader.Len()))
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}

	fmt.Printf("Success! URL: %s\n", url)
}
