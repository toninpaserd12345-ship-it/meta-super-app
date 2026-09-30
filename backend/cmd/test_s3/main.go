package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	r2AccountID := os.Getenv("R2_ACCOUNT_ID")
	r2AccessKeyID := os.Getenv("R2_ACCESS_KEY_ID")
	r2SecretAccessKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	r2BucketName := os.Getenv("R2_BUCKET_NAME")

	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", r2AccountID),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(r2AccessKeyID, r2SecretAccessKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	client := s3.NewFromConfig(cfg)
    body := strings.NewReader("dummy text")

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(r2BucketName),
		Key:         aws.String("test.txt"),
		Body:        body,
		ContentType: aws.String("text/plain"),
	})
	if err != nil {
		log.Fatalf("PutObject failed: %v", err)
	}
	fmt.Println("Upload successful!")
}
