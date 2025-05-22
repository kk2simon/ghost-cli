package llm

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kk2simon/ghost-cli/tools"

	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/genai"
)

type GeminiLLMProvider struct {
	client *genai.Client
}

func (g *GeminiLLMProvider) APIType() string {
	return "gemini"
}

func (g *GeminiLLMProvider) Chat(ctx context.Context,
	prompt Prompt, model string, tools []mcp.Tool,
	toolCaller tools.ToolCaller) (string, error) {

	genaiTools, err := toolsToGoogle(tools)
	if err != nil {
		return "", err
	}

	chats, err := g.client.Chats.Create(ctx, model, &genai.GenerateContentConfig{
		Tools: genaiTools,
	}, nil)

	parts := []genai.Part{
		{Text: prompt.User},
	}
	genContentResp, err := chats.SendMessage(ctx, parts...)
	if err != nil {
		return "", err
	}

	for {
		functionCalls := genContentResp.FunctionCalls()
		if len(functionCalls) == 0 {
			// Get the response text
			respText := ""
			if len(genContentResp.Candidates) > 0 && len(genContentResp.Candidates[0].Content.Parts) > 0 {
				respText = genContentResp.Candidates[0].Content.Parts[0].Text
			}

			// Print the response text
			fmt.Println(respText)

			// Prompt user for input
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

			// Create a new message part for user input
			parts = []genai.Part{
				{Text: userInput},
			}

			// Send the new user message and continue the loop
			genContentResp, err = chats.SendMessage(ctx, parts...)
			if err != nil {
				return "", err
			}

			// Continue the loop to process the new response
			continue
		}
		responses := make([]genai.Part, 0, len(functionCalls))
		for _, call := range functionCalls {
			out, err := toolCaller(call.Name, call.Args)
			if err != nil {
				return "", err
			}
			text := ""
			if len(out.Content) > 0 {
				if tc, ok := out.Content[0].(mcp.TextContent); ok {
					text = tc.Text
				}
			}
			responses = append(responses, genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					ID:   call.ID,
					Name: call.Name,
					Response: map[string]any{
						"output": text,
					},
				},
			})
		}
		genContentResp, err = chats.SendMessage(ctx, responses...)
		if err != nil {
			return "", err
		}
	}

	return "", nil
}

// copied from https://github.com/kylecarbs/aisdk-go/blob/main/google.go#L20
func toolsToGoogle(tools []mcp.Tool) ([]*genai.Tool, error) {
	functionDeclarations := []*genai.FunctionDeclaration{}

	var propertyToSchema func(property map[string]any) (*genai.Schema, error)
	propertyToSchema = func(property map[string]any) (*genai.Schema, error) {
		schema := &genai.Schema{
			Properties: make(map[string]*genai.Schema),
		}

		typeRaw, ok := property["type"]
		if ok {
			typ, ok := typeRaw.(string)
			if !ok {
				return nil, fmt.Errorf("type is not a string: %T", typeRaw)
			}
			schema.Type = genai.Type(strings.ToUpper(typ))
		}

		descriptionRaw, ok := property["description"]
		if ok {
			description, ok := descriptionRaw.(string)
			if !ok {
				return nil, fmt.Errorf("description is not a string: %T", descriptionRaw)
			}
			schema.Description = description
		}

		propertiesRaw, ok := property["properties"]
		if ok {
			properties, ok := propertiesRaw.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("properties is not a map[string]any: %T", propertiesRaw)
			}
			for key, value := range properties {
				propMap, ok := value.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("property %q is not a map[string]any: %T", key, value)
				}
				subschema, err := propertyToSchema(propMap)
				if err != nil {
					return nil, fmt.Errorf("property %q has non-object properties: %w", key, err)
				}
				schema.Properties[key] = subschema
			}
		}

		itemsRaw, ok := property["items"]
		if ok {
			items, ok := itemsRaw.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("items is not a map[string]any: %T", itemsRaw)
			}
			subschema, err := propertyToSchema(items)
			if err != nil {
				return nil, fmt.Errorf("items has non-object properties: %w", err)
			}
			schema.Items = subschema
		}

		return schema, nil
	}

	for _, tool := range tools {
		var schema *genai.Schema
		if tool.InputSchema.Properties != nil {
			schema = &genai.Schema{
				Type:       genai.TypeObject,
				Properties: make(map[string]*genai.Schema),
				Required:   tool.InputSchema.Required,
			}

			for key, value := range tool.InputSchema.Properties {
				propMap, ok := value.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("property %q is not a map[string]any: %T", key, value)
				}
				subschema, err := propertyToSchema(propMap)
				if err != nil {
					return nil, fmt.Errorf("property %q has non-object properties: %w", key, err)
				}
				schema.Properties[key] = subschema
			}
		}

		functionDeclarations = append(functionDeclarations, &genai.FunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  schema,
		})
	}
	return []*genai.Tool{{
		FunctionDeclarations: functionDeclarations,
	}}, nil
}
