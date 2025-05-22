package llm

import (
	"context"
	"fmt"
	"github.com/kk2simon/ghost-cli/tools"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"google.golang.org/genai"
)

type LLMConfig struct {
	Name    string
	APIType string // openai | gemini
	APIKey  string
	Host    string
	Model   string
}

func BuildLLMProvider(ctx context.Context, cfg LLMConfig) (LLMProvider, error) {
	switch cfg.APIType {
	case "gemini":
		// TODO move to NewGeminiLLMProvider
		geminiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  cfg.APIKey,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create gemini client: %v", err)
		}
		return &GeminiLLMProvider{client: geminiClient}, nil

	case "openaichat":
		// TODO move to NewOpenaiChatLLMProvider
		opts := []option.RequestOption{
			option.WithAPIKey(cfg.APIKey), // falls back to ENV if empty
		}
		if cfg.Host != "" {
			opts = append(opts, option.WithBaseURL(cfg.Host))
		}
		openaiClient := openai.NewClient(opts...)
		return &OpenaiChatLLMProvider{client: &openaiClient}, nil

	default:
		return nil, fmt.Errorf("unsupported LLM type: %s", cfg.APIType)
	}
}

type Prompt struct {
	System    string
	Developer string
	User      string
}

type LLMProvider interface {
	APIType() string // openaichat, openairesponse, gemini

	Chat(ctx context.Context, prompt Prompt, model string, tools []mcp.Tool, toolCaller tools.ToolCaller) (string, error)

	// Potentially separate stream method
	// StreamChat(...) (<-chan string, error)
}
