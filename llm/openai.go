package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/fatih/color"
	"github.com/kk2simon/ghost-cli/cli"
	"github.com/kk2simon/ghost-cli/tools"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type OpenaiChatLLMProvider struct {
	client *openai.Client
	logger *slog.Logger
}

func (o *OpenaiChatLLMProvider) APIType() string {
	return "openaichat"
}

// Chat-loop using the official SDK + tool calling.
func (o *OpenaiChatLLMProvider) Chat(
	ctx context.Context, prompt Prompt, model string, tools []mcp.Tool, toolCaller tools.ToolCaller) (string, error) {

	openaiTools, err := buildOpenAITools(tools)
	if err != nil {
		return "", err
	}

	// Initial payload
	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(model),
		Messages: []openai.ChatCompletionMessageParamUnion{openai.UserMessage(prompt.User)},
		Tools:    openaiTools,
	}

	for {
		completion, err := o.client.Chat.Completions.New(ctx, params)
		if err != nil {
			return "", err
		}
		if len(completion.Choices) == 0 {
			// TODO unfriendly to normal user
			return "", fmt.Errorf("no choices")
		}

		msg := completion.Choices[0].Message

		// No tool calls → conversation finished
		if len(msg.ToolCalls) == 0 {
			color.Cyan("LLM:" + msg.Content)

			// Ask user for new prompt
			userInput, err := cli.PromptUser()
			if err != nil {
				return "", fmt.Errorf("error reading user input: %w", err)
			}
			// If user input is empty or exit command, break the loop
			if userInput == "" || userInput == "exit" { // Define an exit command
				return "", nil // Return nil error to indicate successful chat end
			}
			// Append user message and continue chat loop
			params.Messages = append(params.Messages, openai.UserMessage(userInput))
			continue
		}

		// Record assistant message
		params.Messages = append(params.Messages, msg.ToParam())

		// Handle every tool call in this turn
		for _, call := range msg.ToolCalls {
			var args map[string]any
			if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
				return "", fmt.Errorf("tool %s: bad args: %w", call.Function.Name, err)
			}

			res, err := toolCaller(call.Function.Name, args)
			if err != nil {
				return "", fmt.Errorf("tool %s failed: %w", call.Function.Name, err)
			}

			toolOutput := ""
			if len(res.Content) > 0 {
				if tc, ok := res.Content[0].(mcp.TextContent); ok {
					toolOutput = tc.Text
				}
			}

			// Feed tool response back to the model
			params.Messages = append(params.Messages, openai.ToolMessage(toolOutput, call.ID))
		}
	}

	return "", nil
}

func NewOpenaiChatLLMProvider(cfg LLMConfig, logger *slog.Logger) (*OpenaiChatLLMProvider, error) {
	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey), // falls back to ENV if empty
	}
	if cfg.Host != "" {
		opts = append(opts, option.WithBaseURL(cfg.Host))
	}
	openaiClient := openai.NewClient(opts...)
	return &OpenaiChatLLMProvider{client: &openaiClient, logger: logger}, nil
}
func buildOpenAITools(tools []mcp.Tool) ([]openai.ChatCompletionToolParam, error) {
	out := make([]openai.ChatCompletionToolParam, 0, len(tools))
	for _, t := range tools {
		// Marshal → unmarshal dance to get a plain `map[string]any`
		raw, err := json.Marshal(t.InputSchema)
		if err != nil {
			return nil, err
		}
		var params map[string]any
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, err
		}

		out = append(out, openai.ChatCompletionToolParam{
			Function: openai.FunctionDefinitionParam{
				Name:        t.Name,
				Description: openai.String(t.Description),
				Parameters:  openai.FunctionParameters(params),
			},
		})
	}
	return out, nil
}
