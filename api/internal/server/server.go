package server

import (
	"github.com/pocketbase/pocketbase"
	"github.com/shutterbase/shutterbase/internal/s3"
)

type Server struct {
	S3Client         *s3.S3Client
	App              *pocketbase.PocketBase
	WebsocketManager *WebsocketManager
}

type ServerOptions struct {
	S3Client *s3.S3Client
	App      *pocketbase.PocketBase
}

func NewServer(options *ServerOptions) *Server {
	return &Server{
		S3Client: options.S3Client,
		App:      options.App,
	}
}

func (s *Server) RegisterRoutes() error {
	s.registerGetUploadUrlEndpoint()
	s.registerWebsocketServer()
	return nil
}
