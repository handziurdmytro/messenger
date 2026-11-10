package postgres

import (
	"fmt"
	"math"

	"github.com/handziurdmytro/messenger/user-service/internal/port/outbound"
)

func pageBounds(page outbound.Page) (limit, offset int32, err error) {
	if page.Limit <= 0 || page.Limit > math.MaxInt32 {
		return 0, 0, fmt.Errorf("invalid limit %d", page.Limit)
	}
	if page.Offset < 0 || page.Offset > math.MaxInt32 {
		return 0, 0, fmt.Errorf("invalid offset %d", page.Offset)
	}

	return int32(page.Limit), int32(page.Offset), nil
}
