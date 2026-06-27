package util

// Version is the application version. It defaults to "development" and
// can be overridden at build time via:
//
//	go build -ldflags "-X github.com/shutterbase/shutterbase/internal/util.Version=v1.0.0"
var Version = "development"
