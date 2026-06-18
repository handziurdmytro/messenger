package auth

import "time"

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type Identity struct {
	UserID    string
	Email     string
	Roles     []Role
	ExpiresAt time.Time
}

func (i Identity) HasRole(role Role) bool {
	for _, r := range i.Roles {
		if r == role {
			return true
		}
	}
	return false
}
