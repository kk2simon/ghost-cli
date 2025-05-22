package cli

import "flag"

// Flags holds the command-line arguments.
type Flags struct {
	PromptFile string
	LLMName    string
	CfgPath    string // Add this field
	ModelName string
}

// ParseFlags parses the command-line arguments and returns them in a Flags struct.
func ParseFlags() *Flags {
	promptFile := flag.String("p", "prompt.tpl", "prompt template file")
	flag.StringVar(promptFile, "prompt", "prompt.tpl", "prompt template file (long)")
	llmName := flag.String("l", "", "LLM to use (openai|gemini)")
	flag.StringVar(llmName, "llm", "", "LLM to use (openai|gemini) (long)")
	cfgPath := flag.String("c", "", "path to config file")
	flag.StringVar(cfgPath, "config", "", "path to config file (long)")
	modelName := flag.String("m", "", "LLM model to use")
	flag.StringVar(modelName, "model", "", "LLM model to use (long)")
	flag.Parse()

	return &Flags{
		PromptFile: *promptFile,
		LLMName:    *llmName,
		CfgPath:    *cfgPath, // Add this field
		ModelName: *modelName,
	}
}
