package natskit

import "errors"

var (
	ErrInvalidConfig        = errors.New("invalid nats configuration")
	ErrConnectionFailed     = errors.New("nats connection failed")
	ErrJetStreamUnavailable = errors.New("jetstream unavailable")
	ErrDrainFailed          = errors.New("nats connection drain failed")
	ErrInvalidMessageID     = errors.New("invalid NATS message ID")
	ErrPublishFailed        = errors.New("nats publish failed")
)
