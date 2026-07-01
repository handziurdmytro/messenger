package message

import "time"

type Query struct {
	RoomID RoomID
	Before time.Time
	Limit  int
}

func (q Query) WithDefaults() Query {
	if q.Limit == 0 || q.Limit > 100 {
		q.Limit = 50
	}
	if q.Before.IsZero() {
		q.Before = time.Now().UTC()
	}
	return q
}
