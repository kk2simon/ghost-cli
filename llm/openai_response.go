package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/fatih/color"
	"github.com/kk2simon/ghost-cli/base"
	"github.com/kk2simon/ghost-cli/cli"
	"github.com/kk2simon/ghost-cli/tools"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"
)

type OpenaiResponseLLMProvider struct {
	client *openai.Client
	logger *slog.Logger
}

func (o *OpenaiResponseLLMProvider) APIType() string { return "openairesponse" }

// Chat performs conversation with per-session conversationID passed in and returned
func (o *OpenaiResponseLLMProvider) Chat(
	ctx context.Context,
	prompt Prompt,
	model string,
	toolsDesc []mcp.Tool,
	toolCaller tools.ToolCaller,
) (string, error) {
	openaiTools, err := buildOpenAIResponsesTools(toolsDesc)
	if err != nil {
		return "", err
	}

	var localConversationID string
	chatInput := responses.ResponseNewParamsInputUnion{
		OfString: openai.String(prompt.User),
	}

	for {
		params := responses.ResponseNewParams{
			Model: shared.ResponsesModel(model),
			Input: chatInput,
			Store: openai.Bool(true),
			Tools: openaiTools,
		}

		if localConversationID != "" {
			params.PreviousResponseID = param.NewOpt(localConversationID)
		}

		resp, err := o.client.Responses.New(ctx, params)
		if err != nil {
			return "", err
		}

		localConversationID = resp.ID
		o.logger.Info("Conversation ID", "ID", resp.ID)

		o.logger.Debug(base.MustPrettyJSON(resp.Output), "text", "respJSON")

		// chat not completed, try to process outputs
		tmpInput := responses.ResponseNewParamsInputUnion{}

		for _, out := range resp.Output {
			switch out.Type {
			case "message":
				fmt.Println("Message:\n", out.Content[0].Text)
			case "function_call":
				call := out.AsFunctionCall()

				var args map[string]any
				if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
					return "", fmt.Errorf("tool %s: bad args: %w", call.Name, err)
				}

				toolCallResp, err := toolCaller(call.Name, args)
				if err != nil {
					return "", fmt.Errorf("tool %s failed: %w", call.Name, err)
				}

				toolOutput := ""
				if len(toolCallResp.Content) > 0 {
					if tc, ok := toolCallResp.Content[0].(mcp.TextContent); ok {
						toolOutput = tc.Text
					}
				}

				o.logger.Debug(base.MustPrettyJSON(call), "info", "callJSON")

				tmpInput.OfInputItemList = append(tmpInput.OfInputItemList, responses.ResponseInputItemParamOfFunctionCallOutput(
					call.CallID, toolOutput,
				))

			default:
				return "", fmt.Errorf("unhandled output type: %s", out.Type)
			}

			chatInput = tmpInput
		}

		if len(tmpInput.OfInputItemList) == 0 && resp.Status == responses.ResponseStatusCompleted {
			color.Cyan("LLM: %s", resp.OutputText())

			usrInput, err := cli.PromptUser()
			if err != nil {
				return "", fmt.Errorf("error reading user input: %w", err)
			}
			if usrInput == "" || usrInput == "exit" {
				return "", nil
			}
			chatInput = responses.ResponseNewParamsInputUnion{
				OfString: openai.String(usrInput),
			}
			continue
		}
	}
}

func buildOpenAIResponsesTools(tools []mcp.Tool) ([]responses.ToolUnionParam, error) {
	out := make([]responses.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		raw, err := json.Marshal(t.InputSchema)
		if err != nil {
			return nil, err
		}
		var schema map[string]any
		if err := json.Unmarshal(raw, &schema); err != nil {
			return nil, err
		}

		out = append(out, responses.ToolParamOfFunction(t.Name, schema, false))
	}
	// fmt.Println(base.MustPrettyJSON(out))
	return out, nil
}

func NewOpenaiResponseLLMProvider(cfg LLMConfig, logger *slog.Logger) (*OpenaiResponseLLMProvider, error) {
	opts := []option.RequestOption{option.WithAPIKey(cfg.APIKey)}
	if cfg.Host != "" {
		opts = append(opts, option.WithBaseURL(cfg.Host))
	}
	client := openai.NewClient(opts...)
	return &OpenaiResponseLLMProvider{client: &client, logger: logger}, nil
}
