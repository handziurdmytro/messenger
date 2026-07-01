package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	kafkainbound "github.com/handziurdmytro/messenger/message-service/internal/adapter/inbound/kafka"
	kafkaoutbound "github.com/handziurdmytro/messenger/message-service/internal/adapter/outbound/kafka"
	"github.com/handziurdmytro/messenger/message-service/internal/adapter/outbound/scylla"
	"github.com/handziurdmytro/messenger/message-service/internal/config"
)

type App struct {
	consumer   *kafkainbound.MessageConsumer
	publisher  *kafkaoutbound.MessagePublisher
	repository *scylla.MessageRepository
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

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
	)

	return &App{
		consumer:   consumer,
		publisher:  publisher,
		repository: repository,
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	return a.consumer.Start(ctx)
}

func (a *App) Shutdown() error {
	return errors.Join(
		a.consumer.Close(),
		a.publisher.Close(),
	)
}
