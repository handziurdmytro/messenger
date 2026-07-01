package message

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrMessageEmpty  = errors.New("message body is empty")
	ErrRoomIDEmpty   = errors.New("room id is empty")
	ErrFromUserEmpty = errors.New("from user is empty")
)

type ID string
type RoomID string

type Message struct {
	ID        ID
	RoomID    RoomID
	FromUser  string
	Type      string
	Body      string
	CreatedAt time.Time
}

func New(roomID RoomID, fromUser, body, msgType string) (Message, error) {
	if roomID == "" {
		return Message{}, ErrRoomIDEmpty
	}
	if fromUser == "" {
		return Message{}, ErrFromUserEmpty
	}
	if body == "" {
		return Message{}, ErrMessageEmpty
	}

	return Message{
		ID:        ID(uuid.New().String()),
		RoomID:    roomID,
		FromUser:  fromUser,
		Type:      msgType,
		Body:      body,
		CreatedAt: time.Now().UTC(),
	}, nil
}
