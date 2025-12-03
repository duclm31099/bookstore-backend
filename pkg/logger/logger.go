package logger

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Init(env string) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if env == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func Info(msg string, fields map[string]interface{}) {
	log.Info().Fields(fields).Msg(msg)
}
func Debug(msg string) {
	log.Debug().Msg(msg)
}

func Error(msg string, err error) {
	log.Error().Err(err).Msg(msg)
}
