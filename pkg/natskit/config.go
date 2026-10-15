package natskit

import (
	"errors"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

type Config struct {
	URLs           []string
	Name           string
	ConnectTimeout time.Duration
	ReconnectWait  time.Duration
	MaxReconnects  *int
	DrainTimeout   time.Duration

	Options []nats.Option
}

func (c Config) validate() error {
	if len(c.URLs) == 0 {
		return errors.New("natskit: at least one server URL is required")
	}

	for _, url := range c.URLs {
		if strings.TrimSpace(url) == "" {
			return errors.New("natskit: empty server URL")
		}
	}

	if c.ConnectTimeout < 0 {
		return errors.New("natskit: negative connect timeout")
	}
	if c.ReconnectWait < 0 {
		return errors.New("natskit: negative reconnect wait")
	}
	if c.DrainTimeout < 0 {
		return errors.New("natskit: negative drain timeout")
	}

	return nil
}

func (c Config) natsOptions() []nats.Option {
	opts := make([]nats.Option, 0, 5+len(c.Options))

	if c.Name != "" {
		opts = append(opts, nats.Name(c.Name))
	}
	if c.ConnectTimeout > 0 {
		opts = append(opts, nats.Timeout(c.ConnectTimeout))
	}
	if c.ReconnectWait > 0 {
		opts = append(opts, nats.ReconnectWait(c.ReconnectWait))
	}
	if c.MaxReconnects != nil {
		opts = append(opts, nats.MaxReconnects(*c.MaxReconnects))
	}
	if c.DrainTimeout > 0 {
		opts = append(opts, nats.DrainTimeout(c.DrainTimeout))
	}

	return append(opts, c.Options...)
}
