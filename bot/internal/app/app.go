package app

import (
	wsServer "bot/internal/app/ws"
	"bot/internal/config"
	"context"
	"log/slog"
)

type App struct {
	WebSocketServer wsServer.WSApp
}

func New(ctx context.Context, cfg *config.Config, logger *slog.Logger) *App {

	webSocketServer, err := wsServer.New(ctx, cfg, logger)

	if err != nil {
		panic(err)
	}
	return &App{
		WebSocketServer: webSocketServer,
	}
}
