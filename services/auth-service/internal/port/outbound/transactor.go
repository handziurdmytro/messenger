package outbound

import "context"

type Repositories struct {
	Users  UserRepository
	Tokens TokenRepository
	Outbox OutboxRepository
}

type Transactor interface {
	WithinTransaction(
		ctx context.Context,
		fn func(Repositories) error,
	) error
}
