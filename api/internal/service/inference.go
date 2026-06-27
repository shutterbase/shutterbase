package service

import (
	"context"
	"errors"
	"strings"

	"github.com/mxcd/go-config/config"
	openai "github.com/sashabaranov/go-openai"
)

// ImageInference is the single seam the AI tagging service talks to. An impl
// takes a presigned image URL plus the project's system prompt and returns the
// tag names it inferred. Provider-specific transport (OpenAI, OpenRouter, a
// future REST gateway) lives behind this; the service stays provider-agnostic.
type ImageInference interface {
	Infer(ctx context.Context, imageURL, systemPrompt string) ([]string, error)
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
		// ponytail: upgrade to the aikido Go lib if/when it lands as an
		// importable module; the OpenAI-compatible path covers it until then.
		return newOpenAIInference("https://openrouter.ai/api/v1"), nil
	case "http":
		return &HTTPInference{}, nil
	default:
		return nil, errors.New("unknown AI_PROVIDER: " + provider)
	}
}

// StubInference is deterministic. Tests inject Tags to drive a known result;
// with Tags nil it echoes the system prompt as a single tag (dev no-op that
// never matches a real project tag, so it produces no assignments).
type StubInference struct {
	Tags []string
}

func (s *StubInference) Infer(_ context.Context, _ string, systemPrompt string) ([]string, error) {
	if s.Tags != nil {
		return s.Tags, nil
	}
	return []string{systemPrompt}, nil
}

// openAIInference ports the old hook's go-openai usage: system prompt + the
// image URL as a single vision message. baseURL "" = OpenAI; set it for any
// OpenAI-compatible gateway (OpenRouter).
type openAIInference struct {
	model  string
	apiKey string
	baseURL string
}

func newOpenAIInference(baseURL string) *openAIInference {
	return &openAIInference{
		model:   config.Get().String("AI_MODEL"),
		apiKey:  config.Get().String("AI_API_KEY"),
		baseURL: baseURL,
	}
}

func (o *openAIInference) Infer(ctx context.Context, imageURL, systemPrompt string) ([]string, error) {
	cfg := openai.DefaultConfig(o.apiKey)
	if o.baseURL != "" {
		cfg.BaseURL = o.baseURL
	}
	client := openai.NewClientWithConfig(cfg)

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: o.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, MultiContent: []openai.ChatMessagePart{{
				Type:     openai.ChatMessagePartTypeImageURL,
				ImageURL: &openai.ChatMessageImageURL{URL: imageURL},
			}}},
		},
	})
	if err != nil {
		return nil, err
	}
	tags := make([]string, 0, len(resp.Choices))
	for _, choice := range resp.Choices {
		if t := strings.TrimSpace(choice.Message.Content); t != "" {
			tags = append(tags, t)
		}
	}
	return tags, nil
}

// HTTPInference is a placeholder for a future generic REST inference gateway.
// ponytail: not built — no consumer yet. Construct it (AI_PROVIDER=http) and
// fill Infer when a concrete gateway exists; selecting it today is a clear error.
type HTTPInference struct {
	Endpoint string
}

func (h *HTTPInference) Infer(_ context.Context, _ string, _ string) ([]string, error) {
	return nil, errors.New("http inference provider not implemented")
}
