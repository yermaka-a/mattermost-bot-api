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

func (s *TarantoolStorage) CreateDB() ([]interface{}, error) {

	// Создание спейса
	resp, err := s.conn.Do(tarantool.NewEvalRequest(`
        box.schema.space.create('voting', {if_not_exists = true})
    `)).Get()
	if err != nil {
		return resp, err
	}

	// Установка формата
	resp, err = s.conn.Do(tarantool.NewEvalRequest(`
        box.space.voting:format({
            {name = 'id', type = 'string'},
            {name = 'creator_id', type = 'string'},
            {name = 'question', type = 'string'},
            {name = 'options', type = 'array'},
            {name = 'votes', type = 'array'},
            {name = 'created_at', type = 'integer'},
            {name = 'is_active', type = 'boolean'},
            {name = 'expires_at', type = 'integer'}
        })
    `)).Get()
	if err != nil {
		return resp, err
	}

	// Создание индекса
	resp, err = s.conn.Do(tarantool.NewEvalRequest(`
        box.space.voting:create_index('primary', {
            parts = {{field = 'id', type = 'string'}},
            if_not_exists = true
        })
    `)).Get()
	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (s *TarantoolStorage) CreateVote(voting *models.Voting) error {
	values := []interface{}{
		voting.ID,
		voting.CreatorID,
		voting.Question,
		voting.Options,
		voting.Votes,
		voting.CreatedAt,
		voting.IsActive,
		voting.ExpiresAt,
	}
	_, err := s.conn.Do(tarantool.NewInsertRequest("voting").Tuple(values)).Get()
	if err != nil {
		return err
	}
	return nil
}

func (s *TarantoolStorage) GetVote(id string) (*models.Voting, error) {
	var result []models.Voting
	err := s.conn.Do(tarantool.NewEvalRequest(`
	return box.space.voting:get(...)
`).Args([]interface{}{id})).GetTyped(&result)
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("voting not found")
	}

	return &result[0], nil
}

func (s *TarantoolStorage) UpdateVote(voting *models.Voting) error {
	_, err := s.conn.Do(tarantool.NewCallRequest("box.space.voting:update").
		Args([]interface{}{
			voting.ID,
			[][]interface{}{
				{"=", 5, voting.Votes},
			},
		})).
		Get()

	return err

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
	s.conn.CloseGraceful()
}
