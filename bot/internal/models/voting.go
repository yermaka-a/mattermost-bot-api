package models

type Voting struct {
	ID        string   `msgpack:"id"`
	CreatorID string   `msgpack:"creator_id"`
	Question  string   `msgpack:"question"`
	Options   []string `msgpack:"options"`
	Votes     []int64  `msgpack:"votes"`
	CreatedAt int64    `msgpack:"created_at"`
	IsActive  bool     `msgpack:"is_active"`
	ExpiresAt int64    `msgpack:"expires_at,omitempty"`
	// WhoVoted  []string `msgpack:"who_voted"`
}

type VoteResult struct {
	Option    string `json:"option"`
	VoteCount int    `json:"vote_count"`
}
