package natskit

import (
	"context"
	"strings"

	"github.com/handziurdmytro/messenger/pkg/errkit"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Client struct {
	Conn *nats.Conn
	JS   jetstream.JetStream
}

func Connect(ctx context.Context, cfg Config) (*Client, error) {
	if err := cfg.validate(); err != nil {
		return nil, errkit.Wrap(ErrInvalidConfig, "natskit: validate config", err)
	}

	if err := ctx.Err(); err != nil {
		return nil, errkit.Wrap(ErrConnectionFailed, "natskit: connect context", err)
	}

	nc, err := nats.Connect(strings.Join(cfg.URLs, ","), cfg.natsOptions()...)
	if err != nil {
		return nil, errkit.Wrap(ErrConnectionFailed, "natskit: connect", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, errkit.Wrap(ErrJetStreamUnavailable, "natskit: initialize jetstream", err)
	}

	if _, err := js.AccountInfo(ctx); err != nil {
		nc.Close()
		return nil, errkit.Wrap(ErrJetStreamUnavailable, "natskit: check jetstream", err)
	}

	return &Client{
		Conn: nc,
		JS:   js,
	}, nil
}

func (c *Client) Drain() error {
	if err := c.Conn.Drain(); err != nil {
		return errkit.Wrap(ErrDrainFailed, "natskit: drain", err)
	}

	return nil
}

func (c *Client) Close() {
	c.Conn.Close()
}
