package hooks

import (
	ctx "context"
	"encoding/json"
	"fmt"

	"github.com/pocketbase/pocketbase/models"
	"github.com/shutterbase/shutterbase/internal/util"
)

func (h *HookExecutor) addDownloadUrls(record *models.Record) {
	downloadUrls := map[string]string{}
	objectIds := util.GetObjectIds(record.GetString("storageId"))
	for size, objectId := range objectIds {
		recordKey := fmt.Sprintf("%d", size)
		if size == 0 {
			recordKey = "original"
		}
		downloadUrl, err := h.context.S3Client.GetSignedDownloadUrl(ctx.Background(), objectId)
		if err != nil {
			h.context.App.Logger().Error(fmt.Sprintf("Failed to get download URL for image '%s' => %s", record.GetString("computedFileName"), err.Error()))
			continue
		}
		downloadUrls[recordKey] = downloadUrl
	}
	data, err := json.Marshal(downloadUrls)
	if err != nil {
		h.context.App.Logger().Error(fmt.Sprintf("Failed to marshal download URLs for image '%s' => %s", record.GetString("computedFileName"), err.Error()))
		return
	}
	record.Set("downloadUrls", string(data))
}
