package wsbot

import (
	"bot/internal/config"
	"bot/internal/domain/models"
	botService "bot/internal/services/bot"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
)

// bot commands
const (
	HELP        = "@bot_help"
	CREATE_VOTE = "@create"
	VOTE        = "@vote"
	RESULTS     = "@results"
	FINISHED    = "@finished"
	DELETE      = "@delete"
)

// routes
const (
	me = "/users/me"
)

var help_message = fmt.Sprintf("Добрый день я бот для создания опросов, вы можете взаимодействовать со мной следующим образом:\n%s\n%s\n%s\n%s\n%s\n%s",
	"**"+HELP+" - Команда отправляет сообщение об описании функционала",
	"**"+CREATE_VOTE+" \"Ваш вопрос\" \"Вариант 1\" \"Вариант 2\"** - пример как создать опрос",
	"**"+VOTE+" ID_голосования вариант_ответа** - пример как проголосовать в опросе",
	"**"+RESULTS+" Id_голосования** - пример как получить результаты",
	"**"+FINISHED+" Id_голосования** - пример как завершить опрос досрочно (Доступо только создателю опроса)",
	"**"+DELETE+" Id_голосования** - пример как удалить опрос (Доступно только создателю опроса)",
)

type BotAPI interface {
	Listen()
	Close() error
}

type botAPI struct {
	bs  botService.BotService
	m   *MattermostClient
	log *slog.Logger
}

type bot struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type MattermostClient struct {
	ctx      context.Context
	client   *model.Client4
	wsClient *model.WebSocketClient
	bot      *bot
}

func Register(mClient *MattermostClient, bs botService.BotService, log *slog.Logger) BotAPI {
	return &botAPI{
		bs:  bs,
		m:   mClient,
		log: log,
	}
}

func NewMattermostClient(ctx context.Context, cfg *config.Config) (*MattermostClient, error) {
	op := "wsServer.NewMattermostClient"
	client := model.NewAPIv4Client(cfg.MATTERMOST_URL)
	client.SetOAuthToken(cfg.BOT_TOKEN)

	client.HTTPHeader = map[string]string{
		"Authorization": "Bearer " + cfg.BOT_TOKEN,
	}
	var bot bot
	// установить user_id и username для бота Bot
	err := getBotData(client, &bot)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	wsClient, err := model.NewWebSocketClient4(cfg.MATTERMOST_WS, client.AuthToken)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &MattermostClient{
		ctx:      ctx,
		client:   client,
		wsClient: wsClient,
		bot:      &bot,
	}, nil
}

func getBotData(client *model.Client4, bot *bot) error {
	res, err := client.DoAPIRequestReader("GET", client.APIURL+me, nil, client.HTTPHeader)

	if err != nil {
		return err
	}

	data, _ := io.ReadAll(res.Body)

	err = json.Unmarshal(data, &bot)
	if err != nil {
		return err
	}
	return nil
}

func (b *botAPI) Listen() {
	b.m.wsClient.Listen()
LOOP:
	for {
		select {
		case event := <-b.m.wsClient.EventChannel:
			if event.EventType() == model.WebsocketEventPosted {

				postStr, ok := event.GetData()["post"].(string)
				if ok {
					var post map[string]interface{}
					json.Unmarshal([]byte(postStr), &post)
					msg := models.Message{
						Id:        post["id"].(string),
						UserId:    post["user_id"].(string),
						ChannelId: post["channel_id"].(string),
						// Удаляем лишние пробелы из строки
						Message: strings.Join(strings.Fields(strings.TrimSpace(post["message"].(string))), " "),
					}
					if b.m.bot.ID != msg.UserId {
						typeMsg, message := getCommand(msg.Message)
						msg.Message = message
						switch typeMsg {
						case HELP:
							go b.help(msg)

						case CREATE_VOTE:
							go b.createVote(msg)

						case VOTE:
							go b.toVote(msg)

						case RESULTS:
							go b.getVoteResults(msg)

						case FINISHED:
							go b.finishingVote(msg)

						case DELETE:
							go b.deletingVote(msg)

						}
					}
				}
			}
			// Начало беседы с ботом в ЛС для пользователей, кроме создателя (т.к. access token ещё неактивен)
			if event.EventType() == model.WebsocketEventDirectAdded {
				go b.greetingHelp(event.GetData(), b.m.bot)
			}
		case <-b.m.ctx.Done():
			break LOOP
		}
	}
}

func (b *botAPI) Close() error {
	b.m.wsClient.Close()
	err := b.bs.Close()
	b.m.ctx.Done()
	return err

}

func getCommand(message string) (string, string) {
	commands := []string{HELP, CREATE_VOTE, VOTE, FINISHED, RESULTS, DELETE}
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

func (b *botAPI) help(msg models.Message) {
	b.m.client.CreatePost(&model.Post{
		ChannelId: msg.ChannelId,
		Message:   help_message})
}

func (b *botAPI) createVote(msg models.Message) {

	voting, err := b.bs.CreateVote(&msg)
	if err != nil {
		b.log.Error("can't create voting:", slog.String("error", err.Error()))
		b.m.client.CreatePost(&model.Post{
			ChannelId: msg.ChannelId,
			Message:   "Опрос не был создан, возможно данные введены некорректно..."})
		return
	}
	b.m.client.CreatePost(&model.Post{
		ChannelId: msg.ChannelId,
		Message: fmt.Sprintf("**Ваше голосование успешно создано!**\nID голосования:**%s**\nТип вопроса:%s\nВарианты ответов:\n%s",
			voting.ID,
			voting.Question,
			concatOptions(voting.Options),
		)})

}

func concatVotes(votes []int64) string {
	builder := &strings.Builder{}
	for i, v := range votes {
		builder.WriteString(fmt.Sprintf("%d. %d;\t", i+1, v))
	}
	return builder.String()
}
func concatOptions(options []string) string {
	builder := &strings.Builder{}
	for i, v := range options {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, v))
	}
	return builder.String()
}

func (b *botAPI) toVote(msg models.Message) {
	parts := b.bs.ParseVote(msg.Message)
	//Get voting by ID
	if parts == nil {
		b.m.client.CreatePost(&model.Post{
			ChannelId: msg.ChannelId,
			Message:   "Некорректный запрос"})
		return
	}
	voting, err := b.bs.GetVoteLocally(parts)
	if err != nil {
		b.m.client.CreatePost(&model.Post{
			ChannelId: msg.ChannelId,
			Message:   "Такого голосования не нашлось."})
		return
	}
	if voting.IsActive {
		now := time.Now()
		if now.Before(time.Unix(voting.ExpiresAt, 0)) {

			user, err := b.bs.GetUserById(msg.UserId)
			if err != nil {
				b.log.Error(err.Error())
				b.m.client.CreatePost(&model.Post{
					ChannelId: msg.ChannelId,
					Message:   "Что-то пошло не так..."})
				return
			}
			if user == nil {

				user = b.bs.CreateUser(msg.UserId)
			}
			votes, err := b.bs.UpdateVoteLocaly(voting.Votes, parts[1])
			if err != nil {
				b.log.Info(voting.ID, voting.CreatorID, err)
				b.m.client.CreatePost(&model.Post{
					ChannelId: msg.ChannelId,
					Message:   "Вариант ответа некорректный!"})
				return

			}
			option, err := strconv.Atoi(parts[1])
			if err != nil {
				b.m.client.CreatePost(&model.Post{
					ChannelId: msg.ChannelId,
					Message:   "Что-то пошло не так..."})
				return
			}
			// проголосовал ли пользователь уже в этом опросе
			isVoted := false
			b.log.Info("info", "votes", user.Votes)
			if user.Votes[parts[0]] != 0 {
				votes[user.Votes[parts[0]]-1] = votes[user.Votes[parts[0]]-1] - 1
				isVoted = true
			}
			user.Votes[parts[0]] = int64(option)
			voting.Votes = votes
			err = b.bs.UpdateUser(user)
			if err != nil {
				b.m.client.CreatePost(&model.Post{
					ChannelId: msg.ChannelId,
					Message:   "Что-то пошло не так..."})
				return
			}
			err = b.bs.UpdateVote(voting)
			if err != nil {
				b.log.Error(err.Error())
				b.m.client.CreatePost(&model.Post{
					ChannelId: msg.ChannelId,
					Message:   "Что-то пошло не так..."})
				return
			}
			if isVoted {
				b.m.client.CreatePost(&model.Post{
					ChannelId: msg.ChannelId,
					Message:   fmt.Sprintf("Ваш голос перезаписан!\nГолосование:%s\nВопрос:%s\nВарианты:\n%s\nГолоса:\n%s", voting.ID, voting.Question, concatOptions(voting.Options), concatVotes(voting.Votes))})

			} else {
				b.m.client.CreatePost(&model.Post{
					ChannelId: msg.ChannelId,

					Message: fmt.Sprintf("Ваш голос учтён!\nГолосование:%s\nВопрос:%s\nВарианты:\n%s\nГолоса:\n%s", voting.ID, voting.Question, concatOptions(voting.Options), concatVotes(voting.Votes))})
			}
		} else {
			if voting.IsActive {
				voting.IsActive = false
			}
			err := b.bs.UpdateVote(voting)
			if err != nil {
				b.log.Error("func ToVote", "can't update vote", err.Error())
			}
			b.m.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				Message:   fmt.Sprintf("Время голосования истекло %s", time.Unix(voting.ExpiresAt, 0).Format("2006-01-02 15:04:05"))})

		}
	} else {
		b.m.client.CreatePost(&model.Post{
			ChannelId: msg.ChannelId,
			Message:   fmt.Sprintf("Время голосования истекло %s", time.Unix(voting.ExpiresAt, 0).Format("2006-01-02 15:04:05"))})

	}

}

func (b *botAPI) getVoteResults(msg models.Message) {

	isVoteId := b.bs.IsValidID(msg.Message)
	//Get voting by ID
	if isVoteId {
		voting, err := b.bs.GetVote(msg.Message)
		if err != nil {
			b.log.Info("voting not found", "error", err.Error())
			b.m.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				Message:   "Такого голосования не нашлось."})
			return
		}
		b.m.client.CreatePost(&model.Post{
			ChannelId: msg.ChannelId,
			Message:   fmt.Sprintf("Голосование:%s\nВопрос:%s\nВарианты:\n%s\nГолоса:\n%s", voting.ID, voting.Question, concatOptions(voting.Options), concatVotes(voting.Votes))})
	} else {
		b.m.client.CreatePost(&model.Post{
			ChannelId: msg.ChannelId,
			Message:   "Некорректный ID опроса"})
	}
}

func (b *botAPI) deletingVote(msg models.Message) {
	isVoteId := b.bs.IsValidID(msg.Message)
	//Get voting by ID
	if isVoteId {
		voting, err := b.bs.GetVote(msg.Message)
		if err != nil {
			b.log.Info("voting not found", "error", err.Error())
			b.m.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				Message:   "Такого голосования не нашлось."})
			return
		}
		if voting.CreatorID == msg.UserId {
			err := b.bs.DeleteVote(voting.ID)
			if err != nil {
				b.log.Error("can't delete voting", "error", err.Error())
				b.m.client.CreatePost(&model.Post{
					ChannelId: msg.ChannelId,
					Message:   "При удалении произошла ошибка!"})
				return
			}
			b.m.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				Message:   fmt.Sprintf("Голосование с ID:%s успешно удалено!", voting.ID)})
		} else {
			b.m.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				Message:   fmt.Sprintf("У вас нет прав на удаление опроса с ID:%s", voting.ID)})
		}

	} else {
		b.m.client.CreatePost(&model.Post{
			ChannelId: msg.ChannelId,
			Message:   "Некорректный ID опроса"})
	}
}

func (b *botAPI) finishingVote(msg models.Message) {
	isVoteId := b.bs.IsValidID(msg.Message)
	//Get voting by ID
	if isVoteId {
		voting, err := b.bs.GetVote(msg.Message)
		if err != nil {
			b.log.Info("voting not found", "error", err)
			b.m.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				Message:   "Такого голосования не нашлось."})
			return
		}
		if !voting.IsActive {
			b.m.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				Message:   "Голосование уже завершено!"})
			return
		}

		if voting.CreatorID == msg.UserId {
			voting.IsActive = false
			err := b.bs.UpdateVote(voting)
			if err != nil {
				b.log.Error("can't finished voting", "error", err.Error())
				b.m.client.CreatePost(&model.Post{
					ChannelId: msg.ChannelId,
					Message:   "Произошла ошибка!"})
				return
			}
			b.m.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				Message:   fmt.Sprintf("Голосование с ID:%s завершено досрочно!", voting.ID)})
		} else {
			b.m.client.CreatePost(&model.Post{
				ChannelId: msg.ChannelId,
				Message:   "У вас нет прав на завершение этого голосования досрочно."})
		}

	} else {
		b.m.client.CreatePost(&model.Post{
			ChannelId: msg.ChannelId,
			Message:   "Некорректный ID"})
	}
}

func (b *botAPI) greetingHelp(data map[string]interface{}, bot *bot) {
	creatorId := data["creator_id"].(string)
	teammateId := data["teammate_id"].(string)
	if creatorId != "" && teammateId != "" {
		if teammateId == bot.ID {
			chl, _, err := b.m.client.CreateDirectChannel(creatorId, teammateId)
			if err != nil {
				b.log.Error("can't create direct channel", "error", err.Error())
			}
			b.m.client.CreatePost(&model.Post{
				ChannelId: chl.Id,
				Message:   help_message})
		}
	}
}
