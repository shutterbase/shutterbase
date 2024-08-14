package server

import (
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/pocketbase/pocketbase"
	"github.com/shutterbase/shutterbase/internal/s3"
)

type Server struct {
	S3Client         *s3.S3Client
	App              *pocketbase.PocketBase
	WebsocketManager *WebsocketManager
	TagCountCache    *expirable.LRU[string, *ImageTagWithCount]
}

type ServerOptions struct {
	S3Client *s3.S3Client
	App      *pocketbase.PocketBase
}

func NewServer(options *ServerOptions) *Server {
	return &Server{
		S3Client:      options.S3Client,
		App:           options.App,
		TagCountCache: expirable.NewLRU[string, *ImageTagWithCount](100000, nil, time.Minute*5),
	}
}

func (s *Server) RegisterRoutes() error {
	s.registerGetUploadUrlEndpoint()
	s.registerWebsocketServer()

	s.registerSyncImageTagsEndpoint()
	s.registerStatisticsEndpoint()
	return nil
}
