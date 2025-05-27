package main

import (
	"context"
	"fmt"
	"os"

	"github.com/kk2simon/ghost-cli/base"
	"github.com/kk2simon/ghost-cli/cli"
	"github.com/kk2simon/ghost-cli/llm"
	"github.com/kk2simon/ghost-cli/tools"
)

func main() {
	ctx := context.Background()
	appFlags := cli.ParseFlags()
	cfg, err := ParseConfig(appFlags.CfgPath)
	exitIfErr(err, "Failed to parse config")

	logger, closeLogger, err := base.InitLogger(cfg.LogPath, cfg.LogLevel)
	exitIfErr(err, "Failed to initialize logger")
	defer closeLogger()

	logger.Info("Read prompt file", "path", appFlags.PromptFile)

	prompt, err := cli.ParsePromptFile(appFlags.PromptFile)
	exitIfErr(err, "Failed to read prompt")
	logger.Debug("prompt got", "prompt", prompt)

	toolsRuntime, err := tools.InitializeMCP(ctx, cfg.Mcps, logger)
	exitIfErr(err, "Failed to initialize MCP")
	defer toolsRuntime.CloseFunc()

	var llmCfg llm.LLMConfig
	for _, c := range cfg.LLMs {
		if c.Name == appFlags.LLMName || appFlags.LLMName == "" {
			llmCfg = c
			break
		}
	}
	if llmCfg.APIType == "" {
		exitIfErr(fmt.Errorf("no LLM configuration found"), "")
	}

	modelToUse := llmCfg.Model
	if appFlags.ModelName != "" {
		modelToUse = appFlags.ModelName
	}

	llmProvider, err := llm.BuildLLMProvider(ctx, llmCfg, logger)
	exitIfErr(err, "Failed to build LLM")

	resp, err := llmProvider.Chat(ctx, llm.Prompt{User: prompt}, modelToUse,
		toolsRuntime.Tools, toolsRuntime.Caller)
	exitIfErr(err, "Error during gemini chat")
	logger.Info("Chat done", "Result", resp)

}

func exitIfErr(err error, msg string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s: %v\\n", msg, err)
		os.Exit(1)
	}
}
