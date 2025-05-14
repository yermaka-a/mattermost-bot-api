package main

import (
	"bot/internal/app"
	"bot/internal/config"
	"bot/internal/logger"
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

func main() {
	ctx, AppCancel := context.WithCancel(context.Background())
	defer AppCancel()

	// Загрузка переменных окружения
	config := config.MustLoad()

	// Настройка логгера
	zapL := zap.Must(zap.NewProduction())
	defer zapL.Sync()
	logger := logger.SetupLogger(zapL)

	app := app.New(ctx, config, logger)
	go app.WebSocketServer.MustRun()
	logger.Info("Бот успешно запущен", "url-mattermost", config.MATTERMOST_URL)
	// GracefullShutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	AppCancel()
	app.WebSocketServer.Close()
	logger.Info("received shutdown signal")
}
