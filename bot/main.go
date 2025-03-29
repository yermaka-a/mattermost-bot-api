package main

import (
	"bot/internal/app"
	"bot/internal/config"
	"bot/internal/logger"
	"bot/internal/storage"
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/mattermost/mattermost-server/v6/model"

	"github.com/tarantool/go-tarantool/v2"
)

func main() {
	ctx, AppCannel := context.WithCancel(context.Background())
	defer AppCannel()

	// Загрузка переменных окружения
	config := config.GetConfig()

	// Настройка логгера
	log, err := logger.NewLogger()
	if err != nil {
		log.Fatalf("Ошибка создания логгера: %v\n", err)
	}
	// Подключение к Tarantool
	dialer := tarantool.NetDialer{Address: config.TARANTOOL_ADDR, User: config.TARANTOOL_USER, Password: config.TARANTOOL_PASS}
	storage, err := storage.NewTarantoolStorage(dialer)
	if err != nil {
		log.Fatalf("Ошибка подключения к Tarantool", "error", err)
	}
	// Установка подключения к Mattermost

	// Создание и запуск бота
	client := model.NewAPIv4Client(config.MATTERMOST_URL)
	client.SetOAuthToken(config.BOT_TOKEN)
	app.Start(ctx, client, storage, config, log)
	log.Infow("Бот успешно запущен", "url", config.MATTERMOST_URL)
	// GracefullShutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	AppCannel()
	storage.GracefulConnClose()
	log.Infoln("Received shutdown signal")
}
