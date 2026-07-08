package usecase

import (
	"context"
	"fmt"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

type SearchUsersUseCase struct {
	users outbound.UserRepository
}

func NewSearchUsersUseCase(users outbound.UserRepository) *SearchUsersUseCase {
	return &SearchUsersUseCase{users: users}
}

func (uc *SearchUsersUseCase) Execute(ctx context.Context, query string, limit int) ([]user.User, error) {
	if query == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	users, err := uc.users.Search(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}

	return users, nil
}
