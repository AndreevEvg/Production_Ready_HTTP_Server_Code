package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

type Logger struct {
	zerolog.Logger
}

// New создает новый экземпляр логгера с настройками для продакшена
func New(serviceName string, environment string) *Logger {
	// Настраиваем формат времени
	zerolog.TimeFieldFormat = time.RFC3339Nano

	// Включаем поддержку ошибок со стектрейсом
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	// Определяем уровень логирования
	logLevel := zerolog.InfoLevel 
	if environment == "development" {
		logLevel = zerolog.DebugLevel 
	}

	// Создаем контекст логгера с глобальными полями
	ctx := zerolog.New(os.Stdout).
		Level(logLevel).
		With().
		Timestamp().
		Str("service", serviceName).
		Str("env", environment).
		Logger()

	return &Logger{ctx}
}

// Fatal логирует фатальную ошибку и завершает программу
func (l *Logger) Fatal(err error, msg string) {
	l.Error().Stack().Err(err).Msg(msg)
	os.Exit(1)
}