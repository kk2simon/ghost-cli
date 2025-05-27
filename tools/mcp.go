package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/kk2simon/ghost-cli/base"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type McpConfig struct {
	Name    string
	Command string
	Env     []string
	Args    []string
}

type ToolCaller func(name string, arguments map[string]any) (*mcp.CallToolResult, error)

type Runtime struct {
	Tools     []mcp.Tool
	Caller    ToolCaller
	CloseFunc func()
}

func InitializeMCP(ctx context.Context, cfgs []McpConfig, logger *slog.Logger) (*Runtime, error) {
	mcpClients := make([]*client.Client, 0, len(cfgs))
	toolToClient := make(map[string]*client.Client)
	allTools := make([]mcp.Tool, 0)

	spinner := base.StartProgressSpinner("Initialize MCPs", len(cfgs))
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(cfgs))

	wg.Add(len(cfgs))
	for _, mcpCfg := range cfgs {
		go func(cfg McpConfig) {
			defer wg.Done()
			defer spinner.Incr()

			c, err := client.NewStdioMCPClient(cfg.Command, cfg.Env, cfg.Args...)
			if err != nil {
				errChan <- fmt.Errorf("Failed to create MCP client: %v", err)
				return
			}

			initRequest := mcp.InitializeRequest{}
			initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
			initRequest.Params.ClientInfo = mcp.Implementation{
				Name:    "ghost-cli",
				Version: "1.0.0",
			}

			initCtx, cancelInit := context.WithTimeout(ctx, 60*time.Second)
			defer cancelInit()
			_, err = c.Initialize(initCtx, initRequest)
			if err != nil {
				errChan <- fmt.Errorf("Failed to initialize MCP client %s: %v", cfg.Name, err)
				return
			}

			toolsRequest := mcp.ListToolsRequest{}
			toolsResp, err := c.ListTools(ctx, toolsRequest)
			if err != nil {
				errChan <- fmt.Errorf("Failed to list tools: %v", err)
				return
			}

			mu.Lock()
			mcpClients = append(mcpClients, c)
			for _, t := range toolsResp.Tools {
				toolToClient[t.Name] = c
				allTools = append(allTools, t)
			}
			mu.Unlock()
		}(mcpCfg)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		// Return the first error encountered
		return nil, <-errChan
	}

	closeFunc := func() {
		for _, c := range mcpClients {
			c.Close()
		}
	}

	toolCaller := func(name string, arguments map[string]any) (*mcp.CallToolResult, error) {
		b, err := json.MarshalIndent(arguments, "", "  ")
		if err != nil {
			return nil, err
		}

		color.Magenta("Confirm tool call:")
		fmt.Printf(`=======================
Name: %s
Arguments:
%v
=======================
Press Enter to continue, or type anything else to refuse:`, name, string(b))
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{mcp.TextContent{Text: "User refused tool call"}},
			}, nil
		}
		c, ok := toolToClient[name]
		if !ok {
			return &mcp.CallToolResult{
				Content: []mcp.Content{mcp.TextContent{Text: "Error: tool not found in any MCP client"}}, // TODO maybe, should return error
			}, nil
		}
		callReq := mcp.CallToolRequest{
			Request: mcp.Request{
				Method: "tools/call",
			},
		}
		callReq.Params.Name = name
		callReq.Params.Arguments = arguments
		callResult, err := c.CallTool(ctx, callReq)
		if err != nil {
			return callResult, err
		}
		tc, _ := callResult.Content[0].(mcp.TextContent)
		logger.Debug("Tool result", "result", tc.Text)

		{ // print tool call result, (TODO move to base pkg PrintInfoSummary)
			text := tc.Text
			textLen := len(text)
			words := strings.Fields(text) // Split into words
			wordCount := len(words)
			formattedText := ""
			const wordsThreshold = 20 // Threshold to decide if truncation is needed
			const wordsToShow = 10    // Number of words to show at the beginning and end

			if wordCount <= wordsThreshold {
				formattedText = text // Keep original text if short
			} else {
				// Take the first wordsToShow words
				firstWords := strings.Join(words[:wordsToShow], " ")
				// Take the last wordsToShow words
				lastWords := strings.Join(words[wordCount-wordsToShow:], " ")
				formattedText = firstWords + " ... " + lastWords
			}
			fmt.Printf("Tool result (char len: %d, word count: %d): %s\n", textLen, wordCount, formattedText)
		}

		return callResult, nil
	}

	return &Runtime{
		Tools:     allTools,
		Caller:    toolCaller,
		CloseFunc: closeFunc,
	}, nil
}
