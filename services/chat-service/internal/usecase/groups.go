package usecase

import (
	"context"

	"github.com/handziurdmytro/messenger/chat-service/internal/domain/chat"
)

type CreateGroupInput struct {
	Owner   chat.UserID
	Title   string
	Members []chat.UserID
}

type CreateGroupUseCase struct {
	repo ChatRepository
}

func NewCreateGroup(repo ChatRepository) *CreateGroupUseCase {
	return &CreateGroupUseCase{repo: repo}
}

func (uc *CreateGroupUseCase) Execute(ctx context.Context, in CreateGroupInput) (chat.Chat, error) {
	c, err := chat.NewGroup(in.Owner, in.Title, in.Members)
	if err != nil {
		return chat.Chat{}, err
	}

	if err := uc.repo.Save(ctx, c); err != nil {
		return chat.Chat{}, err
	}

	return c, nil
}

type GroupChange func(c *chat.Chat, by chat.UserID) error

type UpdateGroupUseCase struct {
	repo ChatRepository
}

func NewUpdateGroup(repo ChatRepository) *UpdateGroupUseCase {
	return &UpdateGroupUseCase{repo: repo}
}

func (uc *UpdateGroupUseCase) Execute(ctx context.Context, chatID chat.ID, by chat.UserID, change GroupChange) (chat.Chat, error) {
	if _, err := chat.ParseUserID(string(by)); err != nil {
		return chat.Chat{}, err
	}

	c, err := uc.repo.FindByID(ctx, chatID)
	if err != nil {
		return chat.Chat{}, err
	}

	if !c.HasParticipant(by) {
		return chat.Chat{}, ErrForbidden
	}

	if err := change(&c, by); err != nil {
		return chat.Chat{}, err
	}

	if err := uc.repo.Update(ctx, c); err != nil {
		return chat.Chat{}, err
	}

	return c, nil
}

func Rename(title string) GroupChange {
	return func(c *chat.Chat, by chat.UserID) error {
		return c.Rename(by, title)
	}
}

func AddParticipants(users []chat.UserID) GroupChange {
	return func(c *chat.Chat, by chat.UserID) error {
		return c.AddParticipants(by, users)
	}
}

func RemoveParticipant(target chat.UserID) GroupChange {
	return func(c *chat.Chat, by chat.UserID) error {
		return c.RemoveParticipant(by, target)
	}
}
