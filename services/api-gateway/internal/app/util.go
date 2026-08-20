package app

import (
	"context"
	"fmt"
	"time"

	"github.com/handziurdmytro/messenger/api-gateway/internal/adapter/outbound/authclient"
)

func fetchPublicKeyWithRetry(ctx context.Context, client *authclient.HTTPClient, attempts int, delay time.Duration) ([]byte, error) {
	if attempts <= 0 {
		return nil, fmt.Errorf("attempts must be positive, got %d", attempts)
	}

	var lastErr error
	for i := range attempts {
		if i > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			}
		}

		attemptCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		key, err := client.FetchPublicKey(attemptCtx)
		cancel()

		if err == nil {
			return key, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("fetch public key after %d attempts: %w", attempts, lastErr)
}
