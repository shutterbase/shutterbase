package util

import (
	"github.com/pocketbase/pocketbase"
	"github.com/shutterbase/shutterbase/internal/s3"
)

type Context struct {
	App      *pocketbase.PocketBase
	S3Client *s3.S3Client
}
