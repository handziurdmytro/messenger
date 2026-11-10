package block

import "errors"

var (
	ErrUserIDEmpty     = errors.New("user id is empty")
	ErrBlockedIDEmpty  = errors.New("blocked user id is empty")
	ErrCannotBlockSelf = errors.New("cannot block yourself")
	ErrCreatedAtZero   = errors.New("created at is zero")
	ErrAlreadyBlocked  = errors.New("user is already blocked")
	ErrNotFound        = errors.New("block not found")
)
