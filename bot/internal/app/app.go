package app

import (
	"bot/internal/config"
	"bot/internal/logger"
	"bot/internal/storage"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

				if ok {
					var post map[string]interface{}
					json.Unmarshal([]byte(postStr), &post)
					msg := Message{
						UserId:    post["user_id"].(string),
						ChannelId: post["channel_id"].(string),
						Message:   post["message"].(string),
					}
					fmt.Println(post)
					if bot.ID != msg.UserId {

						client.CreatePost(&model.Post{ChannelId: msg.ChannelId, Message: msg.Message})
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
