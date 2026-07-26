package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/mxcd/go-config/config"
	openai "github.com/sashabaranov/go-openai"

	"github.com/shutterbase/shutterbase/pkg/aiserver"
)

// InferenceRequest carries everything a provider may need. The simple
// providers (stub, openai) only read ImageURL + Prompt; the http provider
// forwards the full context so the AI server can prime itself and constrain
// its output to AvailableTags.
type InferenceRequest struct {
	ImageID       string
	ImageURL      string
	ProjectID     string
	ProjectName   string
	Prompt        string
	AvailableTags []string
	CapturedAt    *time.Time
	Author        string
}

// InferredTag is one tag with the provider's confidence (1.0 when the provider
// has no notion of confidence).
type InferredTag struct {
	Name       string
	Confidence float64
}

// ImageInference is the single seam the AI tagging service talks to.
// Provider-specific transport (OpenAI, OpenRouter, the aiserver contract)
// lives behind this; the service stays provider-agnostic.
type ImageInference interface {
	Infer(ctx context.Context, req InferenceRequest) ([]InferredTag, error)
}

// NewInference selects an implementation from AI_PROVIDER. Unknown providers are
// a config error, not a silent fallback. Model and key come from config — the
// model is never hardcoded (SPEC §S6).
func NewInference() (ImageInference, error) {
	switch provider := config.Get().String("AI_PROVIDER"); provider {
	case "", "stub":
		return &StubInference{}, nil
	case "openai":
		return newOpenAIInference(""), nil
	case "openrouter":
		// OpenRouter speaks the OpenAI wire protocol, so go-openai with a base
		// URL override is the real client — no stub needed.
		return newOpenAIInference("https://openrouter.ai/api/v1"), nil
	case "http":
		// The generic AI-server contract (pkg/aiserver), e.g. fsai.
		return &HTTPInference{
			Client: aiserver.NewClient(config.Get().String("AI_HTTP_ENDPOINT"), config.Get().String("AI_API_KEY")),
		}, nil
	default:
		return nil, errors.New("unknown AI_PROVIDER: " + provider)
	}
}

// StubInference is deterministic. Tests inject Tags to drive a known result;
// with Tags nil it echoes the prompt as a single tag (dev no-op that never
// matches a real project tag, so it produces no assignments).
type StubInference struct {
	Tags []string
}

func (s *StubInference) Infer(_ context.Context, req InferenceRequest) ([]InferredTag, error) {
	names := s.Tags
	if names == nil {
		names = []string{req.Prompt}
	}
	tags := make([]InferredTag, 0, len(names))
	for _, n := range names {
		tags = append(tags, InferredTag{Name: n, Confidence: 1})
	}
	return tags, nil
}

// openAIInference ports the old hook's go-openai usage: system prompt + the
// image URL as a single vision message. baseURL "" = OpenAI; set it for any
// OpenAI-compatible gateway (OpenRouter).
type openAIInference struct {
	model   string
	apiKey  string
	baseURL string
}

func newOpenAIInference(baseURL string) *openAIInference {
	return &openAIInference{
		model:   config.Get().String("AI_MODEL"),
		apiKey:  config.Get().String("AI_API_KEY"),
		baseURL: baseURL,
	}
}

func (o *openAIInference) Infer(ctx context.Context, req InferenceRequest) ([]InferredTag, error) {
	cfg := openai.DefaultConfig(o.apiKey)
	if o.baseURL != "" {
		cfg.BaseURL = o.baseURL
	}
	client := openai.NewClientWithConfig(cfg)

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: o.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: req.Prompt},
			{Role: openai.ChatMessageRoleUser, MultiContent: []openai.ChatMessagePart{{
				Type:     openai.ChatMessagePartTypeImageURL,
				ImageURL: &openai.ChatMessageImageURL{URL: req.ImageURL},
			}}},
		},
	})
	if err != nil {
		return nil, err
	}
	tags := make([]InferredTag, 0, len(resp.Choices))
	for _, choice := range resp.Choices {
		if t := strings.TrimSpace(choice.Message.Content); t != "" {
			tags = append(tags, InferredTag{Name: t, Confidence: 1})
		}
	}
	return tags, nil
}

// HTTPInference speaks the shutterbase AI-server contract. The full request
// context (project priming incl. the allowed tag vocabulary) rides along with
// every ingest, so the server needs no separate priming call to stay current.
type HTTPInference struct {
	Client *aiserver.Client
}

func (h *HTTPInference) Infer(ctx context.Context, req InferenceRequest) ([]InferredTag, error) {
	resp, err := h.Client.Ingest(ctx, req.ProjectID, aiserver.IngestRequest{
		Project: aiserver.Project{
			ID:     req.ProjectID,
			Name:   req.ProjectName,
			Prompt: req.Prompt,
			Tags:   req.AvailableTags,
		},
		ImageRef:   req.ImageID,
		ImageURL:   req.ImageURL,
		CapturedAt: req.CapturedAt,
		Author:     req.Author,
	})
	if err != nil {
		return nil, err
	}
	tags := make([]InferredTag, 0, len(resp.Tags))
	for _, t := range resp.Tags {
		tags = append(tags, InferredTag{Name: t.Name, Confidence: t.Confidence})
	}
	return tags, nil
}
