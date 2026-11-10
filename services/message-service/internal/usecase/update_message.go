package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
	contract "github.com/handziurdmytro/messenger/pkg/contracts/message"
	"github.com/handziurdmytro/messenger/pkg/errkit"
)

type EditMessageInput struct {
	ChatID    message.ChatID
	MessageID message.ID
	UserID    string
	Payload   json.RawMessage
}

type EditMessage struct {
	repo      MessageRepository
	publisher MessagePublisher
}

func NewEditMessage(repo MessageRepository, publisher MessagePublisher) *EditMessage {
	return &EditMessage{repo: repo, publisher: publisher}
}

func (uc *EditMessage) Execute(ctx context.Context, in EditMessageInput) (message.Message, error) {
	msg, err := findOwnMessage(ctx, uc.repo, in.ChatID, in.MessageID, in.UserID)
	if err != nil {
		return message.Message{}, err
	}

	if err := msg.Edit(in.Payload, time.Now().UTC()); err != nil {
		return message.Message{}, err
	}

	if err := uc.repo.Update(ctx, msg); err != nil {
		return message.Message{}, errkit.Wrap(ErrPersistence, "update message", err)
	}

	event := contract.EditedEvent{
		MessageID: string(msg.ID),
		ChatID:    string(msg.ChatID),
		Type:      contract.Type(msg.Type),
		Payload:   msg.Payload,
		EditedAt:  *msg.EditedAt,
	}
	if err := uc.publisher.PublishEdited(ctx, event); err != nil {
		return message.Message{}, errkit.Wrap(ErrPublishFailed, "publish message edited", err)
	}

	return msg, nil
}

type DeleteMessage struct {
	repo      MessageRepository
	publisher MessagePublisher
}

func NewDeleteMessage(repo MessageRepository, publisher MessagePublisher) *DeleteMessage {
	return &DeleteMessage{repo: repo, publisher: publisher}
}

func (uc *DeleteMessage) Execute(ctx context.Context, chatID message.ChatID, messageID message.ID, userID string) error {
	msg, err := findOwnMessage(ctx, uc.repo, chatID, messageID, userID)
	if err != nil {
		return err
	}

	if err := msg.Delete(time.Now().UTC()); err != nil {
		return err
	}

	if err := uc.repo.Update(ctx, msg); err != nil {
		return errkit.Wrap(ErrPersistence, "update message", err)
	}

	event := contract.DeletedEvent{
		MessageID: string(msg.ID),
		ChatID:    string(msg.ChatID),
		DeletedAt: *msg.DeletedAt,
	}
	if err := uc.publisher.PublishDeleted(ctx, event); err != nil {
		return errkit.Wrap(ErrPublishFailed, "publish message deleted", err)
	}

	return nil
}

func findOwnMessage(ctx context.Context, repo MessageRepository, chatID message.ChatID, messageID message.ID, userID string) (message.Message, error) {
	if _, err := uuid.Parse(string(chatID)); err != nil {
		return message.Message{}, ErrInvalidChatID
	}
	if _, err := uuid.Parse(string(messageID)); err != nil {
		return message.Message{}, ErrInvalidMessageID
	}

	msg, err := repo.FindByID(ctx, chatID, messageID)
	if err != nil {
		return message.Message{}, err
	}

	if msg.FromUser != userID {
		return message.Message{}, ErrForbidden
	}

	return msg, nil
}
