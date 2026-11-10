package usecase

import (
	"context"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
)

type ApplyBlockUseCase struct {
	blocks BlockRepository
}

func NewApplyBlock(blocks BlockRepository) *ApplyBlockUseCase {
	return &ApplyBlockUseCase{blocks: blocks}
}

func (uc *ApplyBlockUseCase) Execute(ctx context.Context, change BlockChange) error {
	if _, err := chat.ParseUserID(string(change.UserID)); err != nil {
		return err
	}
	if _, err := chat.ParseUserID(string(change.BlockedID)); err != nil {
		return err
	}

	return uc.blocks.Apply(ctx, change)
}
