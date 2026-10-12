package usecase

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/handziurdmytro/messenger/message-service/internal/port/outbound"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

const MaxFileSize int64 = 128 << 20

type UploadFileInput struct {
	Key         string
	File        io.Reader
	Size        int64
	ContentType string
}

func (in UploadFileInput) Validate() error {
	if strings.TrimSpace(in.Key) == "" {
		return ErrEmptyKey
	}
	if in.File == nil || in.Size == 0 {
		return ErrEmptyFile
	}
	if in.Size > MaxFileSize {
		return ErrFileTooLarge
	}
	if in.Size < 0 {
		return ErrInvalidFileSize
	}
	if strings.TrimSpace(in.ContentType) == "" {
		return ErrUnsupportedMIME
	}
	return nil
}

type UploadFileUseCase struct {
	fileRepository outbound.FileRepository
}

func NewUploadFileUseCase(repo outbound.FileRepository) *UploadFileUseCase {
	return &UploadFileUseCase{fileRepository: repo}
}

func (uc *UploadFileUseCase) Execute(ctx context.Context, in UploadFileInput) error {
	if err := in.Validate(); err != nil {
		return err
	}

	var buf bytes.Buffer
	n, err := io.Copy(&buf, io.LimitReader(in.File, MaxFileSize+1))
	if err != nil {
		return errkit.Wrap(ErrInternal, "read file", err)
	}
	if n > MaxFileSize {
		return ErrFileTooLarge
	}
	if n == 0 {
		return ErrEmptyFile
	}

	detectedType := http.DetectContentType(buf.Bytes()[:min(n, 512)])
	if detectedType != in.ContentType {
		return ErrUnsupportedMIME
	}

	if err := uc.fileRepository.Upload(ctx, in.Key, bytes.NewReader(buf.Bytes()), detectedType); err != nil {
		return errkit.Wrap(ErrInternal, "upload file", err)
	}

	return nil
}
