package cli

import "flag"

// Flags holds the command-line arguments.
type Flags struct {
	PromptFile string
	LLMName    string
	CfgPath    string
	ModelName  string
}

// ParseFlags parses the command-line arguments and returns them in a Flags struct.
func ParseFlags() *Flags {
	promptFile := flag.String("p", "prompt.md", "prompt template file")
	llmName := flag.String("l", "", "LLM to use (openai|gemini)")
	cfgPath := flag.String("c", "", "path to config file")
	modelName := flag.String("m", "", "LLM model to use")
	flag.Parse()

	return &Flags{
		PromptFile: *promptFile,
		LLMName:    *llmName,
		CfgPath:    *cfgPath,
		ModelName:  *modelName,
	}
}
