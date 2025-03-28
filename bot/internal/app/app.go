package app

import (
	"bot/internal/config"
	"bot/internal/logger"
	"bot/internal/models"
	"bot/internal/storage"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"regexp"
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
	VOTE        = "/id "
	RESULTS     = "/results "
	FINISHED    = "/finished "
	DELETE      = "/delete "
)

func Start(client *model.Client4, storage *storage.TarantoolStorage, cfg *config.BotConfig, log *logger.Logger) {
	_, err := storage.CreateDB()
	if err != nil {
		log.Fatalln("Can't create database", err)
	}
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
	defer wsClient.Conn.Close()
	for {
		select {
		case event := <-wsClient.EventChannel:
			if event.EventType() == model.WebsocketEventPosted {

				postStr, ok := event.GetData()["post"].(string)
				fmt.Println(event.GetData())
				if ok {
					var post map[string]interface{}
					json.Unmarshal([]byte(postStr), &post)
					msg := Message{
						Id:        post["id"].(string),
						UserId:    post["user_id"].(string),
						ChannelId: post["channel_id"].(string),
						Message:   post["message"].(string),
					}
					//client.CreatePost(&model.Post{RootId: msg.Id, ChannelId: msg.ChannelId, Message: message})
					if bot.ID != msg.UserId {
						typeMsg, message := getCommand(msg.Message)
						msg.Message = message
						switch typeMsg {
						case CREATE_VOTE:
							fmt.Println("CREATE_VOTE")
							app.CreateVote(msg)

						case VOTE:
							fmt.Println("VOTE")
							app.ToVote(msg)

						case RESULTS:
							fmt.Println("RESULTS")
							app.GetVoteResults(msg)

						case FINISHED:
							fmt.Println("FINISHED")
							app.FinishingVote(msg)

						case DELETE:
							fmt.Println("DELETE")
							app.DeletingVote(msg)

						}
					}
				}
			}
		default:
			time.Sleep(1 * time.Second)
		}
	}

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
			CreatedAt: time.Now(),
			IsActive:  true,
			ExpiresAt: time.Now().Add(time.Hour * 24),
		}
		err := a.storage.CreateVote(voting)
		if err != nil {
			a.log.Errorln("Can't create a voting:", voting, "error", err)
			a.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				RootId:    msg.Id,
				Message:   "Упс! Произошла ошибка при создании голосования"})
			return
		}
		a.client.CreatePost(&model.Post{
			ChannelId: msg.ChannelId,
			RootId:    msg.Id,
			Message: fmt.Sprintf("Ваше голосование успешно создано!\nID голосования:%s\nТип вопроса:%s\nВарианты ответов:\n%s\n\nГолоса:\n%s",
				voting.ID,
				voting.Question,
				concateOptions(voting.Options),
				concateVotes(voting.Votes),
			)})
	} else {
		a.client.CreatePost(&model.Post{
			ChannelId: msg.ChannelId,
			RootId:    msg.Id,
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

}

func (a *App) GetVoteResults(msg Message) {

}

func (a *App) DeletingVote(msg Message) {

}

func (a *App) FinishingVote(msg Message) {

}

func (a *App) GracefulConsClose() {

}
