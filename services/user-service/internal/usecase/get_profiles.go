package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

const MaxProfilesPerRequest = 100

var ErrTooManyIDs = errors.New("too many ids")

type GetProfilesUseCase struct {
	users outbound.UserRepository
}

func NewGetProfilesUseCase(users outbound.UserRepository) *GetProfilesUseCase {
	return &GetProfilesUseCase{users: users}
}

func (uc *GetProfilesUseCase) Execute(ctx context.Context, ids []user.ID) ([]user.User, error) {
	unique := make([]user.ID, 0, len(ids))
	seen := make(map[user.ID]struct{}, len(ids))
	for _, id := range ids {
		if id.IsZero() {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}

	if len(unique) == 0 {
		return []user.User{}, nil
	}
	if len(unique) > MaxProfilesPerRequest {
		return nil, ErrTooManyIDs
	}

	users, err := uc.users.FindByIDs(ctx, unique)
	if err != nil {
		return nil, fmt.Errorf("get profiles: %w", err)
	}

	return users, nil
}
