package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/kk2simon/ghost-cli/tools"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/openai/openai-go"
)

type OpenaiChatLLMProvider struct {
	client *openai.Client
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
			fmt.Println("Final response:", msg.Content)

			// Ask user for new prompt
			fmt.Print("> ")
			reader := bufio.NewReader(os.Stdin)
			userInput, err := reader.ReadString('\n')
			if err != nil {
				// Handle error, maybe return or continue
				fmt.Printf("Error reading user input: %v\n", err)
				return "", fmt.Errorf("error reading user input: %w", err)
			}

			userInput = strings.TrimSpace(userInput)
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

// Convert MCP tool schemas to the structures expected by the new SDK.
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
