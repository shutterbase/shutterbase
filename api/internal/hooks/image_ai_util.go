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
			time.Sleep(250 * time.Millisecond)

			if h.aiBackoffUntil != nil && time.Now().Before(*h.aiBackoffUntil) {
				h.context.App.Logger().Warn(fmt.Sprintf("AI backoff in effect until %v. Waiting for 30 seconds", h.aiBackoffUntil))
				time.Sleep(10 * time.Second)
				continue
			}

			h.lock.Lock()
			imageId := ""
			if len(h.aiImageQueue) > 0 {
				imageId = h.aiImageQueue[0].ImageId
			}
			h.lock.Unlock()

			if imageId == "" {
				continue
			}

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

	logRateLimits := func(rateLimits openai.RateLimitHeaders) {
		h.context.App.Logger().Info(fmt.Sprintf("[OpenAI] [%s]: rate limits: [Remaining Requests: %d | reset: %s] [Remaining Tokens: %d | reset: %s]",
			image.GetString("computedFileName"),
			rateLimits.RemainingRequests, rateLimits.ResetRequests.String(),
			rateLimits.RemainingTokens, rateLimits.ResetTokens.String()))
	}

	if err != nil {
		h.context.App.Logger().Error(fmt.Sprintf("[OpenAI] [%s]: error running detection: %v", image.GetString("computedFileName"), err))
		logRateLimits(resp.GetRateLimitHeaders())
		return err
	}

	h.context.App.Logger().Debug(fmt.Sprintf("[OpenAI] [%s]: ran detection. Total Tokens: %d", image.GetString("computedFileName"), resp.Usage.TotalTokens))
	logRateLimits(resp.GetRateLimitHeaders())
	for _, choice := range resp.Choices {
		tagText := choice.Message.Content
		if tagText == "none" {
			h.context.App.Logger().Debug(fmt.Sprintf("[OpenAI] [%s]: detection did not yield a car number ('none')", image.GetString("computedFileName")))
			break
		}

		h.context.App.Logger().Debug(fmt.Sprintf("[OpenAI] [%s]: detection yielded '%s'", image.GetString("computedFileName"), tagText))
		records, err := h.context.App.Dao().FindRecordsByExpr("image_tags", dbx.NewExp("project = {:project}", dbx.Params{"project": projectId}), dbx.NewExp("name = {:name}", dbx.Params{"name": tagText}))
		if err != nil {
			h.context.App.Logger().Error(fmt.Sprintf("[OpenAI] [%s]: error finding ai yielded tag '%s' for project '%s': %v", image.GetString("computedFileName"), tagText, projectId, err))
			return err
		}

		if len(records) == 1 {
			imageTag := records[0]
			h.addTagToImage(image, imageTag, "inferred")
		} else {
			h.context.App.Logger().Debug(fmt.Sprintf("[OpenAI] [%s]: detection yielded '%s' but no tag was found", image.GetString("computedFileName"), tagText))
		}
		break
	}

	return nil
}
