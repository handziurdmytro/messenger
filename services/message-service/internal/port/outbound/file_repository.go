package outbound

import (
	"context"
	"io"
)

type FileRepository interface {
	Upload(ctx context.Context, key string, file io.Reader, contentType string) error
	Download(ctx context.Context, key string) (io.Reader, error)
	Delete(ctx context.Context, key string) error
}
