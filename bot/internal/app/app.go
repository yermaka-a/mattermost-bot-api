package app

import (
	"bot/internal/config"
	"bot/internal/logger"
	"bot/internal/models"
	"bot/internal/storage"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
)

type App struct {
	client  *model.Client4
	storage *storage.TarantoolStorage
	log     *logger.Logger
}

var app App

type Message struct {
	Id        string `json:"id"`
	UserId    string `json:"user_id"`
	ChannelId string `json:"channel_id"`
	Message   string `json:"message"`
}

type MattermostRoutes struct {
	me string
}

var routes MattermostRoutes

type Bot struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

const (
	CREATE_VOTE = "/create"
	VOTE        = "/vote"
	RESULTS     = "/results"
	FINISHED    = "/finished"
	DELETE      = "/delete"
)

func Start(ctx context.Context, client *model.Client4, storage *storage.TarantoolStorage, cfg *config.BotConfig, log *logger.Logger) {

	resp, err := storage.CreateDB()
	if err != nil {
		log.Fatalln("Can't create database", err)
	}
	log.Infoln(resp)
	routes = MattermostRoutes{
		me: "/users/me",
	}
	app = App{
		client,
		storage,
		log,
	}
	client.HTTPHeader = map[string]string{
		"Authorization": "Bearer " + cfg.BOT_TOKEN,
	}
	var bot Bot
	// установить user_id и username для бота Bot
	app.getBotData(&bot)

	wsClient, err := model.NewWebSocketClient4(cfg.MATTERMOST_WS, client.AuthToken)
	if err != nil {
		log.Fatalln("ws connection error", err)
	}
	wsClient.Listen()
	go func() {
	LOOP:
		for {
			select {
			case event := <-wsClient.EventChannel:
				if event.EventType() == model.WebsocketEventPosted {

					postStr, ok := event.GetData()["post"].(string)
					if ok {
						var post map[string]interface{}
						json.Unmarshal([]byte(postStr), &post)
						msg := Message{
							Id:        post["id"].(string),
							UserId:    post["user_id"].(string),
							ChannelId: post["channel_id"].(string),
							// Удаляем лишние пробелы из строки
							Message: strings.Join(strings.Fields(strings.TrimSpace(post["message"].(string))), " "),
						}
						if bot.ID != msg.UserId {
							typeMsg, message := getCommand(msg.Message)
							msg.Message = message
							switch typeMsg {
							case CREATE_VOTE:
								app.CreateVote(msg)

							case VOTE:
								app.ToVote(msg)

							case RESULTS:
								app.GetVoteResults(msg)

							case FINISHED:
								app.FinishingVote(msg)

							case DELETE:
								app.DeletingVote(msg)

							}
						}
					}
				}
				// Начало беседы с ботом в ЛС
				if event.EventType() == model.WebsocketEventDirectAdded {
					creatorId := event.GetData()["creator_id"].(string)
					teammateId := event.GetData()["teammate_id"].(string)
					if creatorId != "" && teammateId != "" {
						if teammateId == bot.ID {
							chl, _, err := client.CreateDirectChannel(creatorId, teammateId)
							if err != nil {
								log.Errorln(err)
							}
							client.CreatePost(&model.Post{
								ChannelId: chl.Id,
								Message: fmt.Sprintf("Добрый день я бот для создания опросов, вы можете взаимодействовать со мной следующим образом:\n%s\n%s\n%s\n%s\n%s",
									"**"+CREATE_VOTE+" \"Ваш вопрос\" \"Вариант 1\" \"Вариант 2\"** - пример как создать опрос",
									"**"+VOTE+" ID_голосования вариант_ответа** - пример как проголосовать в опросе",
									"**"+RESULTS+" Id_голосования** - пример как получить результаты",
									"**"+FINISHED+" Id_голосования** - пример как завершить опрос досрочно (Доступо только создателю опроса)",
									"**"+DELETE+" Id_голосования** - пример как удалить опрос (Доступно только создателю опроса)",
								)})
						}

					}
				}
			case <-ctx.Done():
				break LOOP
			}
		}
	}()

}

func (a *App) getBotData(bot *Bot) {
	res, err := a.client.DoAPIRequestReader("GET", a.client.APIURL+routes.me, nil, a.client.HTTPHeader)

	if err != nil {
		log.Fatalln("Can't GET data from", routes.me, err)
	}

	data, _ := io.ReadAll(res.Body)

	err = json.Unmarshal(data, &bot)
	if err != nil {
		log.Fatalln("Can't unmarshal bot data", err)
	}
}

func getCommand(message string) (string, string) {
	commands := []string{CREATE_VOTE, VOTE, FINISHED, RESULTS, DELETE}
	// Создаем регулярное выражение для поиска команды в начале строки
	commandPattern := "^(" + strings.Join(commands, "|") + ")"
	re := regexp.MustCompile(commandPattern)

	// Ищем команду в строке
	match := re.FindString(message)
	if match != "" {
		// Удаляем команду из строки
		remaining := strings.TrimSpace(strings.TrimPrefix(message, match))
		return match, remaining
	}
	return "", ""
}

func (a *App) CreateVote(msg Message) {
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

		err := a.storage.CreateVote(voting)
		if err != nil {
			a.log.Errorln("Can't create a voting:", voting, "error", err)
			a.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				Message:   "Упс! Произошла ошибка при создании голосования"})
			return
		}
		a.client.CreatePost(&model.Post{
			ChannelId: msg.ChannelId,
			Message: fmt.Sprintf("**Ваше голосование успешно создано!**\nID голосования:**%s**\nТип вопроса:%s\nВарианты ответов:\n%s",
				voting.ID,
				voting.Question,
				concateOptions(voting.Options),
			)})
	} else {

		a.client.CreatePost(&model.Post{
			ChannelId: msg.ChannelId,
			Message:   "Опрос не был создан, возможно данные введены некорректно..."})
	}
}

func concateVotes(votes []int64) string {
	builder := &strings.Builder{}
	for i, v := range votes {
		builder.WriteString(fmt.Sprintf("%d. %d;\t", i+1, v))
	}
	return builder.String()
}
func concateOptions(options []string) string {
	builder := &strings.Builder{}
	for i, v := range options {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, v))
	}
	return builder.String()
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

func (a *App) ToVote(msg Message) {
	parts := parseVote(msg.Message)

	//Get voting by ID
	if parts == nil {
		a.client.CreatePost(&model.Post{
			ChannelId: msg.ChannelId,
			Message:   "Некорректный запрос"})
		return
	}
	voting, err := a.storage.GetVote(parts[0])
	if err != nil {
		a.client.CreatePost(&model.Post{
			ChannelId: msg.ChannelId,
			Message:   "Такого голосования не нашлось."})
		return
	}
	if voting.IsActive {
		now := time.Now()
		if now.Before(time.Unix(voting.ExpiresAt, 0)) {
			user, err := a.storage.GetUserById(msg.UserId)
			if err != nil {
				a.log.Errorln(err)
				a.client.CreatePost(&model.Post{
					ChannelId: msg.ChannelId,
					Message:   "Что-то пошло не так..."})
				return
			}
			if user == nil {
				user = &models.User{
					Votes: map[string]int64{},
				}
				a.storage.CreateUser(user)
			}

			votes, err := updateVote(voting.Votes, parts[1])
			if err != nil {
				a.log.Info(voting.ID, voting.CreatorID, err)
				a.client.CreatePost(&model.Post{
					ChannelId: msg.ChannelId,
					Message:   "Вариант ответа некорректный!"})
				return

			}
			option, err := strconv.Atoi(parts[1])
			if err != nil {
				a.client.CreatePost(&model.Post{
					ChannelId: msg.ChannelId,
					Message:   "Что-то пошло не так..."})
				return
			}
			// проголосовал ли пользователь уже в этом опросе
			isVoted := false
			if user.Votes[parts[0]] != 0 {
				votes[user.Votes[parts[0]]-1] = votes[user.Votes[parts[0]]-1] - 1
				isVoted = true
			}
			user.Votes[parts[0]] = int64(option)
			voting.Votes = votes
			err = a.storage.UpdateUser(user)
			if err != nil {
				a.client.CreatePost(&model.Post{
					ChannelId: msg.ChannelId,
					Message:   "Что-то пошло не так..."})
				return
			}
			err = a.storage.UpdateVote(voting)
			if err != nil {
				a.log.Errorln(err)
				a.client.CreatePost(&model.Post{
					ChannelId: msg.ChannelId,
					Message:   "Что-то пошло не так..."})
				return
			}
			if isVoted {
				a.client.CreatePost(&model.Post{
					ChannelId: msg.ChannelId,
					Message:   fmt.Sprintf("Ваш голос перезаписан!\nГолосование:%s\nВопрос:%s\nВарианты:\n%s\nГолоса:\n%s", voting.ID, voting.Question, concateOptions(voting.Options), concateVotes(voting.Votes))})

			} else {
				a.client.CreatePost(&model.Post{
					ChannelId: msg.ChannelId,

					Message: fmt.Sprintf("Ваш голос учтён!\nГолосование:%s\nВопрос:%s\nВарианты:\n%s\nГолоса:\n%s", voting.ID, voting.Question, concateOptions(voting.Options), concateVotes(voting.Votes))})
			}
		} else {
			if voting.IsActive {
				voting.IsActive = false
			}
			err := a.storage.UpdateVote(voting)
			if err != nil {
				a.log.Errorln("func ToVote", "can't update vote", err)
			}
			a.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				Message:   fmt.Sprintf("Время голосования истекло %s", time.Unix(voting.ExpiresAt, 0).Format("2006-01-02 15:04:05"))})

		}
	} else {
		a.client.CreatePost(&model.Post{
			ChannelId: msg.ChannelId,
			Message:   fmt.Sprintf("Время голосования истекло %s", time.Unix(voting.ExpiresAt, 0).Format("2006-01-02 15:04:05"))})

	}

}

func updateVote(votes []int64, vote string) ([]int64, error) {
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

// Получаем id = parts[0] номер варианта для голосования num = parts[1]
func parseVote(msg string) []string {
	parts := strings.Split(msg, " ")

	if len(parts) != 2 {
		return nil
	}

	id := parts[0]
	if !isValidID(id) {
		return nil
	}
	return parts
}

func isValidID(id string) bool {

	re := regexp.MustCompile("^[a-zA-Z0-9]{20,30}$")
	return re.MatchString(id)
}

func (a *App) GetVoteResults(msg Message) {

	isVoteId := isValidID(msg.Message)
	//Get voting by ID
	if isVoteId {
		voting, err := a.storage.GetVote(msg.Message)
		if err != nil {
			a.log.Infoln("voting not found", err)
			a.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				Message:   "Такого голосования не нашлось."})
			return
		}
		a.client.CreatePost(&model.Post{
			ChannelId: msg.ChannelId,
			Message:   fmt.Sprintf("Голосование:%s\nВопрос:%s\nВарианты:\n%s\nГолоса:\n%s", voting.ID, voting.Question, concateOptions(voting.Options), concateVotes(voting.Votes))})
	} else {
		a.client.CreatePost(&model.Post{
			ChannelId: msg.ChannelId,
			Message:   "Некорректный ID опроса"})
	}
}

func (a *App) DeletingVote(msg Message) {
	isVoteId := isValidID(msg.Message)
	//Get voting by ID
	if isVoteId {
		voting, err := a.storage.GetVote(msg.Message)
		if err != nil {
			a.log.Infoln("voting not found", err)
			a.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				Message:   "Такого голосования не нашлось."})
			return
		}
		if voting.CreatorID == msg.UserId {
			err := a.storage.DeleteVote(voting.ID)
			if err != nil {
				a.log.Errorln("can't delete voting", err)
				a.client.CreatePost(&model.Post{
					ChannelId: msg.ChannelId,
					Message:   "При удалении произошла ошибка!"})
				return
			}
			a.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				Message:   fmt.Sprintf("Голосование с ID:%s успешно удалено!", voting.ID)})
		} else {
			a.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				Message:   fmt.Sprintf("У вас нет прав на удаление опроса с ID:%s", voting.ID)})
		}

	} else {
		a.client.CreatePost(&model.Post{
			ChannelId: msg.ChannelId,
			Message:   "Некорректный ID опроса"})
	}
}

func (a *App) FinishingVote(msg Message) {
	isVoteId := isValidID(msg.Message)
	//Get voting by ID
	if isVoteId {
		voting, err := a.storage.GetVote(msg.Message)
		if err != nil {
			a.log.Infoln("voting not found", err)
			a.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				Message:   "Такого голосования не нашлось."})
			return
		}
		if !voting.IsActive {
			a.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				Message:   "Голосование уже завершено!"})
			return
		}

		if voting.CreatorID == msg.UserId {
			voting.IsActive = false
			err := a.storage.UpdateVote(voting)
			if err != nil {
				a.log.Errorln("can't finished voting", err)
				a.client.CreatePost(&model.Post{
					ChannelId: msg.ChannelId,
					Message:   "Произошла ошибка!"})
				return
			}
			a.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				Message:   fmt.Sprintf("Голосование с ID:%s завершено досрочно!", voting.ID)})
		} else {
			a.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				Message:   "У вас нет прав на завершение этого голосования досрочно."})
		}

	} else {
		a.client.CreatePost(&model.Post{
			ChannelId: msg.ChannelId,
			Message:   "Некорректный ID"})
	}
}
