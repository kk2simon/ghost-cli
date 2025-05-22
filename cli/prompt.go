package cli

import (
	"bufio" // Added
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

func ParsePromptFile(promptFile string) (string, error) {
	promptBytes, err := os.ReadFile(promptFile)
	if err != nil {
		return "", fmt.Errorf("Failed to read prompt: %v", err)
	}
	p := string(promptBytes)
	if !strings.Contains(p, "{{") { // no template placeholder
		return p, nil
	}

	// render template
	tmpl, err := template.New(promptFile).Parse(p)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", promptFile, err)
	}

	cwd, _ := os.Getwd()
	ign, err := ignore.CompileIgnoreFile(filepath.Join(cwd, ".gitignore"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			ign = nil
		}
	}

	treeWt := bytes.NewBuffer([]byte(cwd + "\n"))
	err = DirectoryTree(treeWt, cwd, cwd, ign, nil)
	data := map[string]any{
		"cwd":     cwd,
		"dirTree": treeWt.String(),
	}

	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	p = sb.String()
	return p, nil
}

// DirectoryTree recursively walks the directory tree rooted at 'current'
// while respecting patterns in .gitignore. It prints an ASCII tree
// similar to the Unix `tree` command.
func DirectoryTree(w io.Writer, repoRoot, current string, ign *ignore.GitIgnore, prefixMarkers []bool) error {
	entries, err := os.ReadDir(current)
	if err != nil {
		return fmt.Errorf("cannot read %s: %v", current, err)
	}

	// Apply .gitignore-based filtering
	var visible []os.DirEntry
	for _, e := range entries {
		relPath, _ := filepath.Rel(repoRoot, filepath.Join(current, e.Name()))

		// Skip .git directory explicitly
		if e.IsDir() && e.Name() == ".git" {
			continue
		}
		if ign != nil && ign.MatchesPath(relPath) {
			// If the path matches an ignore rule, omit it (and its children)
			continue
		}
		visible = append(visible, e)
	}

	for i, e := range visible {
		isLast := i == len(visible)-1

		// Build tree branch prefix
		var b strings.Builder
		for _, last := range prefixMarkers {
			if last {
				b.WriteString("    ")
			} else {
				b.WriteString("│   ")
			}
		}
		if isLast {
			b.WriteString("└── ")
		} else {
			b.WriteString("├── ")
		}
		fmt.Fprintln(w, b.String()+e.Name())

		if e.IsDir() {
			// Recurse with updated prefix state
			err = DirectoryTree(w, repoRoot, filepath.Join(current, e.Name()), ign, append(prefixMarkers, isLast))
			if err != nil {
				return fmt.Errorf("cannot print tree for %s: %v", e.Name(), err)
			}
		}
	}

	return nil
}

// ReadLineStdin prompts the user and reads a line of input from stdin.
// It returns the trimmed input string and any error encountered.
func ReadLineStdin() (string, error) {
	fmt.Print("> ")
	reader := bufio.NewReader(os.Stdin)
	userInput, err := reader.ReadString('\n')
	if err != nil {
		// Handle error, maybe return or continue
		fmt.Printf("Error reading user input: %v\n", err)
		return "", fmt.Errorf("error reading user input: %w", err)
	}

	return strings.TrimSpace(userInput), nil
}
