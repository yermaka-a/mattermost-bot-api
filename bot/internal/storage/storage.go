package storage

import (
	"bot/internal/models"
	"context"
	"fmt"
	"time"

	"github.com/tarantool/go-tarantool/v2"
)

type VoteStorage interface {
	CreateDB() error
	CreateVote(vote *models.Vote) error
	GetVote(id string) (*models.Vote, error)
	UpdateVote(vote *models.Vote) error
	DeleteVote(id string) error
	ListActiveVotes() ([]*models.Vote, error)
	GracefulConnClose()
}

type TarantoolStorage struct {
	conn *tarantool.Connection
}

func NewTarantoolStorage(dialer tarantool.NetDialer) (*TarantoolStorage, error) {

	opts := tarantool.Opts{
		Timeout:     5 * time.Second,
		Concurrency: 32,
	}
	conn, err := tarantool.Connect(context.Background(), dialer, opts)
	if err != nil {
		return nil, err
	}
	return &TarantoolStorage{conn: conn}, nil
}

func (s *TarantoolStorage) CreateDB() error {
	_, err := s.conn.Call("box.schema.space.create", []interface{}{
		"votes",
		map[string]bool{"if_not_exists": true}})
	if err != nil {
		return err
	}
	_, err = s.conn.Call("box.space.votes:format", [][]map[string]string{
		{
			{"name": "id", "type": "string"},
			{"name": "creator_id", "type": "string"},
			{"name": "question", "type": "string"},
			{"name": "options", "type": "array"},
			{"name": "votes", "type": "table"},
			{"name": "created_at", "type": "integer"},
			{"name": "is_active", "type": "boolean"},
			{"name": "expires_at", "type": "integer"},
		}})
	if err != nil {
		return err
	}
	_, err = s.conn.Call("box.space.votes:create_index", []interface{}{
		"primary",
		map[string]interface{}{
			"parts":         []string{"id"},
			"if_not_exists": true}})
	if err != nil {
		return err
	}
	return nil
}

func (s *TarantoolStorage) CreateVote(vote *models.Vote) error {
	// _, err := s.conn.Insert("votes", []interface{}{
	// 	vote.ID,
	// 	vote.CreatorID,
	// 	vote.Question,
	// 	vote.Options,
	// 	vote.Votes,
	// 	vote.CreatedAt,
	// 	vote.IsActive,
	// 	vote.ExpiresAt,
	// })
	// return err
	fmt.Println("Create Vote")
	return nil
}

func (s *TarantoolStorage) GetVote(id string) (*models.Vote, error) {
	// resp, err := s.conn.Select("votes", "primary", 0, 1, tarantool.IterEq, []interface{}{id})
	// if err != nil {
	// 	return nil, err
	// }
	// if len(resp.Data) == 0 {
	// 	return nil, nil
	// }

	// vote := &models.Vote{}
	fmt.Println("Get Vote")
	return nil, nil
}

func (s *TarantoolStorage) UpdateVote(vote *models.Vote) error {
	fmt.Println("UpdateVote Vote")
	return nil
}

func (s *TarantoolStorage) DeleteVote(id string) error {
	fmt.Println("DeleteVote Vote")
	return nil
}

func (s *TarantoolStorage) ListActiveVotes() ([]*models.Vote, error) {
	fmt.Println("ListActiveVotes Vote")
	return nil, nil
}

func (s *TarantoolStorage) GracefulConnClose() {
	fmt.Println("GracefulConnClose Vote")
	defer s.conn.CloseGraceful()
}
