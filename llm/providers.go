package llm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kk2simon/ghost-cli/tools"

	"github.com/mark3labs/mcp-go/mcp"
)

type LLMConfig struct {
	Name    string
	APIType string // openai | gemini
	APIKey  string
	Host    string
	Model   string

	// OpenaiResponse struct {
	// 	DeleteConversation bool // whether delete conversation after chat
	// }
}

func BuildLLMProvider(ctx context.Context, cfg LLMConfig, logger *slog.Logger) (LLMProvider, error) {
	switch cfg.APIType {
	case "gemini":
		return NewGeminiLLMProvider(ctx, cfg, logger)

	case "openaichat":
		return NewOpenaiChatLLMProvider(cfg, logger)

	case "openairesponse":
		return NewOpenaiResponseLLMProvider(cfg, logger)

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
