// Command toml-test-decoder reads TOML on stdin and writes toml-test's tagged
// JSON on stdout. Invalid TOML is reported on stderr with exit code 1.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/npikall/toml-edit/eval"
	"github.com/npikall/toml-edit/internal/tagged"
	"github.com/npikall/toml-edit/parser"
)

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(in io.Reader, out io.Writer) error {
	src, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	doc, err := parser.Parse(string(src))
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	table, err := eval.Evaluate(doc)
	if err != nil {
		return fmt.Errorf("evaluate: %w", err)
	}
	if err := json.NewEncoder(out).Encode(tagged.FromTable(table)); err != nil {
		return fmt.Errorf("write json: %w", err)
	}
	return nil
}
