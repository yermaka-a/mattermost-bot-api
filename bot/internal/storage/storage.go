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
	GracefulConnClose()
}

type Storage struct {
	ts *models.TarantoolStorage
}

func NewStorage(dialer tarantool.NetDialer) (*Storage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	opts := tarantool.Opts{
		Timeout:       5 * time.Second,
		Reconnect:     time.Duration(time.Second * 5),
		MaxReconnects: 5,
		Concurrency:   32,
	}
	conn, err := tarantool.Connect(ctx, dialer, opts)
	if err != nil {
		return nil, err
	}
	return &Storage{
		ts: &models.TarantoolStorage{Conn: conn}}, nil
}

func (s *Storage) CreateDB() ([]interface{}, error) {

	// Создание спейса для пользователей
	resp, err := s.ts.Conn.Do(tarantool.NewEvalRequest(`
        box.schema.space.create('users', {if_not_exists = true})
    `)).Get()

	if err != nil {
		return resp, err
	}

	// Установка формата пользователей
	resp, err = s.ts.Conn.Do(tarantool.NewEvalRequest(`
	box.space.users:format({
		{name = 'id', type = 'string'},
		{name = 'votes', type = 'map'}
	})
	`)).Get()

	if err != nil {
		return resp, err
	}

	resp, err = s.ts.Conn.Do(tarantool.NewEvalRequest(`
		box.space.users:create_index('primary', {
			type = 'hash',
			parts = {{field = 'id', type = 'string'}},
			unique = true,
			if_not_exists = true
		})
	`)).Get()
	if err != nil {
		return resp, err
	}

	// Создание спейса для голосования
	resp, err = s.ts.Conn.Do(tarantool.NewEvalRequest(`
        box.schema.space.create('voting', {if_not_exists = true})
    `)).Get()
	if err != nil {
		return resp, err
	}

	// Установка формата голосования
	resp, err = s.ts.Conn.Do(tarantool.NewEvalRequest(`
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

	// Создание индекса голосования
	resp, err = s.ts.Conn.Do(tarantool.NewEvalRequest(`
        box.space.voting:create_index('primary', {
           	type = 'hash',
			parts = {{field = 'id', type = 'string'}},
			unique = true,
            if_not_exists = true
        })
    `)).Get()
	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (s *Storage) CreateVote(voting *models.Voting) error {
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
	_, err := s.ts.Conn.Do(tarantool.NewInsertRequest("voting").Tuple(values)).Get()
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) GetVote(id string) (*models.Voting, error) {
	var result []models.Voting
	err := s.ts.Conn.Do(tarantool.NewEvalRequest(`
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

func (s *Storage) UpdateVote(voting *models.Voting) error {
	_, err := s.ts.Conn.Do(tarantool.NewCallRequest("box.space.voting:update").
		Args([]interface{}{
			voting.ID,
			[][]interface{}{
				{"=", 5, voting.Votes},
				{"=", 7, voting.IsActive},
			},
		})).
		Get()

	return err

}

func (s *Storage) DeleteVote(id string) error {
	_, err := s.ts.Conn.Do(tarantool.NewCallRequest("box.space.voting:delete").Args([]interface{}{
		id,
	})).Get()
	return err
}

func (s *Storage) CreateUser(user *models.User) error {
	values := []interface{}{
		user.UserId,
		user.Votes,
	}
	_, err := s.ts.Conn.Do(tarantool.NewInsertRequest("users").Tuple(values)).Get()
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) UpdateUser(user *models.User) error {
	_, err := s.ts.Conn.Do(tarantool.NewCallRequest("box.space.users:update").
		Args([]interface{}{
			user.UserId,
			[][]interface{}{
				{"=", 2, user.Votes},
			},
		})).
		Get()

	return err
}

func (s *Storage) GetUserById(userId string) (*models.User, error) {
	var result []models.User
	err := s.ts.Conn.Do(tarantool.NewEvalRequest(`
	return box.space.users:get(...)
`).Args([]interface{}{userId})).GetTyped(&result)
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	return &result[0], nil
}

func (s *Storage) GracefulConnClose() {
	s.ts.Conn.CloseGraceful()
}
