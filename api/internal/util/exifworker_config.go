package util

import "github.com/mxcd/go-config/config"

func InitExifWorkerConfig() error {
	err := config.LoadConfig([]config.Value{
		config.String("LOG_LEVEL").Default("info"),
		config.Bool("DEV").Default(false),
		config.Int("EXIF_WORKER_PORT").Default(8080),
		config.String("INTERNAL_POCKETBASE_URL").Default("http://localhost:8090"),
	})
	return err
}
