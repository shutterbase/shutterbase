package util

import "github.com/mxcd/go-config/config"

func InitConfig() error {
	err := config.LoadConfig([]config.Value{
		config.String("LOG_LEVEL").Default("info"),
		config.Bool("DEV").Default(false),
		config.String("DOMAIN_NAME").NotEmpty(),

		config.String("THUMBNAIL_SIZES").NotEmpty().Default("256,512,1024,2048"),

		config.String("S3_ENDPOINT").NotEmpty(),
		config.Bool("S3_SSL").Default(true),
		config.Int("S3_PORT").Default(443),
		config.String("S3_BUCKET").NotEmpty().Default("shutterbase"),
		config.String("S3_ACCESS_KEY").NotEmpty(),
		config.String("S3_SECRET_KEY").NotEmpty().Sensitive(),

		config.String("OPENAI_API_KEY").NotEmpty().Sensitive(),
	})
	return err
}
