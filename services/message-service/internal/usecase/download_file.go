package usecase

import (
	"context"
	"io"
	"strings"

	"github.com/handziurdmytro/messenger/message-service/internal/port/outbound"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

type DownloadFileUseCase struct {
	fileRepository outbound.FileRepository
}

func NewDownloadFileUseCase(repo outbound.FileRepository) *DownloadFileUseCase {
	return &DownloadFileUseCase{fileRepository: repo}
}

func (uc *DownloadFileUseCase) Execute(ctx context.Context, key string) (io.Reader, error) {
	if strings.TrimSpace(key) == "" {
		return nil, ErrEmptyKey
	}

	file, err := uc.fileRepository.Download(ctx, key)
	if err != nil {
		return nil, errkit.Wrap(ErrInternal, "download file", err)
	}

	return file, nil
}
