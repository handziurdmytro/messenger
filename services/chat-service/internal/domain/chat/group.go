package chat

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxGroupParticipants = 100
	MaxTitleLength       = 128
)

func NewGroup(owner UserID, title string, members []UserID) (Chat, error) {
	if _, err := ParseUserID(string(owner)); err != nil {
		return Chat{}, err
	}

	title, err := normalizeTitle(title)
	if err != nil {
		return Chat{}, err
	}

	participants := []UserID{owner}
	for _, member := range members {
		if _, err := ParseUserID(string(member)); err != nil {
			return Chat{}, err
		}
		if !slices.Contains(participants, member) {
			participants = append(participants, member)
		}
	}

	if len(participants) < 2 {
		return Chat{}, ErrInvalidParticipants
	}
	if len(participants) > MaxGroupParticipants {
		return Chat{}, ErrTooManyParticipants
	}

	return Chat{
		id:           NewID(),
		chatType:     TypeGroup,
		title:        title,
		owner:        owner,
		participants: participants,
		createdAt:    time.Now().UTC(),
	}, nil
}

func RestoreGroup(id ID, title string, owner UserID, participants []UserID, createdAt time.Time) (Chat, error) {
	if _, err := ParseID(string(id)); err != nil {
		return Chat{}, err
	}
	if createdAt.IsZero() {
		return Chat{}, ErrInvalidCreatedAt
	}

	title, err := normalizeTitle(title)
	if err != nil {
		return Chat{}, err
	}

	for i, participant := range participants {
		if _, err := ParseUserID(string(participant)); err != nil {
			return Chat{}, err
		}
		if slices.Contains(participants[:i], participant) {
			return Chat{}, ErrInvalidParticipants
		}
	}

	if len(participants) == 0 || !slices.Contains(participants, owner) {
		return Chat{}, ErrInvalidParticipants
	}

	return Chat{
		id:           id,
		chatType:     TypeGroup,
		title:        title,
		owner:        owner,
		participants: slices.Clone(participants),
		createdAt:    createdAt.UTC(),
	}, nil
}

func (c *Chat) Rename(by UserID, title string) error {
	if err := c.requireOwner(by); err != nil {
		return err
	}

	title, err := normalizeTitle(title)
	if err != nil {
		return err
	}

	c.title = title
	return nil
}

func (c *Chat) AddParticipants(by UserID, users []UserID) error {
	if err := c.requireOwner(by); err != nil {
		return err
	}

	added := 0
	for _, u := range users {
		if _, err := ParseUserID(string(u)); err != nil {
			return err
		}
		if slices.Contains(c.participants, u) {
			continue
		}
		if len(c.participants) >= MaxGroupParticipants {
			return ErrTooManyParticipants
		}
		c.participants = append(c.participants, u)
		added++
	}

	if added == 0 {
		return ErrAlreadyParticipant
	}
	return nil
}

func (c *Chat) RemoveParticipant(by, target UserID) error {
	if err := c.requireOwner(by); err != nil {
		return err
	}
	if target == c.owner {
		return ErrCannotRemoveOwner
	}
	if !c.HasParticipant(target) {
		return ErrNotParticipant
	}

	c.participants = slices.DeleteFunc(c.participants, func(u UserID) bool { return u == target })
	return nil
}

func (c *Chat) Leave(user UserID) error {
	if c.chatType != TypeGroup {
		return ErrNotGroup
	}
	if !c.HasParticipant(user) {
		return ErrNotParticipant
	}

	c.participants = slices.DeleteFunc(c.participants, func(u UserID) bool { return u == user })
	if user == c.owner && len(c.participants) > 0 {
		c.owner = c.participants[0]
	}
	return nil
}

func (c Chat) IsEmpty() bool {
	return len(c.participants) == 0
}

func (c Chat) requireOwner(by UserID) error {
	if c.chatType != TypeGroup {
		return ErrNotGroup
	}
	if !c.HasParticipant(by) {
		return ErrNotParticipant
	}
	if by != c.owner {
		return ErrNotOwner
	}
	return nil
}

func normalizeTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" || utf8.RuneCountInString(title) > MaxTitleLength {
		return "", ErrInvalidTitle
	}
	return title, nil
}
