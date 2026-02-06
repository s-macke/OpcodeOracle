package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"google.golang.org/genai"
)

// Default models for each provider.
const (
	DefaultAnthropicModel = "claude-opus-4-6"
	DefaultOpenAIModel    = "gpt-5.2-codex"
	DefaultGeminiModel    = "gemini-3"
)

// NewChatModel creates a new ChatModel based on the provider configuration.
func NewChatModel(ctx context.Context, cfg *Config) (model.ToolCallingChatModel, error) {
	switch cfg.Provider {
	case ProviderAnthropic:
		return newAnthropicModel(ctx, cfg)
	case ProviderOpenAI:
		return newOpenAIModel(ctx, cfg)
	case ProviderGemini:
		return newGeminiModel(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
	}
}

// newAnthropicModel creates an Anthropic (Claude) ChatModel.
func newAnthropicModel(ctx context.Context, cfg *Config) (model.ToolCallingChatModel, error) {
	modelName := cfg.Model
	if modelName == "" {
		modelName = DefaultAnthropicModel
	}

	temperature := float32(cfg.Temperature)

	return claude.NewChatModel(ctx, &claude.Config{
		APIKey:      cfg.APIKey,
		Model:       modelName,
		MaxTokens:   cfg.MaxTokens,
		Temperature: &temperature,
	})
}

// newOpenAIModel creates an OpenAI ChatModel.
func newOpenAIModel(ctx context.Context, cfg *Config) (model.ToolCallingChatModel, error) {
	modelName := cfg.Model
	if modelName == "" {
		modelName = DefaultOpenAIModel
	}

	temperature := float32(cfg.Temperature)
	maxTokens := cfg.MaxTokens

	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:              cfg.APIKey,
		BaseURL:             cfg.BaseURL,
		Model:               modelName,
		MaxCompletionTokens: &maxTokens,
		Temperature:         &temperature,
	})
}

// newGeminiModel creates a Google Gemini ChatModel.
func newGeminiModel(ctx context.Context, cfg *Config) (model.ToolCallingChatModel, error) {
	modelName := cfg.Model
	if modelName == "" {
		modelName = DefaultGeminiModel
	}

	temperature := float32(cfg.Temperature)
	maxTokens := cfg.MaxTokens

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return gemini.NewChatModel(ctx, &gemini.Config{
		Client:      client,
		Model:       modelName,
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
	})
}
