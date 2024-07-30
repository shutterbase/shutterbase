package main

import (
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/mxcd/go-config/config"
	"github.com/shutterbase/shutterbase/internal/client"
	"github.com/shutterbase/shutterbase/internal/exif"
	"github.com/shutterbase/shutterbase/internal/util"
)

func main() {

	err := util.InitExifWorkerConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize config")
	}
	config.Print()

	err = util.InitLogger()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize logger")
	}

	engine := gin.Default()

	if config.Get().Bool("DEV") {
		corsConfig := cors.DefaultConfig()
		corsConfig.AllowOrigins = []string{"http://localhost:9000"}
		corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
		engine.Use(cors.New(corsConfig))
	}

	internalPocketbaseUrl := config.Get().String("INTERNAL_POCKETBASE_URL")
	engine.GET("/api/download/:id/:resolution", func(c *gin.Context) {
		id := c.Param("id")
		resolution := c.Param("resolution")
		token := c.Request.Header.Get("Authorization")
		if token == "" {
			c.JSON(401, gin.H{"error": "missing token"})
			return
		}

		log.Debug().Str("id", id).Str("resolution", resolution).Msg("download")
		client := client.NewClient(internalPocketbaseUrl)
		client.SetToken(token)
		data, err := exif.GetImageFileWithAdjustedExifData(c.Request.Context(), id, resolution, client)
		if err != nil {
			log.Error().Err(err).Msg("failed to get image file with adjusted exif data")
			c.JSON(500, gin.H{"error": "failed to get image file with adjusted exif data"})
			return
		}
		c.Header("Content-Type", "image/jpeg")
		c.Header("Content-Length", fmt.Sprintf("%d", len(data)))
		c.Writer.Write(data)
	})

	engine.Run(fmt.Sprintf(":%d", config.Get().Int("EXIF_WORKER_PORT")))
}
