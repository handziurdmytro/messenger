package outbound

import "context"

type Repositories struct {
	Users  UserRepository
	Tokens TokenRepository
}

type Transactor interface {
	WithinTransaction(
		ctx context.Context,
		fn func(Repositories) error,
	) error
}
