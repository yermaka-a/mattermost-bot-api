package botService

import (
	"bot/internal/domain/models"
	"bot/internal/storage"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
)

type BotService interface {
	CreateVote(msg *models.Message) (*models.Voting, error)
	ParseVote(msg string) []string
	GetVoteLocally(parts []string) (*models.Voting, error)
	GetUserById(userID string) (*models.User, error)
	CreateUser(userID string) *models.User
	UpdateUser(user *models.User) error
	UpdateVote(voting *models.Voting) error
	UpdateVoteLocaly(votes []int64, vote string) ([]int64, error)
	IsValidID(id string) bool
	GetVote(msg string) (*models.Voting, error)
	DeleteVote(id string) error
	Close() error
}

type botService struct {
	log     *slog.Logger
	storage storage.VoteStorage
}

func New(log *slog.Logger, storage storage.VoteStorage) BotService {
	return &botService{
		log:     log,
		storage: storage,
	}
}

func (b *botService) CreateVote(msg *models.Message) (*models.Voting, error) {
	op := "bot.CreateVote"
	question, options := parseQuestionAndOptions(msg.Message)
	if question != "" && options != nil {
		voting := &models.Voting{
			ID:        model.NewId(),
			CreatorID: msg.UserId,
			Question:  question,
			Options:   options,
			Votes:     make([]int64, len(options)),
			CreatedAt: time.Now().Unix(),
			IsActive:  true,
			ExpiresAt: time.Now().Add(time.Hour * 24).Unix(),
		}

		err := b.storage.CreateVote(voting)

		return voting, err
	}
	b.log.Error("incorrect parameters", slog.Any("msg", msg))
	return nil, fmt.Errorf("%s: %w", op, errors.New("incorrect parameters"))
}

func parseQuestionAndOptions(msg string) (string, []string) {
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindAllStringSubmatch(msg, -1)

	if len(matches) > 0 {

		question := matches[0][1]

		options := make([]string, len(matches)-1)
		for i := 1; i < len(matches); i++ {
			options[i-1] = matches[i][1]
		}

		return question, options
	} else {
		return "", nil
	}
}

func (b *botService) GetVoteLocally(parts []string) (*models.Voting, error) {
	voting, err := b.storage.GetVote(parts[0])
	return voting, err
}

// Получаем id = parts[0] номер варианта для голосования num = parts[1]
func (b *botService) ParseVote(msg string) []string {
	parts := strings.Split(msg, " ")

	if len(parts) != 2 {
		return nil
	}

	id := parts[0]
	if !b.IsValidID(id) {
		return nil
	}
	return parts
}

func (b *botService) IsValidID(id string) bool {

	re := regexp.MustCompile("^[a-zA-Z0-9]{20,30}$")
	return re.MatchString(id)
}

func (b *botService) GetUserById(userID string) (*models.User, error) {
	user, err := b.storage.GetUserById(userID)
	return user, err
}

func (b *botService) CreateUser(userID string) *models.User {

	user := &models.User{
		UserId: userID,
		Votes:  map[string]int64{},
	}
	b.storage.CreateUser(user)

	return user

}

func (b *botService) UpdateVoteLocaly(votes []int64, vote string) ([]int64, error) {
	number, err := strconv.Atoi(vote)
	if err != nil {
		return nil, err
	}
	for i := range votes {
		if i+1 == number {
			votes[i] += 1
			return votes, nil
		}
	}
	return nil, fmt.Errorf("such vote isnt found")
}

func (b *botService) UpdateUser(user *models.User) error {
	err := b.storage.UpdateUser(user)
	return err
}

func (b *botService) UpdateVote(voting *models.Voting) error {
	err := b.storage.UpdateVote(voting)
	return err
}

func (b *botService) GetVote(msg string) (*models.Voting, error) {
	voting, err := b.storage.GetVote(msg)
	return voting, err
}

func (b *botService) DeleteVote(id string) error {
	err := b.storage.DeleteVote(id)
	return err
}

func (b *botService) Close() error {
	err := b.storage.GracefulConnClose()
	return err
}
