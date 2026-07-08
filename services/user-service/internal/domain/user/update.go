package user

import "time"

type UpdateInput struct {
	DisplayName *string
	AvatarURL   *string
}

func (u *User) Apply(input UpdateInput) error {
	if input.DisplayName != nil {
		if *input.DisplayName == "" {
			return ErrDisplayNameEmpty
		}
		u.DisplayName = *input.DisplayName
	}

	if input.AvatarURL != nil {
		u.AvatarURL = *input.AvatarURL
	}

	u.UpdatedAt = time.Now().UTC()
	return nil
}
