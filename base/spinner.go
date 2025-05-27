package base

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

// var spinners = []string{"∙∙∙", "●∙∙", "∙●∙", "∙∙●", "∙∙∙"}

var spinners = []string{"   ", ".  ", ".. ", "..."}

const defaultSpinnerInterval = 800 * time.Millisecond

func StartProgressSpinner(taskName string, total int) *Spinner {
	ctx, cancel := context.WithCancel(context.Background())
	s := Spinner{
		ctx:      ctx,
		cancel:   cancel,
		total:    total,
		finished: 0,
		taskName: taskName,
	}

	if term.IsTerminal(int(os.Stdout.Fd())) { // only tick update spinner when in terminal
		s.update(s.frame, s.finished)

		go func() {
			ticker := time.NewTicker(defaultSpinnerInterval)
			defer ticker.Stop()
			for {
				select {
				case <-s.ctx.Done():
					return
				case <-ticker.C:
					s.mu.Lock()
					s.frame++
					currentFrame, currentFinished := s.frame, s.finished
					s.mu.Unlock()
					s.update(currentFrame, currentFinished)
				}
			}
		}()
	}

	return &s
}

type Spinner struct {
	taskName string

	mu                     sync.Mutex
	frame, finished, total int
	ctx                    context.Context
	cancel                 context.CancelFunc
}

func (s *Spinner) Incr() {
	s.mu.Lock()
	s.finished++
	currentFrame, currentFinished := s.frame, s.finished
	shouldStop := (s.total > 0 && s.finished >= s.total)
	s.mu.Unlock()

	s.update(currentFrame, currentFinished)
	if shouldStop {
		s.Stop()
	}
}

func (s *Spinner) update(currentFrame, currentFinished int) {
	frameText := spinners[currentFrame%len(spinners)]
	if s.total != 0 && currentFinished >= s.total {
		frameText = "✅   "
	}

	progressText := ""
	if s.total > 0 {
		progressText = fmt.Sprintf("[%d/%d]", currentFinished, s.total)
	}

	if term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Printf("\r%s %s %s", s.taskName, progressText, frameText)
	} else {
		if s.total != 0 {
			fmt.Printf("%s %s\n", s.taskName, progressText)
		}
	}

}

func (s *Spinner) Stop() {
	s.mu.Lock() // Protect access to ctx.Done and cancel
	select {
	case <-s.ctx.Done():
		s.mu.Unlock()
		return // Already stopped
	default:
		s.cancel()
		// Capture final state for update *after* cancelling
		currentFrame, currentFinished := s.frame, s.finished
		s.mu.Unlock() // Unlock before potentially long-running I/O (fmt.Print)

		if term.IsTerminal(int(os.Stdout.Fd())) {
			s.update(currentFrame, currentFinished) // Show final state
			fmt.Println()                           // Move to next line
		} else {
			// For non-TTY, if it's a determinate spinner that was completed,
			// the last update in Incr would have printed the final count.
			// If it's an indeterminate spinner, or one stopped prematurely,
			// you might want a specific "Done" message.
			if s.total > 0 && currentFinished >= s.total {
				// Last update from Incr should have handled this. A newline is good.
				fmt.Println()
			} else if s.total == 0 {
				// Indeterminate spinner, print a completion message
				fmt.Printf("%s... Done.\n", s.taskName)
			} else {
				// Determinate but stopped early
				fmt.Printf("\r%s [%d/%d] Stopped.\n", s.taskName, currentFinished, s.total)
			}
		}
		return
	}
}
