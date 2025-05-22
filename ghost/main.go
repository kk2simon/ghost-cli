package main

import (
	"context"
	"fmt"
	"os"

	"github.com/kk2simon/ghost-cli/cli" // Added for parsing command-line flags
	"github.com/kk2simon/ghost-cli/llm"
	"github.com/kk2simon/ghost-cli/tools"
)

func main() {
	ctx := context.Background()
	// Parse CLI flags using the new function from the cli package
	appFlags := cli.ParseFlags()
	cfg, err := ParseConfig(appFlags.CfgPath) // Pass appFlags.CfgPath
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse config: %v\n", err)
		os.Exit(1)
	}

	// Load prompt using the PromptFile from appFlags
	prompt, err := cli.ParsePromptFile(appFlags.PromptFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read prompt: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("prompt:", prompt)

	// Initialise MCP tools
	toolsRuntime, err := tools.InitializeMCP(ctx, cfg.Mcps)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize MCP: %v\n", err)
		os.Exit(1)
	}
	defer toolsRuntime.CloseFunc()

	// Pick an LLM – fall back to the first configured one, using LLMName and ModelName from appFlags
	var llmCfg llm.LLMConfig
	for _, c := range cfg.LLMs {
		if c.Name == appFlags.LLMName || appFlags.LLMName == "" {
			llmCfg = c
			break
		}
	}
	if llmCfg.APIType == "" {
		fmt.Fprintln(os.Stderr, "Error: no LLM configuration found")
		os.Exit(1)
	}

	modelToUse := llmCfg.Model
	if appFlags.ModelName != "" {
		modelToUse = appFlags.ModelName
	}

	llmProvider, err := llm.BuildLLMProvider(ctx, llmCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build LLM: %v\n", err)
		os.Exit(1)
	}

	resp, err := llmProvider.Chat(ctx, llm.Prompt{User: prompt}, modelToUse,
		toolsRuntime.Tools, toolsRuntime.Caller)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during gemini chat: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ChatResult:", resp)

}
