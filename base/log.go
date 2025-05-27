package base

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/lmittmann/tint"
)

// defaultLogFile returns the platform-specific default log file path.
func defaultLogFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	switch runtime.GOOS {
	case "windows":
		// Windows: %LOCALAPPDATA%\ghost\logs\ghost.log
		// Fall back to %USERPROFILE%\AppData\Local if LOCALAPPDATA is not set
		appData := os.Getenv("LOCALAPPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(appData, "ghost", "logs", "ghost.log"), nil

	case "darwin":
		// macOS: ~/Library/Logs/ghost/ghost.log
		return filepath.Join(home, "Library", "Logs", "ghost", "ghost.log"), nil

	default: // Unix-like
		// Try XDG_STATE_HOME first, then XDG_CACHE_HOME, then ~/.local/state
		stateHome := os.Getenv("XDG_STATE_HOME")
		if stateHome == "" {
			stateHome = os.Getenv("XDG_CACHE_HOME")
			if stateHome == "" {
				stateHome = filepath.Join(home, ".local", "state")
			}
		}
		return filepath.Join(stateHome, "ghost", "logs", "ghost.log"), nil
	}
}

// InitLogger initializes a new logger that writes to a file and returns it
// along with a closure function to close the opened log file.
// The log file path is determined by LogFilePath(cfgPath).
// The level parameter sets the minimum log level (debug, info, warn, error).
func InitLogger(cfgPath string, level string) (*slog.Logger, func(), error) {
	// Get the log file path
	logPath, err := LogFilePath(cfgPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get log file path: %w", err)
	}

	// Create or open the log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Parse the log level
	var lv slog.Level
	err = lv.UnmarshalText([]byte(level))
	if err != nil {
		// Close the file if level parsing fails after opening the file
		logFile.Close()
		return nil, nil, fmt.Errorf("invalid log level: %s", level)
	}

	// Create a text handler that writes to the log file
	// Using slog.NewTextHandler for file output to avoid ANSI escape codes from tint
	handler := tint.NewHandler(logFile, &tint.Options{
		Level: lv,
	})

	// Create the logger
	logger := slog.New(handler)

	// Log the initialization message
	logger.Info("Logger initialized", "level", level, "path", logPath)

	// Return the logger and a function to close the log file
	return logger, func() { logFile.Close() }, nil
}

func LogFilePath(cfgPath string) (string, error) {
	// If cfgPath is provided, use it directly
	if cfgPath != "" {
		return cfgPath, nil
	}

	// Get platform-specific log file path
	logPath, err := defaultLogFile()
	if err != nil {
		return "", fmt.Errorf("failed to determine log file path: %w", err)
	}

	// Create the log directory if it doesn't exist
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory: %w", err)
	}

	return logPath, nil
}
