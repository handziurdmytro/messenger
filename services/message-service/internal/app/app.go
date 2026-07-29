package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	kafkainbound "github.com/handziurdmytro/messenger/message-service/internal/adapter/inbound/kafka"
	"github.com/handziurdmytro/messenger/message-service/internal/adapter/inbound/rest"
	kafkaoutbound "github.com/handziurdmytro/messenger/message-service/internal/adapter/outbound/kafka"
	"github.com/handziurdmytro/messenger/message-service/internal/adapter/outbound/scylla"
	"github.com/handziurdmytro/messenger/message-service/internal/config"
)

type App struct {
	consumer   *kafkainbound.MessageConsumer
	publisher  *kafkaoutbound.MessagePublisher
	repository *scylla.MessageRepository
	server     *rest.Server
}

func New(cfg config.Config, baseLogger *slog.Logger) (*App, error) {
	brokers := strings.Split(cfg.KafkaBrokers, ",")
	scyllaHosts := strings.Split(cfg.ScyllaHosts, ",")

	repository, err := scylla.NewMessageRepository(scyllaHosts)
	if err != nil {
		return nil, fmt.Errorf("init scylla: %w", err)
	}

	publisher := kafkaoutbound.NewMessagePublisher(brokers, cfg.KafkaOutTopic)

	consumer := kafkainbound.NewMessageConsumer(
		brokers,
		cfg.KafkaInTopic,
		cfg.KafkaGroupID,
		repository,
		publisher,
		baseLogger,
	)

	server := rest.NewServer(cfg.ServerPort, repository, baseLogger)

	return &App{
		consumer:   consumer,
		publisher:  publisher,
		repository: repository,
		server:     server,
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	errch := make(chan error, 2)

	go func() {
		if err := a.consumer.Start(ctx); err != nil {
			errch <- fmt.Errorf("consumer: %w", err)
		}
	}()

	go func() {
		if err := a.server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errch <- fmt.Errorf("rest server: %w", err)
		}
	}()

	select {
	case err := <-errch:
		return err
	case <-ctx.Done():
		return nil
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	return errors.Join(
		a.consumer.Close(),
		a.publisher.Close(),
		a.server.Shutdown(ctx),
	)
}
