package util_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/internal/util"
)

// DeploymentImageTag backs the /health probe that `/deploy-fsg` polls to decide
// whether a rollout actually reached the pods. If it ever stops reflecting the
// env var the k8s manifest sets, deploy verification silently starts lying —
// which is exactly what util.Version did (an ldflags var no build ever sets, so
// it always read "development").
func TestDeploymentImageTagReflectsEnv(t *testing.T) {
	t.Setenv("DEPLOYMENT_IMAGE_TAG", "v9.9.9")
	require.NoError(t, util.InitConfig())
	assert.Equal(t, "v9.9.9", util.DeploymentImageTag())
}

func TestDeploymentImageTagDefaultsToDevelopment(t *testing.T) {
	require.NoError(t, util.InitConfig())
	assert.Equal(t, "development", util.DeploymentImageTag(),
		"unset locally -> the config default, never an empty string")
}
