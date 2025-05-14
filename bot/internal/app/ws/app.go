package wsServer

import (
	"bot/internal/config"
	botService "bot/internal/services/bot"
	"bot/internal/storage"
	wsbot "bot/internal/ws/bot"
	"context"
	"fmt"
	"log/slog"
)

type WSApp interface {
	MustRun()
	Close()
}

type app struct {
	log *slog.Logger
	ba  wsbot.BotAPI
}

func New(ctx context.Context, cfg *config.Config, logger *slog.Logger) (WSApp, error) {
	op := "wsServer.New"

	mClient, err := wsbot.NewMattermostClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	storage, err := storage.NewStorage(*cfg)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	bs := botService.New(logger, storage)
	botAPI := wsbot.Register(mClient, bs, logger)
	app := &app{
		log: logger,
		ba:  botAPI,
	}
	return app, nil
}

func (a *app) MustRun() {
	err := a.run()
	if err != nil {
		panic(err)
	}
}

// Установка подключения к Mattermost
// Подключение к боту
func (a *app) run() error {
	a.ba.Listen()
	return nil
}

func (a *app) Close() {
	op := "wsServer.GracefullClose"
	log := a.log.With("op", op)
	log.Info("closing connection", slog.String("op", op))
	err := a.ba.Close()
	log.Error("connection closure error", slog.String("error", err.Error()))
}
