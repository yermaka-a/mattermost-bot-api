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
	CreateVote(vote *models.Voting) error
	GetVote(id string) (*models.Voting, error)
	UpdateVote(vote *models.Voting) error
	DeleteVote(id string) error
	ListActiveVotes() ([]*models.Voting, error)
	GracefulConnClose()
}

type TarantoolStorage struct {
	conn *tarantool.Connection
}

func NewTarantoolStorage(dialer tarantool.NetDialer) (*TarantoolStorage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	opts := tarantool.Opts{
		Timeout:     5 * time.Second,
		Concurrency: 32,
	}
	conn, err := tarantool.Connect(ctx, dialer, opts)
	if err != nil {
		return nil, err
	}
	return &TarantoolStorage{conn: conn}, nil
}

func (s *TarantoolStorage) CreateDB() (tarantool.Response, error) {
	resp, err := s.conn.Do(tarantool.NewExecuteRequest("box.schema.space.create('voting', {if_not_exists = true})")).GetResponse()
	if err != nil {
		return resp, err
	}
	resp, err = s.conn.Do(tarantool.NewExecuteRequest(`
	box.space.voting:format({
    {name = 'id', type = 'string'},
    {name = 'creator_id', type = 'string'},
    {name = 'question', type = 'string'},
    {name = 'options', type = 'array'},
    {name = 'votes', type = 'array'},
    {name = 'created_at', type = 'integer'},
    {name = 'is_active', type = 'boolean'},
    {name = 'expires_at', type = 'integer'},
})
	`)).GetResponse()
	if err != nil {
		return resp, err
	}
	resp, err = s.conn.Do(tarantool.NewExecuteRequest(
		`box.space.voting:create_index('primary', {parts = {'id'}, if_not_exists = true})`)).
		GetResponse()
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (s *TarantoolStorage) CreateVote(vote *models.Voting) error {
	fmt.Println("Create Vote")
	return nil
}

func (s *TarantoolStorage) GetVote(id string) (*models.Voting, error) {
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

func (s *TarantoolStorage) UpdateVote(vote *models.Voting) error {
	fmt.Println("UpdateVote Vote")
	return nil
}

func (s *TarantoolStorage) DeleteVote(id string) error {
	fmt.Println("DeleteVote Vote")
	return nil
}

func (s *TarantoolStorage) ListActiveVotes() ([]*models.Voting, error) {
	fmt.Println("ListActiveVotes Vote")
	return nil, nil
}

func (s *TarantoolStorage) GracefulConnClose() {
	fmt.Println("GracefulConnClose Vote")
	defer s.conn.CloseGraceful()
}
