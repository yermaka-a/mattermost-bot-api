package config

import (
	"fmt"
	"os"
)

var config *BotConfig

type BotConfig struct {
	MATTERMOST_URL string
	BOT_TOKEN      string
	TARANTOOL_ADDR string
	TARANTOOL_PASS string
	TARANTOOL_USER string
	MATTERMOST_WS  string
}

func GetConfig() *BotConfig {
	if config == nil {
		config = &BotConfig{
			MATTERMOST_URL: getEnv("MATTERMOST_URL", "http://localhost:8065"),
			BOT_TOKEN:      getEnv("BOT_TOKEN", "z8yxbmc5tbbbtk9k4j4j3jghxw"),
			TARANTOOL_ADDR: getEnv("TARANTOOL_ADDR", "localhost:3301"),
			MATTERMOST_WS:  getEnv("MATTERMOST_WS", "ws://localhost:8065"),
			TARANTOOL_PASS: getEnv("TARANTOOL_PASS", "passw0rd"),
			TARANTOOL_USER: getEnv("TARANTOOL_USER", "storage"),
		}
	}
	return config
}

func getEnv(key string, defaultVal string) string {
	if value, isExist := os.LookupEnv(key); isExist {
		return value
	}
	if defaultVal == "" {
		panic(fmt.Sprintf("Doesnt have required variable by key: %s ", key))
	}
	return defaultVal
}
