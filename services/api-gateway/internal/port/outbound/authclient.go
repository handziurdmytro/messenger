package outbound

import "context"

type PublicKeyProvider interface {
	FetchPublicKey(ctx context.Context) ([]byte, error)
}
