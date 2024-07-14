package hooks

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/models"
)

func (h *HookExecutor) addDefaultTags(image *models.Record) error {
	defaultTagTemplates, err := h.getProjectDefaultTagTemplates(image.GetString("project"))
	if err != nil {
		return err
	}

	for _, defaultTag := range defaultTagTemplates {
		tagName := defaultTag.GetString("name")
		switch tagName {
		case "$PROJECT":
			err = h.addProjectDefaultTag(image)
			if err != nil {
				return err
			}
		case "$DATE":
			err = h.addDateDefaultTag(image)
			if err != nil {
				return err
			}
		case "$WEEKDAY":
			err = h.addWeekdayDefaultTag(image)
			if err != nil {
				return err
			}
		case "$COPYRIGHT":
			err = h.addCopyrightDefaultTag(image)
			if err != nil {
				return err
			}
		default:
			h.context.App.Logger().Warn(fmt.Sprintf("Unknown default tag '%s'", tagName))
		}
	}

	return nil
}

func (h *HookExecutor) addProjectDefaultTag(image *models.Record) error {
	projectDefaultTag, err := h.getProjectDefaultTag(image.GetString("project"))
	if err != nil {
		return err
	}

	return h.addTagToImage(image, projectDefaultTag, "default")
}

func (h *HookExecutor) addDateDefaultTag(image *models.Record) error {
	imageDateTag, err := h.getImageDateTag(image)
	if err != nil {
		return err
	}

	return h.addTagToImage(image, imageDateTag, "default")
}

func (h *HookExecutor) addWeekdayDefaultTag(image *models.Record) error {
	imageWeekdayTag, err := h.getImageWeekdayTag(image)
	if err != nil {
		return err
	}

	return h.addTagToImage(image, imageWeekdayTag, "default")
}

func (h *HookExecutor) addCopyrightDefaultTag(image *models.Record) error {
	imageCopyrightTag, err := h.getImageCopyrightTag(image)
	if err != nil {
		return err
	}

	return h.addTagToImage(image, imageCopyrightTag, "default")
}

func (h *HookExecutor) getProjectDefaultTag(projectId string) (*models.Record, error) {
	projectDefaultTag, ok := h.caches.tagCache.Get(projectId)
	if ok {
		return projectDefaultTag, nil
	}

	project, err := h.context.App.Dao().FindRecordById("projects", projectId)
	if err != nil {
		return nil, err
	}

	projectDefaultTag, _ = h.context.App.Dao().FindFirstRecordByFilter(
		"image_tags", "type = 'default' && project = {:projectId} && name = {:projectName}",
		dbx.Params{"projectId": projectId, "projectName": project.GetString("name")},
	)

	if projectDefaultTag != nil {
		h.caches.tagCache.Add(projectId, projectDefaultTag)
		return projectDefaultTag, nil
	}

	projectDefaultTag, err = h.createImageTag(&ImageTagCreateInput{
		Name:        project.GetString("name"),
		Description: "Project tag",
		IsAlbum:     false,
		Type:        "default",
		Project:     projectId,
	})
	if err != nil {
		return nil, err
	}

	h.caches.tagCache.Add(projectId, projectDefaultTag)
	return projectDefaultTag, nil
}

func (h *HookExecutor) getImageDateTag(image *models.Record) (*models.Record, error) {
	capturedAtCorrected, err := parseBackendDateTime(image.GetString("capturedAtCorrected"))
	if err != nil {
		return nil, err
	}
	capturedAtWithThreshold := capturedAtCorrected.Add(time.Duration(DATE_TAG_HOUR_OFFSET) * time.Hour)

	dateString := capturedAtWithThreshold.Format("20060102")
	dateTagCacheKey := fmt.Sprintf("%s-%s", image.GetString("project"), dateString)

	dateTag, ok := h.caches.tagCache.Get(dateTagCacheKey)
	if ok {
		return dateTag, nil
	}

	dateTag, _ = h.context.App.Dao().FindFirstRecordByFilter(
		"image_tags", "type = 'default' && project = {:projectId} && name = {:dateString}",
		dbx.Params{"projectId": image.GetString("project"), "dateString": dateString},
	)

	if dateTag != nil {
		h.caches.tagCache.Add(dateTagCacheKey, dateTag)
		return dateTag, nil
	}

	dateTag, err = h.createImageTag(&ImageTagCreateInput{
		Name:        dateString,
		Description: fmt.Sprintf("Date tag %s", dateString),
		IsAlbum:     false,
		Type:        "default",
		Project:     image.GetString("project"),
	})

	if err != nil {
		return nil, err
	}

	h.caches.tagCache.Add(dateTagCacheKey, dateTag)
	return dateTag, nil
}

func (h *HookExecutor) getImageWeekdayTag(image *models.Record) (*models.Record, error) {
	capturedAtCorrected, err := parseBackendDateTime(image.GetString("capturedAtCorrected"))
	if err != nil {
		return nil, err
	}
	capturedAtWithThreshold := capturedAtCorrected.Add(time.Duration(DATE_TAG_HOUR_OFFSET) * time.Hour)

	weekdayString := capturedAtWithThreshold.Format("Monday")
	weekdayTagCacheKey := fmt.Sprintf("%s-%s", image.GetString("project"), weekdayString)

	weekdayTag, ok := h.caches.tagCache.Get(weekdayTagCacheKey)
	if ok {
		return weekdayTag, nil
	}

	weekdayTag, _ = h.context.App.Dao().FindFirstRecordByFilter(
		"image_tags", "type = 'default' && project = {:projectId} && name = {:weekdayString}",
		dbx.Params{"projectId": image.GetString("project"), "weekdayString": weekdayString},
	)

	if weekdayTag != nil {
		h.caches.tagCache.Add(weekdayTagCacheKey, weekdayTag)
		return weekdayTag, nil
	}

	weekdayTag, err = h.createImageTag(&ImageTagCreateInput{
		Name:        weekdayString,
		Description: fmt.Sprintf("Weekday tag %s", weekdayString),
		IsAlbum:     false,
		Type:        "default",
		Project:     image.GetString("project"),
	})

	if err != nil {
		return nil, err
	}

	h.caches.tagCache.Add(weekdayTagCacheKey, weekdayTag)
	return weekdayTag, nil
}

func (h *HookExecutor) getImageCopyrightTag(image *models.Record) (*models.Record, error) {
	userId := image.GetString("user")

	user, err := h.context.App.Dao().FindRecordById("users", userId)
	if err != nil {
		return nil, err
	}

	copyrightTagString := user.GetString("copyrightTag")
	copyrightTagCacheKey := fmt.Sprintf("%s-%s", image.GetString("project"), copyrightTagString)

	copyrightTag, ok := h.caches.tagCache.Get(copyrightTagCacheKey)
	if ok {
		return copyrightTag, nil
	}

	copyrightTag, _ = h.context.App.Dao().FindFirstRecordByFilter(
		"image_tags", "type = 'default' && project = {:projectId} && name = {:copyrightTagString}",
		dbx.Params{"projectId": image.GetString("project"), "copyrightTagString": copyrightTagString},
	)

	if copyrightTag != nil {
		h.caches.tagCache.Add(copyrightTagCacheKey, copyrightTag)
		return copyrightTag, nil
	}

	copyrightTag, err = h.createImageTag(&ImageTagCreateInput{
		Name:        copyrightTagString,
		Description: fmt.Sprintf("Copyright tag %s", copyrightTagString),
		IsAlbum:     false,
		Type:        "default",
		Project:     image.GetString("project"),
	})

	if err != nil {
		return nil, err
	}

	h.caches.tagCache.Add(copyrightTagCacheKey, copyrightTag)
	return copyrightTag, nil
}

func (h *HookExecutor) getProjectDefaultTagTemplates(projectId string) ([]*models.Record, error) {
	if records, ok := h.caches.projectDefaultTagsCache.Get(projectId); ok {
		return records, nil
	}

	records, err := h.context.App.Dao().FindRecordsByExpr("image_tags", dbx.NewExp("project = {:project}", dbx.Params{"project": projectId}), dbx.NewExp("type = {:type}", dbx.Params{"type": "template"}))
	if err != nil {
		return nil, err
	}

	templateTags := []*models.Record{}
	tagNames := []string{}
	for _, record := range records {
		tagName := record.GetString("name")
		if strings.HasPrefix(tagName, "$") {
			tagNames = append(tagNames, tagName)
			templateTags = append(templateTags, record)
		}
	}

	h.context.App.Logger().Debug(fmt.Sprintf("Found %d default tags for project '%s': [%s]", len(records), projectId, strings.Join(tagNames, ",")))

	h.caches.projectDefaultTagsCache.Add(projectId, templateTags)
	return templateTags, nil
}
