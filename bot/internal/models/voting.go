package models

import (
	"time"
)

type Voting struct {
	ID        string    `json:"id"`
	CreatorID string    `json:"creator_id"`
	Question  string    `json:"question"`
	Options   []string  `json:"options"`
	Votes     []int64   `json:"votes"`
	CreatedAt time.Time `json:"created_at"`
	IsActive  bool      `json:"is_active"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

type VoteResult struct {
	Option    string `json:"option"`
	VoteCount int    `json:"vote_count"`
}
