package util

import "github.com/mxcd/go-config/config"

// Version is the application version. It defaults to "development" and
// can be overridden at build time via:
//
//	go build -ldflags "-X github.com/shutterbase/shutterbase/internal/util.Version=v1.0.0"
var Version = "development"

// DeploymentImageTag is the image tag the deployment pinned (DEPLOYMENT_IMAGE_TAG,
// set by the k8s manifest; "development" locally). This — not Version — is the
// honest answer to "which build is running", because no build passes the ldflags
// that would set Version.
func DeploymentImageTag() string {
	return config.Get().String("DEPLOYMENT_IMAGE_TAG")
}
