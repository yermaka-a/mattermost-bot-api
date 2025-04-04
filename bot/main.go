package main

import (
	"bot/internal/app"
	"bot/internal/config"
	"bot/internal/logger"
	"bot/internal/storage"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mattermost/mattermost-server/v6/model"
	"go.uber.org/zap"

	"github.com/tarantool/go-tarantool/v2"
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

	// Подключение к Tarantool
	dialer := tarantool.NetDialer{Address: config.TARANTOOL_ADDR, User: config.TARANTOOL_USER, Password: config.TARANTOOL_PASS}
	storage, err := storage.NewStorage(dialer)
	if err != nil {
		log.Fatalln("Ошибка подключения к Tarantool", "error", err)
	}
	// Создание БД tarantool
	_, err = storage.CreateDB()
	if err != nil {
		log.Fatalln("Can't create database", "error", err.Error())
	}
	// Установка подключения к Mattermost
	// Создание и запуск бота
	client := model.NewAPIv4Client(config.MATTERMOST_URL)
	client.SetOAuthToken(config.BOT_TOKEN)
	app.Start(ctx, client, storage, config, logger)
	logger.Info("Бот успешно запущен", "url-mattermost", config.MATTERMOST_URL)
	// GracefullShutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	AppCancel()
	storage.GracefulConnClose()
	logger.Info("Received shutdown signal")
}
