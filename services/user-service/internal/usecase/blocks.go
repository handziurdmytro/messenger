package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/handziurdmytro/messenger/user-service/internal/domain/block"
	"github.com/handziurdmytro/messenger/user-service/internal/domain/user"
	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

const (
	defaultBlocksLimit = 50
	maxBlocksLimit     = 200
)

type BlockUserUseCase struct {
	users  outbound.UserRepository
	blocks outbound.BlockRepository
}

func NewBlockUserUseCase(users outbound.UserRepository, blocks outbound.BlockRepository) *BlockUserUseCase {
	return &BlockUserUseCase{users: users, blocks: blocks}
}

func (uc *BlockUserUseCase) Execute(ctx context.Context, userID, blockedID user.ID) error {
	b, err := block.New(userID, blockedID, time.Now().UTC())
	if err != nil {
		return err
	}

	target, err := uc.users.FindByID(ctx, blockedID)
	if err != nil {
		return fmt.Errorf("find blocked user: %w", err)
	}
	if target.IsDeleted() {
		return fmt.Errorf("find blocked user: %w", user.ErrNotFound)
	}

	if err := uc.blocks.Add(ctx, b); err != nil {
		return fmt.Errorf("block user: %w", err)
	}

	return nil
}

type UnblockUserUseCase struct {
	blocks outbound.BlockRepository
}

func NewUnblockUserUseCase(blocks outbound.BlockRepository) *UnblockUserUseCase {
	return &UnblockUserUseCase{blocks: blocks}
}

func (uc *UnblockUserUseCase) Execute(ctx context.Context, userID, blockedID user.ID) error {
	if userID.IsZero() {
		return block.ErrUserIDEmpty
	}
	if blockedID.IsZero() {
		return block.ErrBlockedIDEmpty
	}

	if err := uc.blocks.Remove(ctx, userID, blockedID); err != nil {
		return fmt.Errorf("unblock user: %w", err)
	}

	return nil
}

type ListBlockedUseCase struct {
	blocks outbound.BlockRepository
}

func NewListBlockedUseCase(blocks outbound.BlockRepository) *ListBlockedUseCase {
	return &ListBlockedUseCase{blocks: blocks}
}

func (uc *ListBlockedUseCase) Execute(ctx context.Context, userID user.ID, limit, offset int) ([]user.User, error) {
	if limit <= 0 || limit > maxBlocksLimit {
		limit = defaultBlocksLimit
	}

	users, err := uc.blocks.ListBlocked(ctx, userID, outbound.Page{Limit: limit, Offset: max(offset, 0)})
	if err != nil {
		return nil, fmt.Errorf("list blocked users: %w", err)
	}

	return users, nil
}
