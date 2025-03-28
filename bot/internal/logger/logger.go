package logger

import (
	"go.uber.org/zap"
)

type Logger struct {
	*zap.SugaredLogger
}

func NewLogger() (*Logger, error) {
	config := zap.NewProductionConfig()
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}
	return &Logger{logger.Sugar()}, nil
}
