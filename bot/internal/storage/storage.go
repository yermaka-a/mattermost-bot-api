package storage

import (
	"bot/internal/config"
	"bot/internal/domain/models"
	"context"
	"fmt"
	"time"

	"github.com/tarantool/go-tarantool/v2"
)

type VoteStorage interface {
	CreateVote(vote *models.Voting) error
	GetVote(id string) (*models.Voting, error)
	UpdateVote(vote *models.Voting) error
	DeleteVote(id string) error
	CreateUser(user *models.User) error
	GetUserById(userId string) (*models.User, error)
	UpdateUser(user *models.User) error
	GracefulConnClose() error
}

type tarantoolStorage struct {
	Conn *tarantool.Connection
}

type storage struct {
	ts *tarantoolStorage
}

func NewStorage(config config.Config) (VoteStorage, error) {
	op := "storage.NewStorage"
	// Подключение к Tarantool
	dialer := tarantool.NetDialer{Address: config.TARANTOOL_ADDR, User: config.TARANTOOL_USER, Password: config.TARANTOOL_PASS}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	opts := tarantool.Opts{
		Timeout:       5 * time.Second,
		Reconnect:     time.Duration(time.Second * 5),
		MaxReconnects: 5,
	}
	conn, err := tarantool.Connect(ctx, dialer, opts)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	storage := &storage{
		ts: &tarantoolStorage{Conn: conn}}

	return storage, nil
}

func (s *storage) CreateVote(voting *models.Voting) error {
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

func (s *storage) GetVote(id string) (*models.Voting, error) {
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

func (s *storage) UpdateVote(voting *models.Voting) error {
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

func (s *storage) DeleteVote(id string) error {
	_, err := s.ts.Conn.Do(tarantool.NewCallRequest("box.space.voting:delete").Args([]interface{}{
		id,
	})).Get()
	return err
}

func (s *storage) CreateUser(user *models.User) error {
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

func (s *storage) UpdateUser(user *models.User) error {
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

func (s *storage) GetUserById(userId string) (*models.User, error) {
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

func (s *storage) GracefulConnClose() error {
	err := s.ts.Conn.CloseGraceful()
	if err != nil {
		return err
	}
	return nil
}
