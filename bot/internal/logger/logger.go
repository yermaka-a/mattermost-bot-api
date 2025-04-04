package logger

import (
	"log/slog"

	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

func SetupLogger(zap *zap.Logger) *slog.Logger {

	logger := slog.New(zapslog.NewHandler(zap.Core()))
	return logger
}
