package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"golang.org/x/sync/errgroup"

	natsinbound "github.com/handziurdmytro/messenger/message-service/internal/adapter/inbound/nats"
	"github.com/handziurdmytro/messenger/message-service/internal/adapter/inbound/rest"
	natsoutbound "github.com/handziurdmytro/messenger/message-service/internal/adapter/outbound/nats"
	s3outbound "github.com/handziurdmytro/messenger/message-service/internal/adapter/outbound/s3"
	"github.com/handziurdmytro/messenger/message-service/internal/adapter/outbound/scylla"
	"github.com/handziurdmytro/messenger/message-service/internal/config"
	"github.com/handziurdmytro/messenger/message-service/internal/usecase"
	"github.com/handziurdmytro/messenger/pkg/natskit"
)

type App struct {
	server     *rest.Server
	consumer   *natsinbound.MessageConsumer
	repository *scylla.MessageRepository
	natsClient *natskit.Client
	done       chan struct{}
}

func New(ctx context.Context, cfg config.Config, log *slog.Logger) (*App, error) {
	repo, err := scylla.NewMessageRepository(cfg.Scylla.Hosts)
	if err != nil {
		return nil, fmt.Errorf("init scylla: %w", err)
	}

	maxReconnects := cfg.NATS.MaxReconnects
	natsClient, err := natskit.Connect(ctx, natskit.Config{
		URLs:           cfg.NATS.URLs,
		Name:           cfg.NATS.Name,
		ConnectTimeout: cfg.NATS.ConnectTimeout,
		ReconnectWait:  cfg.NATS.ReconnectWait,
		MaxReconnects:  &maxReconnects,
		DrainTimeout:   cfg.NATS.DrainTimeout,
	})
	if err != nil {
		repo.Close()
		return nil, fmt.Errorf("connect nats: %w", err)
	}

	consumer, err := natsClient.JS.Consumer(ctx, cfg.NATS.Stream, cfg.NATS.Consumer)
	if err != nil {
		natsClient.Close()
		repo.Close()
		return nil, fmt.Errorf("get jetstream consumer: %w", err)
	}

	s3Client := s3.New(s3.Options{
		Region:       cfg.S3.Region,
		BaseEndpoint: aws.String(cfg.S3.Endpoint),
		UsePathStyle: true,
		Credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     cfg.S3.AccessKey,
				SecretAccessKey: cfg.S3.SecretKey,
				Source:          "static",
			}, nil
		}),
	})

	files := s3outbound.NewFileRepository(s3Client, cfg.S3.Bucket)
	publisher := natsoutbound.NewMessagePublisher(natskit.NewPublisher(natsClient.JS))
	sendMessage := usecase.NewSendMessage(repo, publisher)

	server := rest.NewServer(
		cfg.Server.Port,
		usecase.NewGetHistory(repo),
		usecase.NewGetMessage(repo),
		usecase.NewUploadFileUseCase(files),
		usecase.NewDownloadFileUseCase(files),
		log,
	)

	return &App{
		server:     server,
		consumer:   natsinbound.NewMessageConsumer(consumer, natsClient.JS, sendMessage, log),
		repository: repo,
		natsClient: natsClient,
		done:       make(chan struct{}),
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	defer close(a.done)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		if err := a.server.Start(); err != nil {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		if err := a.consumer.Run(gCtx); err != nil {
			return fmt.Errorf("nats consumer: %w", err)
		}
		return nil
	})

	return g.Wait()
}

func (a *App) Shutdown(ctx context.Context) error {
	httpErr := a.server.Shutdown(ctx)

	select {
	case <-a.done:
	case <-ctx.Done():
		a.natsClient.Close()
		a.repository.Close()
		return errors.Join(httpErr, ctx.Err())
	}

	if err := ctx.Err(); err != nil {
		a.natsClient.Close()
		a.repository.Close()
		return errors.Join(httpErr, err)
	}

	drainErr := a.natsClient.Drain()
	a.repository.Close()

	return errors.Join(httpErr, drainErr)
}
