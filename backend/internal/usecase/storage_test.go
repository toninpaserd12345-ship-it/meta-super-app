package usecase

import (
	"context"
	"mime/multipart"
	"net/textproto"
	"strings"
	"testing"
)

type fakeStorageService struct {
	uploadCalls int
	deleteCalls int
	file        *multipart.FileHeader
	folder      string
	fileURL     string
}

func (f *fakeStorageService) UploadFile(_ context.Context, file *multipart.FileHeader, folder string) (string, error) {
	f.uploadCalls++
	f.file = file
	f.folder = folder
	return "https://cdn.example.com/" + file.Filename, nil
}

func (f *fakeStorageService) DeleteFile(_ context.Context, fileURL string) error {
	f.deleteCalls++
	f.fileURL = fileURL
	return nil
}

func TestStorageUseCaseWithoutStorageReturnsSafeErrors(t *testing.T) {
	service := NewStorageUseCase(nil)

	if _, err := service.UploadFile(context.Background(), nil, "replies"); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("UploadFile() error = %v, want storage-not-configured error", err)
	}
	if err := service.DeleteFile(context.Background(), "https://cdn.example.com/file.jpg"); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("DeleteFile() error = %v, want storage-not-configured error", err)
	}
}

func TestStorageUseCaseAcceptsSupportedMediaTypes(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		filename    string
	}{
		{name: "jpeg image", contentType: "image/jpeg", filename: "photo.jpg"},
		{name: "png image", contentType: "image/png", filename: "photo.png"},
		{name: "gif image", contentType: "image/gif", filename: "photo.gif"},
		{name: "webp image", contentType: "image/webp", filename: "photo.webp"},
		{name: "mp4 video", contentType: "video/mp4", filename: "clip.mp4"},
		{name: "webm video", contentType: "video/webm", filename: "clip.webm"},
		{name: "quicktime video", contentType: "video/quicktime", filename: "clip.mov"},
		{name: "mpeg audio", contentType: "audio/mpeg", filename: "voice.mp3"},
		{name: "mp4 audio", contentType: "audio/mp4", filename: "voice.m4a"},
		{name: "wav audio", contentType: "audio/wav", filename: "voice.wav"},
		{name: "x-wav audio", contentType: "audio/x-wav", filename: "voice.wav"},
		{name: "ogg audio", contentType: "audio/ogg", filename: "voice.ogg"},
		{name: "webm audio", contentType: "audio/webm", filename: "voice.webm"},
		{name: "aac audio", contentType: "audio/aac", filename: "voice.aac"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &fakeStorageService{}
			service := NewStorageUseCase(storage)
			file := mediaFile(tt.filename, tt.contentType, 25*1024*1024)

			got, err := service.UploadFile(context.Background(), file, "replies")
			if err != nil {
				t.Fatalf("UploadFile() error = %v", err)
			}
			if got != "https://cdn.example.com/"+tt.filename {
				t.Fatalf("UploadFile() URL = %q", got)
			}
			if storage.uploadCalls != 1 || storage.file != file || storage.folder != "replies" {
				t.Fatalf("storage call = count %d, file %p, folder %q", storage.uploadCalls, storage.file, storage.folder)
			}
		})
	}
}

func TestStorageUseCaseRejectsUnsupportedMediaType(t *testing.T) {
	storage := &fakeStorageService{}
	service := NewStorageUseCase(storage)

	_, err := service.UploadFile(context.Background(), mediaFile("document.pdf", "application/pdf", 1024), "replies")
	if err == nil || !strings.Contains(err.Error(), "unsupported media type") {
		t.Fatalf("UploadFile() error = %v, want unsupported-media-type error", err)
	}
	if storage.uploadCalls != 0 {
		t.Fatalf("storage UploadFile() called %d times for rejected media", storage.uploadCalls)
	}
}

func TestStorageUseCaseRejectsFilesOver25MB(t *testing.T) {
	storage := &fakeStorageService{}
	service := NewStorageUseCase(storage)

	_, err := service.UploadFile(context.Background(), mediaFile("large.mp4", "video/mp4", 25*1024*1024+1), "replies")
	if err == nil || !strings.Contains(err.Error(), "25MB") {
		t.Fatalf("UploadFile() error = %v, want 25MB-limit error", err)
	}
	if storage.uploadCalls != 0 {
		t.Fatalf("storage UploadFile() called %d times for oversized media", storage.uploadCalls)
	}
}

func mediaFile(filename, contentType string, size int64) *multipart.FileHeader {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", contentType)
	return &multipart.FileHeader{Filename: filename, Header: header, Size: size}
}
