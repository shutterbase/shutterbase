package hooks

import (
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/pocketbase/pocketbase/models"
	"github.com/shutterbase/shutterbase/internal/util"
)

type HookExecutor struct {
	context        *util.Context
	caches         *HookExecutorCaches
	aiImageQueue   []*AiDetectionObject
	aiBackoffUntil *time.Time
	lock           sync.Mutex
}

const (
	DATE_TAG_HOUR_OFFSET = -3
)

type HookExecutorCaches struct {
	// caches all default tags for a given project id
	projectDefaultTagsCache *expirable.LRU[string, []*models.Record]
	// caches default tags
	tagCache *expirable.LRU[string, *models.Record]
}

func RegisterHooks(context *util.Context) error {
	hookExecutor := HookExecutor{
		context: context,
		caches: &HookExecutorCaches{
			projectDefaultTagsCache: expirable.NewLRU[string, []*models.Record](100, nil, time.Second*30),
			tagCache:                expirable.NewLRU[string, *models.Record](100, nil, time.Minute*5),
		},
		lock:         sync.Mutex{},
		aiImageQueue: []*AiDetectionObject{},
	}
	hookExecutor.registerProjectAssignmentHooks()
	hookExecutor.registerUserHooks()
	hookExecutor.registerImageHooks()

	hookExecutor.StartImageDetectionProcessor()
	return nil
}
