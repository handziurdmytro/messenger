package usecase

import (
	"context"
	"io"
	"strings"

	"github.com/handziurdmytro/messenger/pkg/errkit"
)

type DownloadFile struct {
	fileRepository FileRepository
}

func NewDownloadFileUseCase(repo FileRepository) *DownloadFile {
	return &DownloadFile{fileRepository: repo}
}

func (uc *DownloadFile) Execute(ctx context.Context, key string) (io.Reader, error) {
	if strings.TrimSpace(key) == "" {
		return nil, ErrEmptyKey
	}

	file, err := uc.fileRepository.Download(ctx, key)
	if err != nil {
		return nil, errkit.Wrap(ErrInternal, "download file", err)
	}

	return file, nil
}
