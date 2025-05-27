package main

import (
	"fmt"
	"os"
	"path"

	"github.com/kk2simon/ghost-cli/llm"
	"github.com/kk2simon/ghost-cli/tools"

	"github.com/BurntSushi/toml"
)

type Config struct {
	LogLevel string
	LogPath  string

	LLMs []llm.LLMConfig
	Mcps []tools.McpConfig
}

func ParseConfig(flagCfgPath string) (Config, error) {

	tryPaths := []string{}
	if envCfgPath := os.Getenv("GHOST_CONFIG_TOML_FILE"); envCfgPath != "" {
		tryPaths = append(tryPaths, envCfgPath)
	}
	if flagCfgPath != "" { // Add this block
		tryPaths = append(tryPaths, flagCfgPath)
	}

	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	tryPaths = append(tryPaths,
		path.Join(cwd, ".ghost", "config.toml"),
		path.Join(home, ".ghost", "config.toml"),
		path.Join(home, ".config", "ghost", "config.toml"),
	)

	var f *os.File
	var err error
	for _, path := range tryPaths {
		f, err = os.Open(path)
		if err == nil {
			fmt.Println("Reading config file:", path)
			break
		}
	}
	if f == nil {
		return Config{}, fmt.Errorf("Failed to find config file in any known location. Tried: %v", tryPaths)
	}
	defer f.Close()
	var cfg Config
	if _, err := toml.NewDecoder(f).Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("Failed to decode config(%s): %v", f.Name(), err)
	}
	return cfg, nil
}
