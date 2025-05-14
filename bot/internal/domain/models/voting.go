package models

import "log/slog"

type Voting struct {
	ID        string   `msgpack:"id"`
	CreatorID string   `msgpack:"creator_id"`
	Question  string   `msgpack:"question"`
	Options   []string `msgpack:"options"`
	Votes     []int64  `msgpack:"votes"`
	CreatedAt int64    `msgpack:"created_at"`
	IsActive  bool     `msgpack:"is_active"`
	ExpiresAt int64    `msgpack:"expires_at,omitempty"`
}

func (v *Voting) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", v.ID),
		slog.String("creator_id", v.CreatorID),
		slog.Int64("created_at", v.CreatedAt),
		slog.Int64("expires_at", v.ExpiresAt),
		slog.Bool("is_active", v.IsActive),
	)
}
