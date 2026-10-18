// Command complexity measures parsed Go functions without building the application.
package main

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"

	"github.com/fzipp/gocyclo"
)

type source struct {
	Path string
	Text string
}
type result struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	Line  int    `json:"line"`
	Score int    `json:"score"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var files []source
	if err := json.NewDecoder(os.Stdin).Decode(&files); err != nil {
		return err
	}
	out := []result{}
	for _, file := range files {
		fs := token.NewFileSet()
		tree, err := parser.ParseFile(fs, file.Path, file.Text, 0)
		if err != nil {
			return err
		}
		for _, stat := range gocyclo.AnalyzeASTFile(tree, fs, nil) {
			out = append(out, result{file.Path, stat.FuncName, stat.Pos.Line, stat.Complexity})
		}
	}
	return json.NewEncoder(os.Stdout).Encode(out)
}
