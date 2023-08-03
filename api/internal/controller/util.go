package controller

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/camera"
	"github.com/shutterbase/shutterbase/ent/timeoffset"
	"github.com/shutterbase/shutterbase/internal/repository"
)

const TIME_OFFSET_CACHE_TTL = 30 * time.Second

type TimeOffsetCacheEntry struct {
	CameraId    uuid.UUID
	Time        time.Time
	TimeOffsets []*ent.TimeOffset
}

func getPaginationParameters(c *gin.Context) repository.PaginationParameters {
	limitParameter := c.DefaultQuery("limit", "100")
	offsetParameter := c.DefaultQuery("offset", "0")
	searchParameter := c.DefaultQuery("search", "")
	sortParameter := c.DefaultQuery("sort", "")
	orderDirectionParameter := c.DefaultQuery("order", "")

	limit, err := strconv.Atoi(limitParameter)
	if err != nil {
		limit = 100
	}
	offset, err := strconv.Atoi(offsetParameter)
	if err != nil {
		offset = 0
	}

	return repository.PaginationParameters{
		Limit:          limit,
		Offset:         offset,
		Search:         searchParameter,
		Sort:           sortParameter,
		OrderDirection: orderDirectionParameter,
	}
}

func getBestTimeOffset(ctx context.Context, cameraId uuid.UUID, t time.Time) (*ent.TimeOffset, error) {
	timeOffsetCacheEntry, ok := timeOffsetCache.Get(cameraId)
	if ok {
		if time.Since(timeOffsetCacheEntry.Time) < TIME_OFFSET_CACHE_TTL {
			return findBestTimeOffsetMatch(timeOffsetCacheEntry.TimeOffsets, t), nil
		}
	}
	timeOffsets, err := repository.GetDatabaseClient().TimeOffset.Query().Where(timeoffset.HasCameraWith(camera.ID(cameraId))).All(ctx)
	if err != nil {
		return nil, err
	}
	timeOffsetCacheEntry = TimeOffsetCacheEntry{
		CameraId:    cameraId,
		Time:        time.Now(),
		TimeOffsets: timeOffsets,
	}
	timeOffsetCache.Add(cameraId, timeOffsetCacheEntry)

	return findBestTimeOffsetMatch(timeOffsets, t), nil
}

func findBestTimeOffsetMatch(timeOffsets []*ent.TimeOffset, t time.Time) *ent.TimeOffset {
	var closestTimeOffset *ent.TimeOffset
	var closestDeltaTime time.Duration
	for _, timeOffset := range timeOffsets {
		// calculate delta time between timeOffset and t
		deltaTime := timeOffset.CameraTime.Sub(t).Abs()
		if closestTimeOffset == nil || deltaTime < closestDeltaTime {
			closestTimeOffset = timeOffset
			closestDeltaTime = deltaTime
		}
	}
	return closestTimeOffset
}
