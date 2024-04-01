package util

import (
	"os"
	"time"

	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

func InitLogger() error {
	setLogLevel()
	setLogOutput()
	return nil
}

func setLogOutput() {
	dev := config.Get().Bool("DEV")

	zerolog.TimeFieldFormat = "2006-01-02T15:04:05.000Z"
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	if dev {
		log.Logger = log.Logger.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			NoColor:    false,
			TimeFormat: time.RFC3339,
		}).With().Caller().Logger()
	} else {
		log.Logger = log.Logger.With().Caller().Logger()
	}
}

func setLogLevel() {
	logLevel := config.Get().String("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	switch logLevel {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "warning":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "err":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}
