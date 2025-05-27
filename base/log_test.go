package base

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestSpinner(t *testing.T) {
	s := StartProgressSpinner("Init MCPs", 5)
	defer s.Stop()
	for i := 0; i < 5; i++ {
		time.Sleep(1 * time.Second)
		s.Incr()
	}

	s1 := StartProgressSpinner("Wait LLM response", 0)
	time.Sleep(5 * time.Second)
	s1.Stop()
}

func TestSlog(t *testing.T) {
	opts := &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Check if the attribute key is slog.MessageKey (which is "msg")
			if a.Key == slog.MessageKey {
				// Change the key to an empty string
				a.Key = ""
			}
			return a
		},
		Level: slog.LevelDebug, // Optional: set a specific log level
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	logger.Info("test")
}
