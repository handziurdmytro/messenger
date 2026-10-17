package usecase

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
)

func generateTestRSAKeyPEM(t *testing.T) []byte {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() setup error = %v, wantErr nil", err)
	}
	privDER := x509.MarshalPKCS1PrivateKey(priv)
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privDER,
	})
}

type transactorFunc func(
	context.Context,
	func(outbound.Repositories) error,
) error

func (f transactorFunc) WithinTransaction(
	ctx context.Context,
	fn func(outbound.Repositories) error,
) error {
	return f(ctx, fn)
}

func testTransactor(
	users outbound.UserRepository,
	tokens outbound.TokenRepository,
	outbox outbound.OutboxRepository,
) outbound.Transactor {
	return transactorFunc(func(
		_ context.Context,
		fn func(outbound.Repositories) error,
	) error {
		return fn(outbound.Repositories{
			Users:  users,
			Tokens: tokens,
			Outbox: outbox,
		})
	})
}
