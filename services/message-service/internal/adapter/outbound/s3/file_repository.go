package s3

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/handziurdmytro/messenger/message-service/internal/port/outbound"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

type FileRepository struct {
	client   *s3.Client
	transfer *transfermanager.Client
	bucket   string
}

func NewFileRepository(client *s3.Client, bucket string) *FileRepository {
	return &FileRepository{
		client:   client,
		transfer: transfermanager.New(client),
		bucket:   bucket,
	}
}

func (r *FileRepository) Upload(ctx context.Context, key string, file io.Reader, contentType string) error {
	_, err := r.transfer.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return errkit.Wrap(outbound.ErrPersistence, "s3: upload file", err)
	}
	return nil
}

func (r *FileRepository) Download(ctx context.Context, key string) (io.Reader, error) {
	result, err := r.transfer.GetObject(ctx, &transfermanager.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, errkit.Wrap(outbound.ErrPersistence, "s3: download file", err)
	}
	return result.Body, nil
}

func (r *FileRepository) Delete(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return errkit.Wrap(outbound.ErrPersistence, "s3: delete file", err)
	}
	return nil
}
