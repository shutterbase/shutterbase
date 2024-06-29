package hooks

import (
	"time"

	"github.com/pocketbase/pocketbase/models"
)

func (h *HookExecutor) addTagToImage(image *models.Record, tag *models.Record, assignmentType string) error {
	collection, err := h.context.App.Dao().FindCollectionByNameOrId("image_tag_assignments")
	if err != nil {
		return err
	}

	imageTagAssignment := models.NewRecord(collection)

	imageTagAssignment.Set("type", assignmentType)
	imageTagAssignment.Set("imageTag", tag.Id)
	imageTagAssignment.Set("image", image.Id)

	err = h.context.App.Dao().SaveRecord(imageTagAssignment)
	if err != nil {
		return err
	}

	return nil
}

type ImageTagCreateInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsAlbum     bool   `json:"isAlbum"`
	Type        string `json:"type"`
	Project     string `json:"project"`
}

func (h *HookExecutor) createImageTag(input *ImageTagCreateInput) (*models.Record, error) {
	collection, err := h.context.App.Dao().FindCollectionByNameOrId("image_tags")
	if err != nil {
		return nil, err
	}

	imageTag := models.NewRecord(collection)
	imageTag.Set("name", input.Name)
	imageTag.Set("description", input.Description)
	imageTag.Set("isAlbum", input.IsAlbum)
	imageTag.Set("type", input.Type)
	imageTag.Set("project", input.Project)

	err = h.context.App.Dao().SaveRecord(imageTag)
	if err != nil {
		return nil, err
	}

	return imageTag, nil
}

func parseBackendDateTime(dateTime string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05.000Z", dateTime)
}
