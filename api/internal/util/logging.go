package util

import (
	"os"

	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func InitLogger() error {
	setLogLevel()
	setLogOutput()
	log.Info().Msgf("Logger initialized on level '%s'", zerolog.GlobalLevel().String())
	return nil
}

func setLogOutput() {
	devMode := config.Get().Bool("DEV_MODE")
	if devMode {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02T15:04:05.000Z"})
	} else {
		zerolog.TimeFieldFormat = "2006-01-02T15:04:05.000Z"
	}
}

func setLogLevel() {
	logLevel := config.Get().String("LOG_LEVEL")
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
