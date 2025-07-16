package hooks

import (
	"context"
	"fmt"
	"time"

	"github.com/mxcd/go-config/config"
	"github.com/pocketbase/dbx"
	openai "github.com/sashabaranov/go-openai"
	"github.com/shutterbase/shutterbase/internal/util"
)

type AiDetectionObject struct {
	ImageId string
}

func (h *HookExecutor) queueImageDetection(imageId string) {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.aiImageQueue = append(h.aiImageQueue, &AiDetectionObject{ImageId: imageId})
}

func (h *HookExecutor) StartImageDetectionProcessor() {
	go func() {
		for {
			if h.aiBackoffUntil != nil && time.Now().Before(*h.aiBackoffUntil) {
				h.context.App.Logger().Warn("AI backoff in effect for 30 seconds")
				time.Sleep(10 * time.Second)
			} else if len(h.aiImageQueue) > 0 {
				h.lock.Lock()
				imageId := ""
				if len(h.aiImageQueue) > 0 {
					imageId = h.aiImageQueue[0].ImageId
				}
				h.lock.Unlock()

				if imageId != "" {
					err := h.runImageDetection(imageId)
					if err != nil {
						h.context.App.Logger().Error(fmt.Sprintf("Error running image detection: %v", err))
						backoffTimeUntil := time.Now().Add(30 * time.Second)
						h.aiBackoffUntil = &backoffTimeUntil
					} else {
						h.lock.Lock()
						h.aiImageQueue = h.aiImageQueue[1:]
						h.lock.Unlock()
					}
				}
			} else {
				time.Sleep(250 * time.Millisecond)
			}
		}
	}()
}

func (h *HookExecutor) runImageDetection(imageId string) error {
	image, err := h.context.App.Dao().FindRecordById("images", imageId)
	if err != nil {
		return err
	}

	h.addDownloadUrls(image)

	objectIds := util.GetObjectIds(image.GetString("storageId"))

	downloadUrl, _, err := h.getDownloadUrl(512, objectIds[512])
	if err != nil {
		return err
	}

	projectId := image.GetString("project")
	project, err := h.context.App.Dao().FindRecordById("projects", projectId)
	if err != nil {
		return err
	}

	systemMessageString := project.GetString("aiSystemMessage")
	if systemMessageString == "" {
		return nil
	}

	client := openai.NewClient(config.Get().String("OPENAI_API_KEY"))

	systemMessage := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemMessageString,
	}

	userMessage := openai.ChatCompletionMessage{
		Role: openai.ChatMessageRoleUser,
		MultiContent: []openai.ChatMessagePart{
			{
				ImageURL: &openai.ChatMessageImageURL{
					URL: downloadUrl,
				},
				Type: openai.ChatMessagePartTypeImageURL,
			},
		},
	}

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				systemMessage,
				userMessage,
			},
		},
	)

	if err != nil {
		h.context.App.Logger().Error(fmt.Sprintf("OpenAI error running detection: %v", err))
		return err
	}

	h.context.App.Logger().Debug(fmt.Sprintf("ran detection for image '%s'. Total Tokens: %d", image.GetString("computedFileName"), resp.Usage.TotalTokens))
	for _, choice := range resp.Choices {
		tagText := choice.Message.Content
		if tagText == "none" {
			h.context.App.Logger().Debug(fmt.Sprintf("detection for image '%s' did not yield a car number ('none')", image.GetString("computedFileName")))
			break
		}

		h.context.App.Logger().Debug(fmt.Sprintf("detection for image '%s' yielded '%s'", image.GetString("computedFileName"), tagText))
		records, err := h.context.App.Dao().FindRecordsByExpr("image_tags", dbx.NewExp("project = {:project}", dbx.Params{"project": projectId}), dbx.NewExp("name = {:name}", dbx.Params{"name": tagText}))
		if err != nil {
			h.context.App.Logger().Error(fmt.Sprintf("error finding ai yielded tag '%s' for project '%s': %v", tagText, projectId, err))
			return err
		}

		if len(records) == 1 {
			imageTag := records[0]
			h.addTagToImage(image, imageTag, "inferred")
		} else {
			h.context.App.Logger().Debug(fmt.Sprintf("detection for image '%s' yielded '%s' but no tag was found", image.GetString("computedFileName"), tagText))
		}
		break
	}

	return nil
}
