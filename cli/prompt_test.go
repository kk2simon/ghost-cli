package cli

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	ignore "github.com/sabhiram/go-gitignore"
)

func Test_DirectoryTree(t *testing.T) {
	root, _ := os.Getwd()

	absRoot, err := filepath.Abs(root)
	if err != nil {
		log.Fatal(err)
	}

	// Attempt to compile .gitignore rules (ignore error if file not present)
	ign, _ := ignore.CompileIgnoreFile(filepath.Join(absRoot, ".gitignore"))

	w := bytes.NewBuffer([]byte(absRoot))

	err = DirectoryTree(w, absRoot, absRoot, ign, nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(w.String())
}
